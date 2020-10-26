# builder container
FROM golang:1.14.4-alpine AS builder
LABEL stage=intermediate

WORKDIR /app
COPY . ./

# build the service
RUN CGO_ENABLED=0 go build \
    -mod=vendor \
    -a \
    -o /usr/local/bin/venom \
    ./cli/venom/.
RUN chmod u+x /usr/local/bin/venom
RUN venom version
