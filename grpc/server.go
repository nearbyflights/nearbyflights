package grpc

import (
	"context"
	"errors"
	"google.golang.org/grpc/health"
	"sync"
	"time"

	"github.com/nearbyflights/nearbyflights/db"
	service "github.com/nearbyflights/nearbyflights/proto"
	"github.com/nearbyflights/nearbyflights/schedule"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	HealthServer *health.Server
	Options      db.ClientOptions
	Context      context.Context
	Wg			 *sync.WaitGroup
	service.UnimplementedNearbyFlightsServer
}

func (s *Server) Receive(stream service.NearbyFlights_ReceiveServer) error {
	errorCh := make(chan error)
	newOptions := make(chan schedule.Options)
	client := db.NewClient(s.Options)
	scheduler := schedule.Scheduler{Client: client}

	ctx := stream.Context()

	s.Wg.Add(1)
	go func() {
		defer s.Wg.Done()

		for {
			options, err := stream.Recv()
			if err != nil {
				st, ok := status.FromError(err)

				if !ok {
					log.Error(err)
					continue
				}

				if st.Code() == codes.Canceled {
					log.Info("stream closed: finish receive routine")
					return
				}
			}

			if options == nil {
				continue
			}

			log.Info("received new options from client")

			newOptions <- schedule.Options{
				Latitude:  options.Latitude,
				Longitude: options.Longitude,
				Radius:    options.Radius,
				Interval:  time.Second * time.Duration(options.IntervalInSeconds),
			}
		}
	}()

	flights, err := scheduler.GetFlights(ctx, newOptions)
	if err != nil {
		return err
	}

	s.Wg.Add(1)
	go func() {
		defer s.Wg.Done()

		for {
			select {
			case flights := <-flights:
				for _, f := range flights {
					stream.Send(&service.Flight{
						Latitude:  f.Latitude,
						Longitude: f.Longitude,
						Country:   f.Country,
						CallSign:  f.CallSign,
						Icao24:    f.Icao24,
						Velocity:  f.Velocity})
				}
			case <-s.Context.Done():
				errorCh <- errors.New("server stopped")
				return
			case <-ctx.Done():
				log.Info("stream closed: finish send routine")
				return
			}
		}
	}()

	error := <-errorCh
	log.Error(error)

	log.Info("closing db connection")
	client.Close()

	return error
}
