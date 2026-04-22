package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleAnalyze_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/analyze", nil)
	w := httptest.NewRecorder()
	handleAnalyze(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleAnalyze_MissingURL(t *testing.T) {
	body := bytes.NewBufferString(`{"url":""}`)
	req := httptest.NewRequest(http.MethodPost, "/api/analyze", body)
	w := httptest.NewRecorder()
	handleAnalyze(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleAnalyze_BadJSON(t *testing.T) {
	body := bytes.NewBufferString(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/analyze", body)
	w := httptest.NewRecorder()
	handleAnalyze(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleAnalyze_InvalidURL(t *testing.T) {
	body := bytes.NewBufferString(`{"url":"://broken"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/analyze", body)
	w := httptest.NewRecorder()
	handleAnalyze(w, req)

	var resp AnalyzeResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Error == nil {
		t.Error("expected an error payload")
	}
}
