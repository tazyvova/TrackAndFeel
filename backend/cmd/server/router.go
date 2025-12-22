package main

import (
	"net/http"

	"TrackAndFeel/backend/internal/handlers"
	"github.com/jackc/pgx/v5/pgxpool"
)

func buildRouter(pool *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(Commit))
	})
	mux.Handle("/api/upload", handlers.Upload(pool))
	mux.Handle("/api/activities", handlers.ListActivities(pool))
	mux.Handle("/api/activities/", handlers.GetActivityTrack(pool)) // expects /api/activities/{id}/track

	return mux
}
