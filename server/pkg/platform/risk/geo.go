package risk

import "math"

// HaversineMeters returns great-circle distance in meters between two WGS84 points.
func HaversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earth = 6371000.0
	toRad := math.Pi / 180
	φ1 := lat1 * toRad
	φ2 := lat2 * toRad
	Δφ := (lat2 - lat1) * toRad
	Δλ := (lng2 - lng1) * toRad
	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) +
		math.Cos(φ1)*math.Cos(φ2)*math.Sin(Δλ/2)*math.Sin(Δλ/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earth * c
}
