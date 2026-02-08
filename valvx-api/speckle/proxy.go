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


func (p *Proxy) writeCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", p.AllowedOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Max-Age", "86400")
}
