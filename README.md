# nearbyflights

This is a gRPC server that will return nearby flights for multiple clients via client/server streaming based on their current coordinates, search radius and interval between searches.

## Running

```
go run main.go
```

This assumes you have an ORY Hydra login service and a PostgreSQL database running at localhost.

## Logic

This gRPC server only has one endpoint: `Receive`. This endpoint has client and server streaming for sending search options (current coordinates, search radius and interval between searches) by the client or nearby flights based on the user's criteria by the server.

For calling the `Receive` endpoint you must be authorized by an ORY Hydra OpenID server configured by using the `INTROSPECTION_URL` env variable.. 

## Development

### Regenerate Protobuf 

For regenerating the Protobuf file for the server run: 

```
cd proto
.\protoc.exe --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative service.proto
```

This uses the Protobuf compiler for Windows, use our own in case you are using Linux.

### Server certificate

This server uses TLS by default and for obvious reasons the certificate and the private key aren't available in this repo. For generating your own files run:

```
openssl genrsa -out server.key 2048
openssl req -nodes -new -x509 -sha256 -config cert.conf -extensions 'req_ext' -key server.key -out server.crt
```

The `cert.conf` file must be in the following format:
```
[ req ]
default_bits = 2048
prompt = no
default_md = sha256
req_extensions = req_ext
distinguished_name = dn
 
[ dn ]
C = UK
ST = .
L = .
O = .
OU = .
emailAddress = email@email.com
CN = localhost
 
[ req_ext ]
subjectAltName = @alt_names
 
[ alt_names ]
DNS.1 = localhost
``` 

Move the generated files to the proto/x509 folder, and you are good to go.

### Environment variables

| Name              | Description                         | Default                                  |
| ----------------- | ----------------------------------- | -----------------------------------------|
| POSTGRES_URL      | PostgreSQL host                     | localhost:5432                           |
| POSTGRES_USER     | PostgreSQL username                 | admin                                    |
| POSTGRES_PASSWORD | PostgreSQL password                 | secret                                   |
| POSTGRES_DB       | PostgreSQL database name            | flights                                  |
| INTROSPECTION_URL | URL for Open ID token introspection | http://localhost:4445/oauth2/introspect  |

### Run in Docker

```
docker build -t nearbyflights . && docker run --rm --detach --name nearbyflights-standalone nearbyflights
```