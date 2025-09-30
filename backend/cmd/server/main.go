package main

import (
  "encoding/json"
  "log"
  "net/http"
  "os"
)

type Activity struct {
  ID string `json:"id"`
}

func main() {
  mux := http.NewServeMux()

  mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte("ok"))
  })

  mux.HandleFunc("/api/activities", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode([]Activity{})
  })

  port := os.Getenv("PORT")
  if port == "" {
    port = "8080"
  }
  log.Printf("backend listening on :%s", port)
  log.Fatal(http.ListenAndServe(":"+port, mux))
}
