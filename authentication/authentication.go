package authentication

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var introspection string

type Introspection struct {
	Active     bool   `json:"active"`
	ClientId   string `json:"client_id"`
	Expiration int    `json:"exp"`
	Iat        int    `json:"iat"`
	Issuer     string `json:"iss"`
	Scope      string `json:"scope"`
	Sub        string `json:"sub"`
	TokenType  string `json:"token_type"`
}

func NewAuthInterceptor(introspectionUrl string) func(interface{}, grpc.ServerStream, *grpc.StreamServerInfo, grpc.StreamHandler) error {
	introspection = introspectionUrl
	return validateToken
}

func GetClientId(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("error while reading metadata")
	}

	clientId := md[clientId.String()]
	if len(clientId) < 1 {
		return "", errors.New("invalid client id")
	}

	return clientId[0], nil
}

func validateToken(srv interface{}, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := stream.Context()

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.InvalidArgument, "missing metadata")
	}

	ok, id := valid(md["authorization"])
	if !ok {
		return status.Errorf(codes.Unauthenticated, "invalid token")
	}

	md.Set(clientId.String(), id)

	err := stream.SetHeader(md)
	if err != nil {
		return status.Errorf(codes.Internal, "error setting client ID")
	}

	err = stream.SendHeader(md)
	if err != nil {
		return status.Errorf(codes.Internal, "error sending client ID")
	}

	return handler(srv, stream)
}

func valid(authorization []string) (bool, string) {
	if len(authorization) < 1 {
		return false, ""
	}

	token := strings.TrimPrefix(authorization[0], "Bearer ")

	resp, err := http.PostForm(introspection, url.Values{"token": {token}})
	if err != nil {
		log.Errorf("error when getting introspection response: %v", err)
		return false, ""
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("error when reading introspection response: %v", err)
		return false, ""
	}

	var introspection Introspection
	err = json.Unmarshal(body, &introspection)
	if err != nil {
		log.Errorf("error when unmarshalling introspection response %v", err)
		return false, ""
	}

	if !introspection.Active {
		log.Errorf("token is not active (expired or revoked)")
		return false, ""
	}

	fmt.Println(introspection)
	fmt.Printf("user %v authorized\n", introspection.Sub)

	return introspection.Active, introspection.ClientId
}
