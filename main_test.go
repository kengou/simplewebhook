package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type mockReadCloser struct{}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated read error")
}

func (m *mockReadCloser) Close() error {
	return nil
}

func TestHealthCheckHandler(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthCheckHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := `{"alive":true}`
	if strings.TrimSpace(rr.Body.String()) != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}

	if ctype := rr.Header().Get("Content-Type"); ctype != "application/json" {
		t.Errorf("handler returned wrong content type: got %v want %v",
			ctype, "application/json")
	}
}

func TestWebhookHandler_ValidJSON(t *testing.T) {
	payload := `{"key":"value","number":123}`
	req, err := http.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(webhookHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestWebhookHandler_InvalidJSON(t *testing.T) {
	payload := `{"key":"value"` // Malformed JSON
	req, err := http.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(webhookHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}

	expectedBody := "Bad Request: Invalid JSON"
	if !strings.Contains(rr.Body.String(), expectedBody) {
		t.Errorf("handler returned unexpected body: got %v want body to contain %v",
			rr.Body.String(), expectedBody)
	}
}

func TestWebhookHandler_BodyReadError(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/webhook", &mockReadCloser{})
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(webhookHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusInternalServerError)
	}

	expectedBody := "Internal Server Error"
	if !strings.Contains(rr.Body.String(), expectedBody) {
		t.Errorf("handler returned unexpected body: got %v want body to contain %v",
			rr.Body.String(), expectedBody)
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue string
		expected     string
		setup        func(key, value string)
		teardown     func(key string)
	}{
		{
			name:         "should return env variable value when set",
			key:          "TEST_ENV_VAR",
			value:        "test_value",
			defaultValue: "default",
			expected:     "test_value",
			setup: func(key, value string) {
				os.Setenv(key, value)
			},
			teardown: func(key string) {
				os.Unsetenv(key)
			},
		},
		{
			name:         "should return default value when env variable is not set",
			key:          "TEST_ENV_VAR_UNSET",
			value:        "",
			defaultValue: "default_value",
			expected:     "default_value",
			setup:        func(key, value string) {},
			teardown:     func(key string) {},
		},
		{
			name:         "should return default value when env variable is an empty string",
			key:          "TEST_ENV_VAR_EMPTY",
			value:        "",
			defaultValue: "default_for_empty",
			expected:     "default_for_empty",
			setup: func(key, value string) {
				os.Setenv(key, value)
			},
			teardown: func(key string) {
				os.Unsetenv(key)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(tt.key, tt.value)
			defer tt.teardown(tt.key)

			result := getEnvOrDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvOrDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}
