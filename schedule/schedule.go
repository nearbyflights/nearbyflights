package schedule

import (
	"context"
	"time"

	"github.com/nearbyflights/nearbyflights/authentication"
	"github.com/nearbyflights/nearbyflights/bbox"
	"github.com/nearbyflights/nearbyflights/db"
	"github.com/nearbyflights/nearbyflights/dupe"
	log "github.com/sirupsen/logrus"
)

type Options struct {
	Interval  time.Duration
	Latitude  float64
	Longitude float64
	Radius    float64
}

type Scheduler struct {
	Client db.Client
}

func (s *Scheduler) GetFlights(ctx context.Context, newOptions chan Options) (<-chan []db.Flight, error) {
	currentOptions := <-newOptions
	flightsCh := make(chan []db.Flight)
	ticker := time.NewTicker(currentOptions.Interval)

	go func() {
		for {
			select {
			case <-ticker.C:
				flights, err := s.getFlights(ctx, currentOptions)
				if err != nil {
					log.Error(err)
					continue
				}

				flightsCh <- flights
			case receivedOptions := <-newOptions:
				currentOptions = receivedOptions
				ticker = time.NewTicker(currentOptions.Interval)
			case <-ctx.Done():
				log.Info("stream closed: finish get flights routine")
				return
			}
		}
	}()

	return flightsCh, nil
}

func (s *Scheduler) getFlights(ctx context.Context, options Options) ([]db.Flight, error) {
	boundingBox := bbox.NewBoundingBox(options.Latitude, options.Longitude, options.Radius)

	log.Infof("search bounds: http://bboxfinder.com/#%v \n", boundingBox)

	flights, err := s.Client.GetFlights(boundingBox)
	if err != nil {
		return nil, err
	}

	log.Infof("returned flights before dupe check: %v", flights)

	newFlights := flights[:0]
	clientId, err := authentication.GetClientId(ctx)
	if err != nil {
		log.Errorf("error while parsing client ID: %v", err)
	}

	for _, f := range flights {
		if !dupe.Exists(clientId, f.Icao24, time.Hour) {
			newFlights = append(newFlights, f)
		}
	}

	log.Infof("returned flights after dupe check: %v", newFlights)

	return newFlights, nil
}
