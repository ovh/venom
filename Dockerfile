FROM golang:1.15-buster as build

RUN apt-get update && \
    apt-get install -y unixodbc unixodbc-dev

WORKDIR /go/src/venom
COPY . .
RUN go build -a -installsuffix cgo -ldflags "-X github.com/ovh/venom.Version=$(git describe)" -o /go/bin ./...
RUN chmod a+rx /go/bin/venom

FROM debian:buster-slim 

RUN apt-get update && \
    apt-get install -y unixodbc && \
    rm -rf /var/lib/apt/lists/*

COPY --from=build /go/bin/venom /opt/venom

ENTRYPOINT ["/opt/venom"]
