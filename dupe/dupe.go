package dupe

import "time"

type flightDupe struct {
	Icao24   string
	LastSeen time.Time
}

var flightsByUser = make(map[string][]flightDupe)

func Exists(token string, icao string, interval time.Duration) bool {
	flights := flightsByUser[token]

	exists := false
	dupeFlights := flights[:0]
	for _, f := range flights {
		if time.Now().Before(f.LastSeen.Add(interval)) {
			dupeFlights = append(dupeFlights, f)

			if f.Icao24 == icao {
				exists = true
			}
		}
	}

	flightsByUser[token] = dupeFlights

	if exists {
		return true
	}

	flightsByUser[token] = append(flightsByUser[token], flightDupe{Icao24: icao, LastSeen: time.Now()})

	return false
}
