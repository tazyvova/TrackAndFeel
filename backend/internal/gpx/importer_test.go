package gpx

import (
	"math"
	"testing"
	"time"
)

func TestParseTime(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		expect bool
	}{
		{"rfc3339nano", "2020-01-02T15:04:05.123Z", true},
		{"rfc3339", "2020-01-02T15:04:05Z", true},
		{"customMillis", "2020-01-02T15:04:05.000Z", true},
		{"withOffset", "2020-01-02T15:04:05-0700", true},
		{"spaceOffset", "2020-01-02 15:04:05+07:00", true},
		{"invalid", "not-a-time", false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, ok := parseTime(tc.input)
			if ok != tc.expect {
				t.Fatalf("parseTime(%q) ok=%v want %v", tc.input, ok, tc.expect)
			}
		})
	}
}

func TestBBoxToWKT(t *testing.T) {
	got := bboxToWKT(10, 20, 30, 40)
	want := "POLYGON((20.000000 10.000000,40.000000 10.000000,40.000000 30.000000,20.000000 30.000000,20.000000 10.000000))"
	if got != want {
		t.Fatalf("bboxToWKT()=%q want %q", got, want)
	}
}

func TestHaversineMeters(t *testing.T) {
	// Distance between two nearby points (Golden Gate Bridge approx)
	d := haversineMeters(37.8199, -122.4783, 37.8078, -122.4750)
	if math.Abs(d-1350) > 50 { // allow some tolerance
		t.Fatalf("haversineMeters distance=%f unexpected", d)
	}
}

func TestNullIntPtr(t *testing.T) {
	if got := nullIntPtr(5, false); got != nil {
		t.Fatalf("expected nil when ok=false, got %v", got)
	}
	got := nullIntPtr(5, true)
	if got == nil || *got != 5 {
		t.Fatalf("unexpected value: %v", got)
	}
}

func TestParseTimeLocation(t *testing.T) {
	ts := "2020-01-02T15:04:05Z"
	parsed, ok := parseTime(ts)
	if !ok {
		t.Fatalf("expected parse to succeed")
	}
	if !parsed.Equal(time.Date(2020, 1, 2, 15, 4, 5, 0, time.UTC)) {
		t.Fatalf("unexpected time: %v", parsed)
	}
}
