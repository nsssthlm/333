// Package speckle implements a proxy layer between the ValvX frontend
// and the Speckle backend. This keeps the Speckle API token server-side
// and avoids CORS issues.
package speckle

import (
	"io"
	"net/http"
	"strings"
	"time"
)

// Proxy forwards requests from the ValvX frontend to the Speckle server,
// adding authentication headers.
type Proxy struct {
	SpeckleInternalURL string // e.g., "http://127.0.0.1:8080"
	SpeckleToken       string
	AllowedOrigin      string // e.g., "https://app.valvx.se"
	Client             *http.Client
}

// NewProxy creates a new Speckle proxy.
func NewProxy(internalURL, token, allowedOrigin string) *Proxy {
	return &Proxy{
		SpeckleInternalURL: strings.TrimRight(internalURL, "/"),
		SpeckleToken:       token,
		AllowedOrigin:      allowedOrigin,
		Client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// RegisterRoutes sets up proxy endpoints.
func (p *Proxy) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/speckle/graphql", p.ProxyGraphQL)
	mux.HandleFunc("/api/speckle/objects/", p.ProxyObjects)
	mux.HandleFunc("/api/projects/{projectId}/models", p.ListModels)
}

// ProxyGraphQL forwards GraphQL requests to Speckle.
func (p *Proxy) ProxyGraphQL(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		p.writeCORS(w)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Forward to Speckle
	proxyReq, err := http.NewRequestWithContext(
		r.Context(),
		"POST",
		p.SpeckleInternalURL+"/graphql",
		r.Body,
	)
	if err != nil {
		http.Error(w, "proxy error", http.StatusInternalServerError)
		return
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+p.SpeckleToken)

	resp, err := p.Client.Do(proxyReq)
	if err != nil {
		http.Error(w, "speckle unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	p.writeCORS(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// ProxyObjects forwards object data requests to Speckle.
// Path: /api/speckle/objects/{streamId}/{objectId}
func (p *Proxy) ProxyObjects(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		p.writeCORS(w)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Extract the Speckle path after /api/speckle/objects/
	path := strings.TrimPrefix(r.URL.Path, "/api/speckle/objects/")

	proxyReq, err := http.NewRequestWithContext(
		r.Context(),
		r.Method,
		p.SpeckleInternalURL+"/objects/"+path,
		r.Body,
	)
	if err != nil {
		http.Error(w, "proxy error", http.StatusInternalServerError)
		return
	}

	proxyReq.Header.Set("Authorization", "Bearer "+p.SpeckleToken)
	proxyReq.Header.Set("Accept", r.Header.Get("Accept"))

	resp, err := p.Client.Do(proxyReq)
	if err != nil {
		http.Error(w, "speckle unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	p.writeCORS(w)
	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// ListModels returns the available Speckle models for a ValvX project,
// by querying the arca_speckle_mapping table.
func (p *Proxy) ListModels(w http.ResponseWriter, r *http.Request) {
	// This endpoint is handled by the main API which has DB access.
	// The proxy just provides the route registration.
	// In production, this would query:
	//   SELECT fv.id, f.name, f.ext, sm.speckle_model_id, sm.speckle_object_id, sm.status
	//   FROM arca_file_version fv
	//   JOIN arca_file f ON f.id = fv.file_id
	//   JOIN arca_speckle_mapping sm ON sm.file_version_id = fv.id
	//   JOIN arca_folder_file ff ON ff.file_id = f.id
	//   JOIN arca_folder fo ON fo.id = ff.folder_id
	//   WHERE fo.project_id = $1 AND sm.status = 'ready'

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("[]"))
}

func (p *Proxy) writeCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", p.AllowedOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Max-Age", "86400")
}
