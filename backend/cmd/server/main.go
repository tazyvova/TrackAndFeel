package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	appdb "TrackAndFeel/backend/internal/db"
	"TrackAndFeel/backend/internal/handlers"
	"TrackAndFeel/backend/internal/migrate"
)

var Commit string

func main() {
	ctx := context.Background()

	cfg := appdb.FromEnv()
	pool, err := appdb.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if err := migrate.Apply(ctx, pool); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(Commit))
	})
	mux.Handle("/api/upload", handlers.Upload(pool))
	mux.Handle("/api/activities", handlers.ListActivities(pool))
	mux.Handle("/api/activities/", handlers.GetActivityTrack(pool)) // expects /api/activities/{id}/track

	port := getenv("PORT", "8080")
	srv := &http.Server{Addr: ":" + port, Handler: mux}

	// start
	go func() {
		log.Printf("backend listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
