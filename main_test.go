package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type errorBody struct{}

func (e *errorBody) Read(p []byte) (int, error) { return 0, errors.New("read error") }
func (e *errorBody) Close() error               { return nil }

func TestWebhookHandler_OKWithBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("hello"))
	rec := httptest.NewRecorder()
	webhookHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d got %d", http.StatusOK, rec.Code)
	}
}

func TestWebhookHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	rec := httptest.NewRecorder()
	webhookHandler(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d got %d", http.StatusMethodNotAllowed, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Only POST method is supported") {
		t.Fatalf("expected response body to contain method not allowed message")
	}
}

func TestWebhookHandler_InternalServerErrorOnReadError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Body = &errorBody{}
	rec := httptest.NewRecorder()
	webhookHandler(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d got %d", http.StatusInternalServerError, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Internal Server Error") {
		t.Fatalf("expected response body to contain internal server error message")
	}
}

func TestWebhookHandler_OKWithEmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(""))
	rec := httptest.NewRecorder()
	webhookHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d got %d", http.StatusOK, rec.Code)
	}
}
