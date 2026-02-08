package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// SpeckleModel represents a 3D model ready for viewing.
type SpeckleModel struct {
	FileVersionID   string  `json:"fileVersionId"`
	FileName        string  `json:"fileName"`
	FileExt         string  `json:"fileExt"`
	FileSize        int64   `json:"fileSize"`
	SpeckleModelID  string  `json:"speckleModelId"`
	SpeckleObjectID *string `json:"speckleObjectId,omitempty"`
	Status          string  `json:"status"`
	CreatedAt       string  `json:"createdAt"`
}

// handleListModels returns models with ready Speckle mappings for a project.
func handleListModels(w http.ResponseWriter, r *http.Request, db *sql.DB, defaultSpeckleProject string) {
	projectID := r.PathValue("projectId")
	if projectID == "" {
		http.Error(w, "missing projectId", http.StatusBadRequest)
		return
	}

	rows, err := db.QueryContext(r.Context(), `
		SELECT fv.id, f.name, f.ext, fv.size, sm.speckle_model_id, sm.speckle_object_id, sm.status, fv.created_at
		FROM arca_file_version fv
		JOIN arca_file f ON f.id = fv.file_id
		JOIN arca_speckle_mapping sm ON sm.file_version_id = fv.id
		JOIN arca_folder_file ff ON ff.file_id = f.id
		JOIN arca_folder fo ON fo.id = ff.folder_id
		WHERE fo.project_id = $1
		ORDER BY fv.created_at DESC`, projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var models []SpeckleModel
	for rows.Next() {
		var m SpeckleModel
		if err := rows.Scan(&m.FileVersionID, &m.FileName, &m.FileExt, &m.FileSize,
			&m.SpeckleModelID, &m.SpeckleObjectID, &m.Status, &m.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		models = append(models, m)
	}

	if models == nil {
		models = []SpeckleModel{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}
