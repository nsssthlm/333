package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type Project struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

type Folder struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID *string `json:"parentId"`
}

type File struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Ext       string `json:"ext"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func handleListProjects(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	rows, err := db.QueryContext(r.Context(),
		`SELECT id, name, created_at FROM core_project ORDER BY name`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	projects := []Project{}
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		projects = append(projects, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

func handleListFolders(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	projectID := r.PathValue("projectId")
	if projectID == "" {
		http.Error(w, "missing projectId", http.StatusBadRequest)
		return
	}

	rows, err := db.QueryContext(r.Context(),
		`SELECT id, name, parent_id FROM arca_folder WHERE project_id = $1 ORDER BY name`, projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	folders := []Folder{}
	for rows.Next() {
		var f Folder
		if err := rows.Scan(&f.ID, &f.Name, &f.ParentID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		folders = append(folders, f)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(folders)
}

func handleListFiles(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	folderID := r.PathValue("folderId")
	if folderID == "" {
		http.Error(w, "missing folderId", http.StatusBadRequest)
		return
	}

	rows, err := db.QueryContext(r.Context(), `
		SELECT f.id, f.name, COALESCE(f.ext,''), COALESCE(fv.size, 0), f.created_at, f.updated_at
		FROM arca_file f
		JOIN arca_folder_file ff ON ff.file_id = f.id
		LEFT JOIN arca_file_version fv ON fv.file_id = f.id
		WHERE ff.folder_id = $1
		ORDER BY f.name`, folderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	files := []File{}
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.Name, &f.Ext, &f.Size, &f.CreatedAt, &f.UpdatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		files = append(files, f)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func handleGetProject(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	projectID := r.PathValue("projectId")
	var p Project
	err := db.QueryRowContext(r.Context(),
		`SELECT id, name, created_at FROM core_project WHERE id = $1`, projectID).
		Scan(&p.ID, &p.Name, &p.CreatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}
