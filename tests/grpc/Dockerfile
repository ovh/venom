FROM golang:1.26

WORKDIR /app

COPY pki ./pki
COPY go.* ./
COPY mutual-tls-server.go ./

RUN go build -o server ./mutual-tls-server.go

EXPOSE 50051

ENTRYPOINT [ "./server" ]
