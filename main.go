package main

import (
	"context"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/envconfig"
	"github.com/nearbyflights/nearbyflights/db"
	grpcService "github.com/nearbyflights/nearbyflights/grpc"
	service "github.com/nearbyflights/nearbyflights/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type Configuration struct {
	PostgresUrl           string `required:"true" envconfig:"POSTGRES_URL" default:"localhost:5432"`
	User                  string `required:"true" envconfig:"POSTGRES_USER" default:"admin"`
	Password              string `required:"true" envconfig:"POSTGRES_PASSWORD" default:"secret"`
	DatabaseName          string `required:"true" envconfig:"POSTGRES_DB" default:"flights"`
	IntrospectionUrl      string `required:"true" envconfig:"INTROSPECTION_URL" default:"http://localhost:4445/oauth2/introspect"`
	TlsCertificatePath    string `required:"true" envconfig:"TLS_CERTIFICATE_PATH" default:"./proto/x509/server.crt"`
	TlsCertificateKeyPath string `required:"true" envconfig:"TLS_CERTIFICATE_KEY_PATH" default:"./proto/x509/server.key"`
}

func init() {
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{ForceColors: true})
}

func main() {
	var c Configuration
	err := envconfig.Process("", &c)
	if err != nil {
		log.Fatal(err.Error())
	}

	database := db.ClientOptions{
		Address:  c.PostgresUrl,
		User:     c.User,
		Password: c.Password,
		Database: c.DatabaseName,
	}

	cert, err := credentials.NewServerTLSFromFile(c.TlsCertificatePath, c.TlsCertificateKeyPath)
	if err != nil {
		log.Fatalf("error loading TLS certificate %v", err)
	}

	opts := []grpc.ServerOption{
		// Intercept request to check the token.
		// grpc.StreamInterceptor(authentication.NewAuthInterceptor(c.IntrospectionUrl)),
		// Enable TLS for all incoming connections.
		grpc.Creds(cert),
	}

	grpcServer := grpc.NewServer(opts...)
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("error creating the server %v", err)
	}

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("nearbyflights", grpc_health_v1.HealthCheckResponse_SERVING)

	log.Info("starting server at port :8080")

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		signal := <-signals
		log.Printf("server closed: %v", signal)
		cancel()
		log.Exit(0)
	}()

	server := &grpcService.Server{HealthServer: healthServer, Options: database, Context: ctx, UnimplementedNearbyFlightsServer: service.UnimplementedNearbyFlightsServer{}}
	service.RegisterNearbyFlightsServer(grpcServer, server)
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatal(err)
	}
}
