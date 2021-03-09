package main

import (
	"google.golang.org/grpc/credentials"
	"net"
	"os"

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
	server := &grpcService.Server{UnimplementedNearbyFlightsServer: service.UnimplementedNearbyFlightsServer{}, Options: database}

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

	log.Info("starting server at port :8080")

	service.RegisterNearbyFlightsServer(grpcServer, server)
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatal(err)
	}
}
