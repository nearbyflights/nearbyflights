package db

import (
	"fmt"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/types"
	"github.com/nearbyflights/nearbyflights/bbox"
	log "github.com/sirupsen/logrus"
)

type Flight struct {
	Id        int     `sql:"id"`
	Geometry  types.Q `sql:"geom"`
	Latitude  float64 `sql:"latitude"`
	Longitude float64 `sql:"longitude"`
	Country   string  `sql:"country"`
	CallSign  string  `sql:"call_sign"`
	Icao24    string  `sql:"icao"`
	Velocity  float64 `sql:velocity`
}

type Client struct {
	database *pg.DB
}

type ClientOptions struct {
	Address  string
	User     string
	Password string
	Database string
}

func NewClient(options ClientOptions) Client {
	db := pg.Connect(&pg.Options{
		Addr:     options.Address,
		User:     options.User,
		Password: options.Password,
		Database: options.Database,
	})

	return Client{db}
}

func (c *Client) AddTestFlight(flight Flight) error {
	flight.CallSign = "test-flight"
	_, err := c.database.Model(&flight).Insert()
	return err
}

func (c *Client) RemoveTestFlight() error {
	_, err := c.database.Model((*Flight)(nil)).Where("call_sign = 'test-flight'").Delete()
	return err
}

func (c *Client) GetFlights(box bbox.BoundingBox) ([]Flight, error) {
	var flights []Flight
	err := c.database.Model(&flights).Where(fmt.Sprintf("geom && ST_MakeEnvelope(%v, %v, %v, %v, 4326)", box.MinLongitude, box.MinLatitude, box.MaxLongitude, box.MaxLatitude)).Select()
	if err != nil {
		return nil, err
	}

	log.Infof("found %v flight(s)", len(flights))

	return flights, nil
}

func (c *Client) Close() {
	c.database.Close()
}
