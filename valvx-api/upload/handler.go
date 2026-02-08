// Package upload implements the TUS-based chunked upload engine.
//
// Uses tusd v2 with S3 store backend for streaming directly to MinIO.
// No temp files on disk — chunks go straight to S3 multipart upload.
//
// Features:
// - 5 MB chunk size for fast parallel transfer
// - Automatic resume on network failure
// - Post-upload hooks: create arca_file + arca_file_version, trigger Speckle IFC import
package upload

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"github.com/tus/tusd/v2/pkg/handler"
	"github.com/tus/tusd/v2/pkg/s3store"
)

// Config holds upload engine configuration.
type Config struct {
	MinioEndpoint  string
	MinioBucket    string
	MinioAccessKey string
	MinioSecretKey string
	MaxUploadSize  int64
	ChunkSize      int64
}

// Handler manages TUS uploads and post-upload processing.
type Handler struct {
	DB         *sql.DB
	Config     Config
	Bridge     *SpeckleBridge
	tusHandler *handler.Handler
}

// NewHandler creates a new upload handler with a real TUS server backed by S3/MinIO.
func NewHandler(db *sql.DB, cfg Config, bridge *SpeckleBridge) *Handler {
	h := &Handler{
		DB:     db,
		Config: cfg,
		Bridge: bridge,
	}

	awsConfig := &aws.Config{
		Region:           aws.String("us-east-1"),
		Endpoint:         aws.String(cfg.MinioEndpoint),
		S3ForcePathStyle: aws.Bool(true),
		Credentials:      credentials.NewStaticCredentials(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
	}

	sess, err := awssession.NewSession(awsConfig)
	if err != nil {
		log.Printf("Warning: could not create S3 session for TUS: %v", err)
		return h
	}

	s3Client := s3.New(sess)
	store := s3store.New(cfg.MinioBucket, s3Client)

	composer := handler.NewStoreComposer()
	store.UseIn(composer)

	tusHandler, err := handler.NewHandler(handler.Config{
		BasePath:                "/api/uploads/",
		StoreComposer:           composer,
		MaxSize:                 cfg.MaxUploadSize,
		NotifyCompleteUploads:   true,
		NotifyCreatedUploads:    true,
		RespectForwardedHeaders: true,
	})
	if err != nil {
		log.Printf("Warning: could not create TUS handler: %v", err)
		return h
	}

	h.tusHandler = tusHandler
	go h.processCompletedUploads()

	return h
}

// RegisterRoutes sets up the TUS upload endpoint.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	if h.tusHandler != nil {
		mux.Handle("POST /api/uploads/", http.StripPrefix("/api/uploads/", h.tusHandler))
		mux.Handle("HEAD /api/uploads/", http.StripPrefix("/api/uploads/", h.tusHandler))
		mux.Handle("PATCH /api/uploads/", http.StripPrefix("/api/uploads/", h.tusHandler))
		mux.Handle("DELETE /api/uploads/", http.StripPrefix("/api/uploads/", h.tusHandler))
		mux.Handle("POST /api/uploads", http.StripPrefix("/api/uploads", h.tusHandler))

		optionsHandler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Tus-Resumable", "1.0.0")
			w.Header().Set("Tus-Version", "1.0.0")
			w.Header().Set("Tus-Extension", "creation,creation-with-upload,termination")
			w.Header().Set("Tus-Max-Size", fmt.Sprintf("%d", h.Config.MaxUploadSize))
			w.WriteHeader(http.StatusNoContent)
		}
		mux.HandleFunc("OPTIONS /api/uploads", optionsHandler)
		mux.HandleFunc("OPTIONS /api/uploads/", optionsHandler)
	} else {
		unavailable := func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "upload service unavailable", http.StatusServiceUnavailable)
		}
		mux.HandleFunc("/api/uploads", unavailable)
		mux.HandleFunc("/api/uploads/", unavailable)
	}
}

func (h *Handler) processCompletedUploads() {
	if h.tusHandler == nil {
		return
	}
	for {
		select {
		case event := <-h.tusHandler.CompleteUploads:
			go h.onUploadComplete(event)
		case event := <-h.tusHandler.CreatedUploads:
			log.Printf("Upload created: %s (%d bytes)", event.Upload.ID, event.Upload.Size)
		}
	}
}

func (h *Handler) onUploadComplete(event handler.HookEvent) {
	info := event.Upload
	metadata := info.MetaData

	filename := metadata["filename"]
	folderId := metadata["folderId"]
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	if ext == "" {
		ext = metadata["ext"]
	}

	creatorID := metadata["creatorId"]
	if creatorID == "" {
		creatorID = metadata["creator_id"]
	}

	log.Printf("Upload complete: %s (%s, %d bytes, folder=%s)", filename, ext, info.Size, folderId)

	ctx := context.Background()
	fileID := uuid.New().String()
	fileVersionID := uuid.New().String()
	now := time.Now().UTC()

	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("Upload post-processing error (begin tx): %v", err)
		return
	}
	defer tx.Rollback()

	cleanName := strings.TrimSuffix(filename, "."+ext)
	_, err = tx.ExecContext(ctx,
		`INSERT INTO arca_file (id, created_at, updated_at, name, ext) VALUES ($1, $2, $3, $4, $5)`,
		fileID, now, now, cleanName, ext)
	if err != nil {
		log.Printf("Upload post-processing error (insert file): %v", err)
		return
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO arca_file_version (id, created_at, updated_at, number, size, file_id, creator_id)
		 VALUES ($1, $2, $3, 1, $4, $5, $6)`,
		fileVersionID, now, now, info.Size, fileID, creatorID)
	if err != nil {
		log.Printf("Upload post-processing error (insert file_version): %v", err)
		return
	}

	if folderId != "" && folderId != "root" {
		tx.ExecContext(ctx,
			`INSERT INTO arca_folder_file (folder_id, file_id) VALUES ($1, $2)`,
			folderId, fileID)
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Upload post-processing error (commit): %v", err)
		return
	}

	log.Printf("Created file record: %s (version %s)", fileID, fileVersionID)

	if isIFCFile(ext) && h.Bridge != nil {
		go func() {
			if err := h.Bridge.TriggerImport(context.Background(), fileVersionID, info.ID); err != nil {
				log.Printf("Speckle import trigger failed for %s: %v", fileVersionID, err)
			}
		}()
	}
}

func isIFCFile(ext string) bool {
	ext = strings.ToLower(ext)
	return ext == "ifc" || ext == "ifczip"
}

func parseTUSMetadata(header string) map[string]string {
	result := make(map[string]string)
	if header == "" {
		return result
	}
	pairs := strings.Split(header, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		parts := strings.SplitN(pair, " ", 2)
		if len(parts) == 2 {
			decoded, err := base64.StdEncoding.DecodeString(parts[1])
			if err == nil {
				result[parts[0]] = string(decoded)
			} else {
				result[parts[0]] = parts[1]
			}
		} else if len(parts) == 1 {
			result[parts[0]] = ""
		}
	}
	return result
}
