package series

// SpeedToPaceMinPerKm converts an array of speeds (m/s) into pace expressed as
// decimal minutes per kilometre (e.g. 5.55 = 5'33"). Nil or non-positive
// speeds produce nil entries.
func SpeedToPaceMinPerKm(speeds []*float64) []*float64 {
	res := make([]*float64, len(speeds))
	for i, v := range speeds {
		if v == nil || *v <= 0 {
			res[i] = nil
			continue
		}

		secPerKm := 1000 / *v
		pace := secPerKm / 60

		// copy to avoid aliasing input values
		p := pace
		res[i] = &p
	}
	return res
}
