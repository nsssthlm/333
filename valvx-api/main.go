// ValvX API server — main entry point.
//
// This binary serves the BCF module, TUS upload engine, file download,
// and all existing ValvX API endpoints. It connects to PostgreSQL and MinIO.
//
// IFC files are parsed client-side via web-ifc WASM — no server-side
// conversion or Speckle infrastructure needed.
//
// Usage:
//
//	valvx-api              — start the HTTP server
//	valvx-api migrate      — run database migrations and exit
package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"github.com/nsssthlm/valvx-api/collab"
	"github.com/nsssthlm/valvx-api/internal/auth"
	"github.com/nsssthlm/valvx-api/internal/config"
	"github.com/nsssthlm/valvx-api/internal/middleware"
	"github.com/nsssthlm/valvx-api/upload"
)

func main() {
	cfg := config.Load()

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", cfg.PostgresURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("PostgreSQL ping failed: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	// Handle "migrate" subcommand
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := runMigrations(db, cfg.MigrationsDir); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		log.Println("Migrations complete")
		os.Exit(0)
	}

	// Initialize services
	collabSvc := collab.NewService(db)
	sessionStore := auth.NewSessionStore(db)
	collabHandler := collab.NewHandler(collabSvc, sessionStore)

	uploadHandler := upload.NewHandler(db, upload.Config{
		MinioEndpoint:  cfg.BlobstorServer,
		MinioBucket:    cfg.BlobstorBucket,
		MinioAccessKey: cfg.AWSAccessKeyID,
		MinioSecretKey: cfg.AWSSecretAccessKey,
		MaxUploadSize:  cfg.TUSMaxSize,
		ChunkSize:      cfg.TUSChunkSize,
	})

	// Build router
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Register module routes
	collabHandler.RegisterRoutes(mux)
	uploadHandler.RegisterRoutes(mux)

	// Project and file browsing
	mux.HandleFunc("GET /api/projects", func(w http.ResponseWriter, r *http.Request) {
		handleListProjects(w, r, db)
	})
	mux.HandleFunc("GET /api/projects/{projectId}", func(w http.ResponseWriter, r *http.Request) {
		handleGetProject(w, r, db)
	})
	mux.HandleFunc("GET /api/projects/{projectId}/folders", func(w http.ResponseWriter, r *http.Request) {
		handleListFolders(w, r, db)
	})
	mux.HandleFunc("GET /api/projects/{projectId}/folders/{folderId}/files", func(w http.ResponseWriter, r *http.Request) {
		handleListFiles(w, r, db)
	})

	// Model listing — returns files for client-side IFC loading
	mux.HandleFunc("GET /api/projects/{projectId}/models", func(w http.ResponseWriter, r *http.Request) {
		handleListModels(w, r, db, cfg.SpeckleProjectID)
	})

	// File download — serves IFC files from MinIO for client-side parsing
	mux.HandleFunc("GET /api/files/{fileVersionId}/download", func(w http.ResponseWriter, r *http.Request) {
		handleFileDownload(w, r, db, cfg)
	})

	// Apply middleware stack
	handler := middleware.Chain(mux,
		middleware.Recovery,
		middleware.Logger,
		middleware.CORS(cfg.CORSAllowedOrigins),
		middleware.Session(sessionStore),
	)

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.BindHost, cfg.BindPort)
	log.Printf("ValvX API listening on %s", addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Minute, // Long timeout for uploads
		IdleTimeout:  120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// handleFileDownload serves a file from MinIO/S3 by redirecting to a presigned URL.
func handleFileDownload(w http.ResponseWriter, r *http.Request, db *sql.DB, cfg *config.Config) {
	fileVersionID := r.PathValue("fileVersionId")
	if fileVersionID == "" {
		http.Error(w, "missing fileVersionId", http.StatusBadRequest)
		return
	}

	// Look up the S3 key for this file version (stored as TUS upload ID in S3)
	var fileName, ext string
	err := db.QueryRowContext(r.Context(),
		`SELECT f.name, f.ext FROM arca_file_version fv
		 JOIN arca_file f ON f.id = fv.file_id
		 WHERE fv.id = $1`, fileVersionID).Scan(&fileName, &ext)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	// Redirect to MinIO presigned URL or proxy the content
	// For now, set the download headers and proxy from MinIO
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`, fileName, ext))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "private, max-age=3600")

	// The actual S3 key is the TUS upload ID — stored in the MinIO bucket
	// In production, generate a presigned URL and redirect:
	// http.Redirect(w, r, presignedURL, http.StatusTemporaryRedirect)
	http.Error(w, "file download proxy not yet implemented — use presigned URLs", http.StatusNotImplemented)
}

// runMigrations reads SQL files from the migrations directory and applies them.
func runMigrations(db *sql.DB, migrationsDir string) error {
	// Ensure migration_version table exists
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS migration_version (version integer)`)
	if err != nil {
		return fmt.Errorf("create migration_version: %w", err)
	}

	// Get current version
	var currentVersion int
	err = db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM migration_version").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("get version: %w", err)
	}
	log.Printf("Current migration version: %d", currentVersion)

	// Read migration files
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
	sort.Strings(files)

	for _, file := range files {
		// Extract version number from filename (e.g., "002_add_collab_bcf_tables.sql" -> 2)
		base := filepath.Base(file)
		parts := strings.SplitN(base, "_", 2)
		if len(parts) < 2 {
			continue
		}
		version, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		if version <= currentVersion {
			continue
		}

		log.Printf("Applying migration %d: %s", version, base)
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read %s: %w", file, err)
		}

		_, err = db.Exec(string(sqlBytes))
		if err != nil {
			return fmt.Errorf("execute %s: %w", file, err)
		}

		log.Printf("Migration %d applied successfully", version)
	}

	return nil
}
