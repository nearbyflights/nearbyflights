package dupe

import (
	"testing"
	"time"
)

// ICAO for the Shuttle Carrier Aircraft owned by NASA
var icao = "AC82EC"

func TestExists(t *testing.T) {
	_ = Exists("1", icao, time.Second*5)

	exists := Exists("1", icao, time.Second*5)

	if !exists {
		t.Error("exists should be true")
	}
}

func TestExists_NotExists_AnotherFlight(t *testing.T) {
	_ = Exists("1", icao, time.Second*5)

	exists := Exists("1", "random", time.Second*5)

	if exists {
		t.Error("exists should be false")
	}
}

func TestExists_NotExists_DifferentUser(t *testing.T) {
	_ = Exists("1", icao, time.Second*5)

	exists := Exists("2", icao, time.Second*5)

	if exists {
		t.Error("exists should be false")
	}
}

func TestExists_NotExists_DupeExpired(t *testing.T) {
	_ = Exists("1", icao, time.Nanosecond*0)

	exists := Exists("1", icao, time.Nanosecond*0)

	if exists {
		t.Error("exists should be false")
	}
}
