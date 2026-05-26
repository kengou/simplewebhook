package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const maxBodyBytes = 1 << 20 // 1 MiB

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	webhookSecret := os.Getenv("WEBHOOK_SECRET")
	if webhookSecret == "" {
		slog.Warn("WEBHOOK_SECRET not set; webhook requests will not be authenticated")
	}

	logHeadersEnv := os.Getenv("LOG_HEADERS")
	logHeaders, err := strconv.ParseBool(logHeadersEnv)
	if err != nil && logHeadersEnv != "" {
		slog.Warn("Invalid LOG_HEADERS value; defaulting to false", "value", logHeadersEnv, "error", err)
	}
	if logHeaders {
		slog.Warn("LOG_HEADERS enabled; request headers (including any secrets) will be logged verbatim")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /webhook", makeWebhookHandler(webhookSecret, logHeaders))
	mux.HandleFunc("GET /healthz", healthCheckHandler)

	addr := "0.0.0.0:" + getEnvOrDefault("PORT", "8080")
	srv := &http.Server{
		Handler:      mux,
		Addr:         addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		slog.Info("Server starting to listen", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		stop()
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	case <-ctx.Done():
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := srv.Shutdown(shutdownCtx); err != nil {
		cancel()
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}
	cancel()
	slog.Info("Server shutdown gracefully")
}

var healthResponse = []byte("{\"alive\":true}\n")

func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(healthResponse) //nolint:errcheck
}

func makeWebhookHandler(secret string, logHeaders bool) http.HandlerFunc {
	secretBytes := []byte(secret)
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		defer r.Body.Close()

		body, err := io.ReadAll(r.Body)
		if err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			slog.Error("Failed to read request body", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if secret != "" && !validateHMAC(body, r.Header.Get("X-Hub-Signature-256"), secretBytes) {
			http.Error(w, "Forbidden: invalid signature", http.StatusForbidden)
			return
		}

		if !json.Valid(body) {
			http.Error(w, "Bad Request: Invalid JSON", http.StatusBadRequest)
			return
		}

		attrs := []any{"content_length", len(body), "body", json.RawMessage(body)}
		if logHeaders {
			attrs = append(attrs,
				"method", r.Method,
				"path", r.URL.Path,
				"query", r.URL.RawQuery,
				"host", r.Host,
				"remote_addr", r.RemoteAddr,
				"headers", r.Header,
			)
		}
		slog.Info("Received webhook request", attrs...)
		w.WriteHeader(http.StatusOK)
	}
}

func validateHMAC(body []byte, signature string, secretBytes []byte) bool {
	const prefix = "sha256="
	if len(signature) <= len(prefix) {
		return false
	}
	mac := hmac.New(sha256.New, secretBytes)
	mac.Write(body)
	expected := prefix + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expected))
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
