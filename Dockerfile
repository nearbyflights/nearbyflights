FROM golang:1.15.2-buster

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["sh", "-c", "nearbyflights"]

EXPOSE 8080