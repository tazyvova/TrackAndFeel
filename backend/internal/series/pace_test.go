package series

import "testing"

func TestSpeedToPaceMinPerKm(t *testing.T) {
	tests := []struct {
		name   string
		speed  *float64
		expect *float64
	}{
		{name: "nil input", speed: nil, expect: nil},
		{name: "zero speed", speed: ptr(0), expect: nil},
		{name: "negative speed", speed: ptr(-1), expect: nil},
		{name: "4 mps", speed: ptr(4), expect: ptr(4.10)},
		{name: "3.33 mps", speed: ptr(3.33), expect: ptr(5.00)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SpeedToPaceMinPerKm([]*float64{tt.speed})[0]
			if (got == nil) != (tt.expect == nil) {
				t.Fatalf("nil mismatch: got %v expect %v", got, tt.expect)
			}
			if got == nil {
				return
			}
			if diff := *got - *tt.expect; diff < -0.005 || diff > 0.005 {
				t.Fatalf("pace mismatch: got %v expect %v", *got, *tt.expect)
			}
		})
	}
}

func ptr(f float64) *float64 {
	return &f
}
