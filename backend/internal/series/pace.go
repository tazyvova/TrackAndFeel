package series

import "math"

// SpeedToPaceMinPerKm converts an array of speeds (m/s) into pace expressed as
// decimal minutes per kilometre (e.g. 5.12 = 5'12"). Nil or non-positive
// speeds produce nil entries.
func SpeedToPaceMinPerKm(speeds []*float64) []*float64 {
	res := make([]*float64, len(speeds))
	for i, v := range speeds {
		if v == nil || *v <= 0 {
			res[i] = nil
			continue
		}

		secPerKm := 1000 / *v
		minutes := math.Floor(secPerKm / 60)
		seconds := math.Round(math.Mod(secPerKm, 60))
		pace := minutes + seconds/100

		// copy to avoid aliasing input values
		p := pace
		res[i] = &p
	}
	return res
}
