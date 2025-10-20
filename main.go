package main

import (
	"io"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	http.HandleFunc("/webhook", webhookHandler)

	if err := http.ListenAndServe(getEnvOrDefault("PORT", "8080"), nil); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
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

	slog.Info("Received webhook request", "body", string(body))
	w.WriteHeader(http.StatusOK)
}

func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return ":" + defaultValue
	}
	return ":" + value
}
