package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// ProjectFile represents a file available for client-side viewing.
type ProjectFile struct {
	FileVersionID string `json:"fileVersionId"`
	FileName      string `json:"fileName"`
	FileExt       string `json:"fileExt"`
	FileSize      int64  `json:"fileSize"`
	CreatedAt     string `json:"createdAt"`
}

// handleListModels returns files for a project that can be loaded in the 3D viewer.
// IFC files are parsed client-side via web-ifc WASM — no server mapping needed.
func handleListModels(w http.ResponseWriter, r *http.Request, db *sql.DB, _ string) {
	projectID := r.PathValue("projectId")
	if projectID == "" {
		http.Error(w, "missing projectId", http.StatusBadRequest)
		return
	}

	rows, err := db.QueryContext(r.Context(), `
		SELECT fv.id, f.name, f.ext, fv.size, fv.created_at
		FROM arca_file_version fv
		JOIN arca_file f ON f.id = fv.file_id
		JOIN arca_folder_file ff ON ff.file_id = f.id
		JOIN arca_folder fo ON fo.id = ff.folder_id
		WHERE fo.project_id = $1
		ORDER BY fv.created_at DESC`, projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var files []ProjectFile
	for rows.Next() {
		var f ProjectFile
		if err := rows.Scan(&f.FileVersionID, &f.FileName, &f.FileExt, &f.FileSize, &f.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		files = append(files, f)
	}

	if files == nil {
		files = []ProjectFile{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}
