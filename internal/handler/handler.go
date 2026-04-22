package handler

// HTTP layer: POST /api/analyze
// Package handler wires HTTP routes to the analyzer

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/yourusername/page-insight-tool/internal/analyzer"
)

// AnalyzeRequest is the expected JSON body for POST /api/analyze.
type AnalyzeRequest struct {
	URL string `json:"url"`
}

// AnalyzeResponse wraps either a successful result or an error payload.
type AnalyzeResponse struct {
	Data  *analyzer.Result `json:"data,omitempty"`
	Error *ErrorPayload    `json:"error,omitempty"`
}

// ErrorPayload carries structured error details to the frontend.
type ErrorPayload struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

// New returns an http.Handler that serves the API and the embedded SPA.
func New(staticFS http.FileSystem) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/analyze", handleAnalyze)
	mux.Handle("/", http.FileServer(staticFS))
	return mux
}

// handleAnalyze accepts POST /api/analyze with a JSON body {"url": "..."}.
func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, 405, "only POST is accepted")
		return
	}

	var req AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.URL) == "" {
		writeError(w, http.StatusBadRequest, 400, "request body must be JSON: {\"url\": \"https://...\"}")
		return
	}

	log.Printf("Analyzing URL: %s", req.URL)

	result, err := analyzer.Analyze(req.URL)
	if err != nil {
		if ae, ok := err.(*analyzer.AnalyzeError); ok {
			writeError(w, http.StatusUnprocessableEntity, ae.StatusCode, ae.Message)
			return
		}
		writeError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AnalyzeResponse{Data: result})
}

func writeError(w http.ResponseWriter, httpStatus, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(AnalyzeResponse{
		Error: &ErrorPayload{StatusCode: code, Message: message},
	})
}
