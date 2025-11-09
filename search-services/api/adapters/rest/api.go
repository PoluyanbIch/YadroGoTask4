package rest

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"yadro.com/course/api/adapters/words"
	"yadro.com/course/api/config"
	"yadro.com/course/api/core"
)

type PingResponse struct {
	Replies map[string]string `json:"replies"`
}

func NewPingHandler(log *slog.Logger, pingers map[string]core.Pinger, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		replies := make(map[string]string)

		for name, pinger := range pingers {
			err := pinger.Ping(ctx)
			if err != nil {
				log.Warn("ping failed", "service", name, "error", err)
				replies[name] = "unavailable"
			} else {
				replies[name] = "ok"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		resp := PingResponse{Replies: replies}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to write ping response", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

func NewWordsHandler(log *slog.Logger, cfg config.Config, wordsClient *words.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			http.Error(w, "missing phrase", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), cfg.HTTPConfig.Timeout)
		defer cancel()

		wordsList, err := wordsClient.Norm(ctx, phrase)
		if err != nil {
			handleNormError(w, log, err)
			return
		}

		resp := map[string]interface{}{
			"words": wordsList,
			"total": len(wordsList),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to write response", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}
}

func handleNormError(w http.ResponseWriter, log *slog.Logger, err error) {
	switch err {
	case core.ErrBadArguments:
		http.Error(w, "phrase too large", http.StatusBadRequest)
	case core.ErrServiceUnavailable:
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	default:
		log.Error("failed to normalize phrase", "error", err)
		http.Error(w, "internal service error", http.StatusInternalServerError)
	}
}

func NewUpdateHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func NewUpdateStatsHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func NewUpdateStatusHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func NewDropHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}
