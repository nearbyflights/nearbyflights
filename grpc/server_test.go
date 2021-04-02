package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-pg/pg/types"
	"github.com/nearbyflights/nearbyflights/authentication"
	"github.com/nearbyflights/nearbyflights/db"
	service "github.com/nearbyflights/nearbyflights/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var listener *bufconn.Listener
var client db.Client

// coordinates in the middle of the Pacific Ocean to avoid bumping with a real flight from the database
var (
	latitude  = 7.067274
	longitude = 220.202269
)

func init() {
	// mock authentication used when trying to validate the authentication token
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `
{
    "active": true,
    "client_id": "my-client",
    "exp": 1527078658,
    "iat": 1527075058,
    "iss": "http://127.0.0.1:4444/",
    "sub": "my-client",
    "token_type": "access_token"
}`)
	}))

	// we create our gRPC server using the bufconn package, an in-memory buffer used to test a gRPC server/client interaction
	listener = bufconn.Listen(bufSize)

	// we need a PostgreSQL database running on localhost for this to work
	database := db.ClientOptions{
		Address:  "localhost:5432",
		User:     "admin",
		Password: "secret",
		Database: "flights",
	}
	client = db.NewClient(database)

	wg := &sync.WaitGroup{}

	// we add a test flight that will be returned in the stream
	// we can't have the cron service running during this test because this could delete the test flight
	client.AddTestFlight(db.Flight{Geometry: types.Q(fmt.Sprintf("ST_SetSRID(ST_MakePoint(%v, %v),4326)", longitude, latitude)), Latitude: latitude, Longitude: longitude, Country: "BR", Icao24: "123456", Velocity: 10})

	// following logic will instantiate and serve the gRPC server
	server := &Server{UnimplementedNearbyFlightsServer: service.UnimplementedNearbyFlightsServer{}, Options: database, Context: context.Background(), Wg: wg}

	cert, err := credentials.NewServerTLSFromFile("../proto/x509/server.crt", "../proto/x509/server.key")
	if err != nil {
		log.Fatalf("error loading TLS certificate %v", err)
	}

	opts := []grpc.ServerOption{
		// Intercept request to check the token.
		grpc.StreamInterceptor(authentication.NewAuthInterceptor(ts.URL)),
		// Enable TLS for all incoming connections.
		grpc.Creds(cert),
	}

	grpcServer := grpc.NewServer(opts...)
	service.RegisterNearbyFlightsServer(grpcServer, server)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("error while running server: %v", err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return listener.Dial()
}

func TestReceive(t *testing.T) {
	// remove the test flight created for this test
	t.Cleanup(func() {
		client.RemoveTestFlight()
	})

	// fill the metadata with a mock authentication token
	md := metadata.New(map[string]string{"authorization": "Bearer test"})

	// force a 5 seconds timeout, we must have a response from the stream before this time expires
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// instantiates the gRPC client
	ctx = metadata.NewOutgoingContext(ctx, md)

	caCert, err := ioutil.ReadFile("../proto/x509/server.crt")
	if err != nil {
		log.Fatalln(err)
	}

	rootCAs := x509.NewCertPool()
	rootCAs.AppendCertsFromPEM(caCert)

	tlsConf := &tls.Config{
		RootCAs:            rootCAs,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
		ServerName:         "localhost",
	}

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(credentials.NewTLS(tlsConf)))
	if err != nil {
		t.Fatalf("failed to dial server: %v", err)
	}

	defer conn.Close()

	grpcClient := service.NewNearbyFlightsClient(conn)

	// initiate the duplex stream between client and server
	stream, err := grpcClient.Receive(ctx)
	if err != nil {
		t.Fatalf("error when receiving data from stream: %v", err)
	}

	// send new options (search interval, where am I and the radius I'm looking for flights)
	err = stream.Send(&service.Options{IntervalInSeconds: 1, Latitude: latitude, Longitude: longitude, Radius: 10000})
	if err != nil {
		t.Fatalf("error when sending options to stream: %v", err)
	}

	// receive nearby flights from the server in the stream
	// in a real client this would need to be wrapped in a loop for getting all nearby flights indefinitely
	flight, err := stream.Recv()
	if err != nil {
		t.Fatalf("error when reading data from stream: %v", err)
	}

	// the returned flight must be the test flight we created for this test
	if flight == nil || flight.CallSign != "test-flight" {
		t.Fatalf("unexpected flight info: %v", flight)
	}

	fmt.Println(flight)
}
