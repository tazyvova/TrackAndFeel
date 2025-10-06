package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	appdb "TrackAndFeel/backend/internal/db"
	"TrackAndFeel/backend/internal/migrate"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------- GPX minimal XML model (with Garmin HR extension) ----------

type gpxFile struct {
	XMLName xml.Name `xml:"gpx"`
	Tracks  []gpxTrk `xml:"trk"`
}
type gpxTrk struct {
	Segments []gpxSeg `xml:"trkseg"`
}
type gpxSeg struct {
	Points []gpxPt `xml:"trkpt"`
}
type gpxPt struct {
	Lat  float64  `xml:"lat,attr"`
	Lon  float64  `xml:"lon,attr"`
	Ele  *float64 `xml:"ele"`
	Time string   `xml:"time"`
	Ext  *gpxExt  `xml:"extensions"`
}
type gpxExt struct {
	// Garmin TrackPointExtension namespace-aware tag
	TrackExt *gpxTpx `xml:"http://www.garmin.com/xmlschemas/TrackPointExtension/v1 TrackPointExtension"`
}
type gpxTpx struct {
	HR *int `xml:"hr"`
}

// Parsed point used by our pipeline
type parsedPt struct {
	t     time.Time
	lat   float64
	lon   float64
	ele   *float64
	hr    *int
	speed *float64
}

// ---------- main & server ----------

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
	mux.Handle("/api/upload", uploadHandler(pool))
	mux.Handle("/api/activities", activitiesHandler(pool))
	mux.Handle("/api/activities/", activityTrackHandler(pool)) // expects /api/activities/{id}/track

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

// ---------- upload handler ----------

func uploadHandler(pool *pgxpool.Pool) http.Handler {
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

		actID, err := importGPX(r.Context(), pool, file)
		if err != nil {
			log.Printf("upload error: %v", err)
			http.Error(w, "failed to import gpx", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": actID.String()})
	})
}

// ---------- GPX import ----------

func importGPX(ctx context.Context, pool *pgxpool.Pool, f multipart.File) (uuid.UUID, error) {
	raw, err := io.ReadAll(f)
	if err != nil {
		return uuid.Nil, fmt.Errorf("read: %w", err)
	}
	var g gpxFile
	if err := xml.Unmarshal(raw, &g); err != nil {
		return uuid.Nil, fmt.Errorf("xml unmarshal: %w", err)
	}

	// flatten points
	pts := make([]gpxPt, 0, 2048)
	for _, trk := range g.Tracks {
		for _, seg := range trk.Segments {
			pts = append(pts, seg.Points...)
		}
	}
	if len(pts) < 2 {
		return uuid.Nil, errors.New("not enough points")
	}

	series := make([]parsedPt, 0, len(pts))
	var (
		totalDistM          float64
		minLat, maxLat      = 90.0, -90.0
		minLon, maxLon      = 180.0, -180.0
		avgHR, maxHR, hrCnt int
	)
	var prev *parsedPt

	for i := range pts {
		p := &pts[i]

		tt, ok := parseTime(p.Time)
		if !ok {
			// skip points without a valid timestamp
			continue
		}

		if p.Lat < minLat {
			minLat = p.Lat
		}
		if p.Lat > maxLat {
			maxLat = p.Lat
		}
		if p.Lon < minLon {
			minLon = p.Lon
		}
		if p.Lon > maxLon {
			maxLon = p.Lon
		}

		var hr *int
		if p.Ext != nil && p.Ext.TrackExt != nil && p.Ext.TrackExt.HR != nil {
			hr = p.Ext.TrackExt.HR
			hrCnt++
			avgHR += *hr
			if *hr > maxHR {
				maxHR = *hr
			}
		}

		np := parsedPt{t: tt, lat: p.Lat, lon: p.Lon, ele: p.Ele, hr: hr}

		// speed from previous
		if prev != nil {
			d := haversineMeters(prev.lat, prev.lon, np.lat, np.lon)
			dt := np.t.Sub(prev.t).Seconds()
			if dt > 0 && d >= 0 {
				v := d / dt
				np.speed = &v
				totalDistM += d
			}
		}
		series = append(series, np)
		prev = &series[len(series)-1]
	}
	if len(series) < 2 {
		return uuid.Nil, errors.New("no timed points")
	}

	start := series[0].t
	end := series[len(series)-1].t
	durationSec := int(end.Sub(start).Seconds())
	if durationSec < 0 {
		durationSec = 0
	}
	if hrCnt > 0 {
		avgHR = int(math.Round(float64(avgHR) / float64(hrCnt)))
	}

	actID := uuid.New()

	// write in a transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	wkt := bboxToWKT(minLat, minLon, maxLat, maxLon)
	_, err = tx.Exec(ctx, `
		INSERT INTO activities (id, started_at, sport, duration_sec, distance_m, avg_hr, max_hr, bounds)
		VALUES ($1, $2, $3, $4, $5, $6, $7, ST_GeogFromText($8))
	`, actID, start, nil, durationSec, int(totalDistM), nullIntPtr(avgHR, hrCnt > 0), nullIntPtr(maxHR, hrCnt > 0), wkt)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert activity: %w", err)
	}

	var b pgx.Batch
	for _, s := range series {
		b.Queue(`
			INSERT INTO trackpoints (activity_id, t, ele_m, hr, speed_mps, geom)
			VALUES ($1, $2, $3, $4, $5, ST_SetSRID(ST_MakePoint($6, $7), 4326)::geography)
		`, actID, s.t, s.ele, s.hr, s.speed, s.lon, s.lat)
	}
	if err := tx.SendBatch(ctx, &b).Close(); err != nil {
		return uuid.Nil, fmt.Errorf("insert trackpoints: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return actID, nil
}

// ---------- helpers ----------

func parseTime(s string) (time.Time, bool) {
	// common GPX time formats
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",  // some devices
		"2006-01-02T15:04:05Z0700",  // with offset
		"2006-01-02 15:04:05Z07:00", // space + offset
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func bboxToWKT(minLat, minLon, maxLat, maxLon float64) string {
	// POLYGON in lon/lat order, closed ring
	return fmt.Sprintf(
		"POLYGON((%[1]f %[2]f,%[3]f %[2]f,%[3]f %[4]f,%[1]f %[4]f,%[1]f %[2]f))",
		minLon, minLat, maxLon, maxLat,
	)
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000.0
	rad := func(d float64) float64 { return d * math.Pi / 180.0 }
	dLat := rad(lat2 - lat1)
	dLon := rad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rad(lat1))*math.Cos(rad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func nullIntPtr(v int, ok bool) *int {
	if !ok {
		return nil
	}
	return &v
}

// ----------- GET /api/activities ---------------
func activitiesHandler(pool *pgxpool.Pool) http.Handler {
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

// -------- GET /api/activities/{id}/track --------
func activityTrackHandler(pool *pgxpool.Pool) http.Handler {
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

		for _, p := range points {
			coords = append(coords, [2]float64{p.Lon, p.Lat})
			timeISO = append(timeISO, p.T.UTC().Format(time.RFC3339Nano))
			ele = append(ele, p.Ele)
			hr = append(hr, p.HR)
			spd = append(spd, p.Spd)
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
				"time_iso":  timeISO,
				"elevation": ele,
				"hr":        hr,
				"speed_mps": spd,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
}
