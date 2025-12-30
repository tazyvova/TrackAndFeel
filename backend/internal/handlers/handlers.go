package handlers

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

"trackandfeel/backend/internal/gpx"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Upload handles POST /api/upload.
func Upload(pool *pgxpool.Pool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		file, hdr, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing form field 'file'", http.StatusBadRequest)
			return
		}
		defer file.Close()

		ext := filepath.Ext(hdr.Filename)
		if ext != "" && ext != ".gpx" && ext != ".GPX" {
			http.Error(w, "only .gpx supported", http.StatusBadRequest)
			return
		}

		actID, err := gpx.Import(r.Context(), pool, file)
		if err != nil {
			http.Error(w, "failed to import gpx", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": actID.String()})
	})
}

// ListActivities handles GET /api/activities.
func ListActivities(pool *pgxpool.Pool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// pagination
		limit := 20
		offset := 0
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		if v := r.URL.Query().Get("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				offset = n
			}
		}

		rows, err := pool.Query(r.Context(), `
                        SELECT id, started_at, sport, duration_sec, distance_m, avg_hr, max_hr
                        FROM activities
                        ORDER BY started_at DESC
                        LIMIT $1 OFFSET $2
                `, limit, offset)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type activity struct {
			ID          uuid.UUID `json:"id"`
			StartedAt   time.Time `json:"started_at"`
			Sport       *string   `json:"sport,omitempty"`
			DurationSec *int      `json:"duration_sec,omitempty"`
			DistanceM   *int      `json:"distance_m,omitempty"`
			AvgHR       *int      `json:"avg_hr,omitempty"`
			MaxHR       *int      `json:"max_hr,omitempty"`
		}
		list := make([]activity, 0, limit)
		for rows.Next() {
			var a activity
			if err := rows.Scan(&a.ID, &a.StartedAt, &a.Sport, &a.DurationSec, &a.DistanceM, &a.AvgHR, &a.MaxHR); err != nil {
				http.Error(w, "scan error", http.StatusInternalServerError)
				return
			}
			list = append(list, a)
		}
		if rows.Err() != nil {
			http.Error(w, "rows error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"limit":  limit,
			"offset": offset,
			"items":  list,
		})
	})
}

// GetActivityTrack handles GET /api/activities/{id}/track.
func GetActivityTrack(pool *pgxpool.Pool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Expect path like /api/activities/{id}/track
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/activities/"), "/")
		if len(parts) != 2 || parts[1] != "track" {
			http.NotFound(w, r)
			return
		}
		idStr := parts[0]
		actID, err := uuid.Parse(idStr)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}

		// summary
		var (
			startedAt   time.Time
			sport       *string
			durationSec *int
			distanceM   *int
			avgHR       *int
			maxHR       *int
		)
		err = pool.QueryRow(r.Context(), `
                        SELECT started_at, sport, duration_sec, distance_m, avg_hr, max_hr
                        FROM activities WHERE id=$1
                `, actID).Scan(&startedAt, &sport, &durationSec, &distanceM, &avgHR, &maxHR)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}

		// points
		rows, err := pool.Query(r.Context(), `
                        SELECT
                          t,
                          ele_m,
                          hr,
                          speed_mps,
                          ST_X((geom::geometry)) AS lon,
                          ST_Y((geom::geometry)) AS lat
                        FROM trackpoints
                        WHERE activity_id=$1
                        ORDER BY t
                `, actID)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type point struct {
			T   time.Time
			Ele *float64
			HR  *int
			Spd *float64
			Lon float64
			Lat float64
		}
		points := make([]point, 0, 2048)
		for rows.Next() {
			var p point
			if err := rows.Scan(&p.T, &p.Ele, &p.HR, &p.Spd, &p.Lon, &p.Lat); err != nil {
				http.Error(w, "scan error", http.StatusInternalServerError)
				return
			}
			points = append(points, p)
		}
		if rows.Err() != nil {
			http.Error(w, "rows error", http.StatusInternalServerError)
			return
		}
		if len(points) == 0 {
			http.Error(w, "no trackpoints", http.StatusNotFound)
			return
		}

		// Build GeoJSON LineString + chart series
		coords := make([][2]float64, 0, len(points))
		timeISO := make([]string, 0, len(points))
		ele := make([]*float64, 0, len(points))
		hr := make([]*int, 0, len(points))
		spd := make([]*float64, 0, len(points))
		spdKMH := make([]*float64, 0, len(points))
		pace := make([]*float64, 0, len(points))
		elapsedSec := make([]float64, 0, len(points))

		baseline := points[0].T

		for _, p := range points {
			coords = append(coords, [2]float64{p.Lon, p.Lat})
			timeISO = append(timeISO, p.T.UTC().Format(time.RFC3339Nano))
			ele = append(ele, p.Ele)
			hr = append(hr, p.HR)
			spd = append(spd, p.Spd)
			spdKMH = append(spdKMH, toKMH(p.Spd))
			pace = append(pace, paceMinPerKm(p.Spd))
			elapsedSec = append(elapsedSec, p.T.Sub(baseline).Seconds())
		}

		resp := map[string]any{
			"id": actID.String(),
			"summary": map[string]any{
				"started_at":   startedAt,
				"sport":        sport,
				"duration_sec": durationSec,
				"distance_m":   distanceM,
				"avg_hr":       avgHR,
				"max_hr":       maxHR,
			},
			"geojson": map[string]any{
				"type": "Feature",
				"geometry": map[string]any{
					"type":        "LineString",
					"coordinates": coords,
				},
				"properties": map[string]any{},
			},
			"series": map[string]any{
				"time_iso":        timeISO,
				"elapsed_sec":     elapsedSec,
				"elevation":       ele,
				"hr":              hr,
				"speed_mps":       spd,
				"speed_kmh":       spdKMH,
				"pace_min_per_km": pace,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
}

func paceMinPerKm(speed *float64) *float64 {
	if speed == nil || *speed <= 0 {
		return nil
	}
	secPerKm := 1000.0 / *speed
	minutes := math.Floor(secPerKm / 60)
	seconds := math.Round(math.Mod(secPerKm, 60))
	v := minutes + seconds/100
	return &v
}

func toKMH(speed *float64) *float64 {
	if speed == nil {
		return nil
	}
	v := *speed * 3.6
	return &v
}
