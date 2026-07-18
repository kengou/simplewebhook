package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func FuzzValidateHMAC(f *testing.F) {
	secret := []byte("test-secret")
	f.Add([]byte(`{"event":"push"}`), "sha256=abc123", secret)
	f.Add([]byte{}, "sha256=", secret)
	f.Add([]byte(`{}`), "", secret)

	f.Fuzz(func(t *testing.T, body []byte, sig string, secretBytes []byte) {
		validateHMAC(body, sig, secretBytes)
	})
}

func FuzzWebhookHandler(f *testing.F) {
	f.Add([]byte(`{"key":"value"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`not json`))
	f.Add([]byte{})

	handler := makeWebhookHandler("", false)
	f.Fuzz(func(t *testing.T, body []byte) {
		req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/webhook", bytes.NewReader(body))
		if err != nil {
			return
		}
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	})
}

type mockReadCloser struct{}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated read error")
}

func (m *mockReadCloser) Close() error {
	return nil
}

func computeTestHMAC(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestHealthCheckHandler(t *testing.T) {
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/healthz", http.NoBody)
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
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/webhook", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	makeWebhookHandler("", false).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestWebhookHandler_ValidYAML(t *testing.T) {
	for _, ct := range []string{"application/yaml", "application/x-yaml"} {
		t.Run(ct, func(t *testing.T) {
			var buf bytes.Buffer
			prev := slog.Default()
			slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
			defer slog.SetDefault(prev)

			payload := "key: value\nnumber: 123\n"
			req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/webhook", strings.NewReader(payload))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", ct)

			rr := httptest.NewRecorder()
			makeWebhookHandler("", false).ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
			}

			if !strings.Contains(buf.String(), `"body":{"key":"value","number":123}`) {
				t.Errorf("expected YAML body logged as structured JSON; got: %s", buf.String())
			}
		})
	}
}

func TestWebhookHandler_InvalidYAML(t *testing.T) {
	payload := "key: value\n  bad indentation: [unclosed\n"
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/webhook", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/yaml")

	rr := httptest.NewRecorder()
	makeWebhookHandler("", false).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	if !strings.Contains(rr.Body.String(), "Bad Request: Invalid YAML") {
		t.Errorf("handler returned unexpected body: got %v", rr.Body.String())
	}
}

func TestWebhookHandler_UnsupportedContentType(t *testing.T) {
	payload := "hello world"
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/webhook", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "text/plain")

	rr := httptest.NewRecorder()
	makeWebhookHandler("", false).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnsupportedMediaType {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnsupportedMediaType)
	}
}

func TestWebhookHandler_InvalidContentType(t *testing.T) {
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/webhook", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "not/a/valid/content-type///")

	rr := httptest.NewRecorder()
	makeWebhookHandler("", false).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestWebhookHandler_InvalidJSON(t *testing.T) {
	payload := `{"key":"value"` // Malformed JSON
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/webhook", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	makeWebhookHandler("", false).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}

	if !strings.Contains(rr.Body.String(), "Bad Request: Invalid JSON") {
		t.Errorf("handler returned unexpected body: got %v", rr.Body.String())
	}
}

func TestWebhookHandler_BodyReadError(t *testing.T) {
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/webhook", &mockReadCloser{})
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	makeWebhookHandler("", false).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusInternalServerError)
	}

	if !strings.Contains(rr.Body.String(), "Internal Server Error") {
		t.Errorf("handler returned unexpected body: got %v", rr.Body.String())
	}
}

func TestWebhookHandler_BodyTooLarge(t *testing.T) {
	largeBody := bytes.Repeat([]byte("a"), maxBodyBytes+1)
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/webhook", bytes.NewReader(largeBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	makeWebhookHandler("", false).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusRequestEntityTooLarge {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusRequestEntityTooLarge)
	}
}

func TestWebhookHandler_ValidSignature(t *testing.T) {
	secret := "test-secret"
	payload := []byte(`{"event":"push"}`)
	sig := computeTestHMAC(payload, secret)

	req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/webhook", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Hub-Signature-256", sig)

	rr := httptest.NewRecorder()
	makeWebhookHandler(secret, false).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestWebhookHandler_InvalidSignature(t *testing.T) {
	secret := "test-secret"
	payload := []byte(`{"event":"push"}`)

	req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/webhook", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Hub-Signature-256", "sha256=invalidsignature")

	rr := httptest.NewRecorder()
	makeWebhookHandler(secret, false).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusForbidden {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusForbidden)
	}
}

func TestWebhookHandler_MissingSignature(t *testing.T) {
	payload := []byte(`{"event":"push"}`)
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/webhook", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	makeWebhookHandler("required-secret", false).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusForbidden {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusForbidden)
	}
}

func TestWebhookHandler_LogHeaders(t *testing.T) {
	payload := `{"event":"push"}`

	t.Run("logs headers and metadata when enabled", func(t *testing.T) {
		var buf bytes.Buffer
		prev := slog.Default()
		slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
		defer slog.SetDefault(prev)

		req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/webhook", bytes.NewBufferString(payload))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Custom-Header", "custom-value")

		rr := httptest.NewRecorder()
		makeWebhookHandler("", true).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		out := buf.String()
		for _, want := range []string{"X-Custom-Header", "custom-value", `"method":"POST"`, `"headers":`, "remote_addr"} {
			if !strings.Contains(out, want) {
				t.Errorf("log output missing %q; got: %s", want, out)
			}
		}
	})

	t.Run("omits headers when disabled", func(t *testing.T) {
		var buf bytes.Buffer
		prev := slog.Default()
		slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
		defer slog.SetDefault(prev)

		req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/webhook", bytes.NewBufferString(payload))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Custom-Header", "custom-value")

		rr := httptest.NewRecorder()
		makeWebhookHandler("", false).ServeHTTP(rr, req)

		out := buf.String()
		for _, notWant := range []string{"X-Custom-Header", `"headers":`, "remote_addr"} {
			if strings.Contains(out, notWant) {
				t.Errorf("log output unexpectedly contains %q; got: %s", notWant, out)
			}
		}
	})
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		set          bool
		defaultValue string
		expected     string
	}{
		{
			name:         "should return env variable value when set",
			key:          "TEST_ENV_VAR",
			value:        "test_value",
			set:          true,
			defaultValue: "default",
			expected:     "test_value",
		},
		{
			name:         "should return default value when env variable is not set",
			key:          "TEST_ENV_VAR_UNSET",
			defaultValue: "default_value",
			expected:     "default_value",
		},
		{
			name:         "should return default value when env variable is an empty string",
			key:          "TEST_ENV_VAR_EMPTY",
			value:        "",
			set:          true,
			defaultValue: "default_for_empty",
			expected:     "default_for_empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.set {
				t.Setenv(tt.key, tt.value)
			}

			result := getEnvOrDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvOrDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}
