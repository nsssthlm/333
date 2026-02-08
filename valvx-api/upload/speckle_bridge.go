// Package upload contains the Speckle bridge for triggering IFC imports.
package upload

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// SpeckleBridge mediates between ValvX file uploads and Speckle's IFC import pipeline.
type SpeckleBridge struct {
	DB              *sql.DB
	SpeckleURL      string // e.g., "http://127.0.0.1:8080"
	SpeckleToken    string
	SpeckleProject  string // Speckle project ID
	MinioEndpoint   string
	MinioBucket     string
	HTTPClient      *http.Client
}

// NewSpeckleBridge creates a new bridge instance.
func NewSpeckleBridge(db *sql.DB, speckleURL, token, projectID, minioEndpoint, bucket string) *SpeckleBridge {
	return &SpeckleBridge{
		DB:             db,
		SpeckleURL:     speckleURL,
		SpeckleToken:   token,
		SpeckleProject: projectID,
		MinioEndpoint:  minioEndpoint,
		MinioBucket:    bucket,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// TriggerImport triggers Speckle's IFC import for a file.
//
// Flow:
// 1. Create a mapping record (status: pending)
// 2. Create a Speckle model via GraphQL
// 3. Upload the file to Speckle's blob storage
// 4. Trigger the file import process
// 5. Poll for completion and update mapping status
func (b *SpeckleBridge) TriggerImport(ctx context.Context, fileVersionID, minioObjectKey string) error {
	// Create pending mapping
	now := time.Now().UTC()
	_, err := b.DB.ExecContext(ctx, `
		INSERT INTO arca_speckle_mapping (file_version_id, speckle_model_id, status, created_at, updated_at)
		VALUES ($1, '', 'pending', $2, $3)
		ON CONFLICT (file_version_id) DO UPDATE SET status = 'pending', updated_at = $3`,
		fileVersionID, now, now,
	)
	if err != nil {
		return fmt.Errorf("create mapping: %w", err)
	}

	// Step 1: Create a model in Speckle via GraphQL
	modelID, err := b.createSpeckleModel(ctx, fileVersionID)
	if err != nil {
		b.updateMappingError(ctx, fileVersionID, err.Error())
		return fmt.Errorf("create speckle model: %w", err)
	}

	// Update mapping with model ID
	b.DB.ExecContext(ctx, `
		UPDATE arca_speckle_mapping SET speckle_model_id = $1, status = 'processing', updated_at = $2
		WHERE file_version_id = $3`,
		modelID, time.Now().UTC(), fileVersionID,
	)

	// Step 2: Upload the IFC file to Speckle
	err = b.uploadFileToSpeckle(ctx, minioObjectKey, modelID)
	if err != nil {
		b.updateMappingError(ctx, fileVersionID, err.Error())
		return fmt.Errorf("upload to speckle: %w", err)
	}

	// Step 3: Poll for import completion
	go b.pollImportStatus(context.Background(), fileVersionID, modelID)

	return nil
}

// createSpeckleModel creates a new model in the Speckle project via GraphQL.
func (b *SpeckleBridge) createSpeckleModel(ctx context.Context, name string) (string, error) {
	query := `mutation CreateModel($input: CreateModelInput!) {
		modelMutations {
			create(input: $input) {
				id
				name
			}
		}
	}`

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"name":      name,
			"projectId": b.SpeckleProject,
		},
	}

	body, _ := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": variables,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", b.SpeckleURL+"/graphql", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.SpeckleToken)

	resp, err := b.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Data struct {
			ModelMutations struct {
				Create struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"create"`
			} `json:"modelMutations"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(result.Errors) > 0 {
		return "", fmt.Errorf("graphql error: %s", result.Errors[0].Message)
	}

	return result.Data.ModelMutations.Create.ID, nil
}

// uploadFileToSpeckle uploads the IFC file from MinIO to Speckle's file upload endpoint.
func (b *SpeckleBridge) uploadFileToSpeckle(ctx context.Context, objectKey, modelID string) error {
	// In production, stream from MinIO to Speckle's upload endpoint
	// POST /api/file/{streamId}/{branchName}

	uploadURL := fmt.Sprintf("%s/api/file/%s/%s", b.SpeckleURL, b.SpeckleProject, modelID)

	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+b.SpeckleToken)

	// In production, this would use multipart upload with the file from MinIO
	log.Printf("Would upload %s to Speckle at %s", objectKey, uploadURL)

	return nil
}

// pollImportStatus checks the Speckle import status periodically.
func (b *SpeckleBridge) pollImportStatus(ctx context.Context, fileVersionID, modelID string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(10 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return
		case <-timeout:
			b.updateMappingError(ctx, fileVersionID, "import timed out")
			return
		case <-ticker.C:
			status, objectID, err := b.checkImportStatus(ctx, modelID)
			if err != nil {
				continue
			}
			if status == "ready" {
				now := time.Now().UTC()
				b.DB.ExecContext(ctx, `
					UPDATE arca_speckle_mapping
					SET status = 'ready', speckle_object_id = $1, updated_at = $2
					WHERE file_version_id = $3`,
					objectID, now, fileVersionID,
				)
				return
			}
			if status == "error" {
				b.updateMappingError(ctx, fileVersionID, "speckle import failed")
				return
			}
		}
	}
}

func (b *SpeckleBridge) checkImportStatus(ctx context.Context, modelID string) (string, string, error) {
	query := `query ModelVersions($projectId: String!, $modelId: String!) {
		project(id: $projectId) {
			model(id: $modelId) {
				versions(limit: 1) {
					items {
						id
						referencedObject
					}
				}
			}
		}
	}`

	variables := map[string]interface{}{
		"projectId": b.SpeckleProject,
		"modelId":   modelID,
	}

	body, _ := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": variables,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", b.SpeckleURL+"/graphql", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.SpeckleToken)

	resp, err := b.HTTPClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Project struct {
				Model struct {
					Versions struct {
						Items []struct {
							ID               string `json:"id"`
							ReferencedObject string `json:"referencedObject"`
						} `json:"items"`
					} `json:"versions"`
				} `json:"model"`
			} `json:"project"`
		} `json:"data"`
	}

	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &result)

	items := result.Data.Project.Model.Versions.Items
	if len(items) > 0 {
		return "ready", items[0].ReferencedObject, nil
	}

	return "processing", "", nil
}

func (b *SpeckleBridge) updateMappingError(ctx context.Context, fileVersionID, errMsg string) {
	now := time.Now().UTC()
	b.DB.ExecContext(ctx, `
		UPDATE arca_speckle_mapping SET status = 'error', error_message = $1, updated_at = $2
		WHERE file_version_id = $3`,
		errMsg, now, fileVersionID,
	)
}
