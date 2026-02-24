package rules

import (
	"math"
	"strconv"
	"strings"
)

const earthRadiusKM = 6371.0

func parseMetadataBool(metadata map[string]interface{}, key string) (bool, bool) {
	if metadata == nil {
		return false, false
	}
	raw, ok := metadata[key]
	if !ok {
		return false, false
	}

	switch v := raw.(type) {
	case bool:
		return v, true
	case float64:
		if v == 1 {
			return true, true
		}
		if v == 0 {
			return false, true
		}
		return false, false
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return false, false
		}
		return parsed, true
	default:
		return false, false
	}
}

func haversineKM(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := deg2rad(lat2 - lat1)
	dLon := deg2rad(lon2 - lon1)
	lat1 = deg2rad(lat1)
	lat2 = deg2rad(lat2)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1)*math.Cos(lat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKM * c
}

func deg2rad(v float64) float64 {
	return v * math.Pi / 180
}
