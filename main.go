package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	router := mux.NewRouter()

	router.HandleFunc("/webhook", webhookHandler).Methods(http.MethodPost)
	router.HandleFunc("/healthz", HealthCheckHandler).Methods(http.MethodGet)

	srv := &http.Server{
		Handler:      router,
		Addr:         "0.0.0.0:" + getEnvOrDefault("PORT", "8080"),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}

	slog.Info("Server starting to listen", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]bool{"alive": true}) //nolint:errcheck
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is supported", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Failed to read request body", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	jsonMap := make(map[string]interface{})
	if err = json.Unmarshal(body, &jsonMap); err != nil {
		slog.Error("Failed to parse JSON", "error", err)
		http.Error(w, "Bad Request: Invalid JSON", http.StatusBadRequest)
		return
	}

	slog.Info("Received webhook request", "body", jsonMap)
	w.WriteHeader(http.StatusOK)
}

func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
