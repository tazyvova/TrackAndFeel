package gpx

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// gpxFile represents the minimal GPX XML model with Garmin HR extension support.
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

// parsedPt is the intermediate point representation used during import.
type parsedPt struct {
	t     time.Time
	lat   float64
	lon   float64
	ele   *float64
	hr    *int
	speed *float64
}

// Import reads a GPX file and stores its data into the database, returning the new activity ID.
func Import(ctx context.Context, pool *pgxpool.Pool, f multipart.File) (uuid.UUID, error) {
	raw, err := io.ReadAll(f)
	if err != nil {
		return uuid.Nil, fmt.Errorf("read: %w", err)
	}
	var g gpxFile
	if err := xml.Unmarshal(raw, &g); err != nil {
		return uuid.Nil, fmt.Errorf("xml unmarshal (raw size %d): %w", len(raw), err)
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
