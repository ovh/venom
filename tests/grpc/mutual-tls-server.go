/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package main implements a server for Greeter service.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/reflection"
)

var (
	port = 50051
)

const (
	caCertLocation     = "./pki/ca.crt"
	serverCertLocation = "./pki/server.crt"
	serverKeyLocation  = "./pki/server.key"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received: %v", in.GetName())
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	tlsCredentials, err := loadTLSCredentials()
	if err != nil {
		log.Fatalf("cannot load TLS credentials: %v", err)
	}

	s := grpc.NewServer(grpc.Creds(tlsCredentials))
	reflection.Register(s)
	pb.RegisterGreeterServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	serverCA, err := ioutil.ReadFile(caCertLocation)
	if err != nil {
		return nil, fmt.Errorf("failed to read server CA's PEM: %v", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(serverCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}

	// Load server's certificate and private key.
	serverCert, err := tls.LoadX509KeyPair(serverCertLocation, serverKeyLocation)
	if err != nil {
		return nil, fmt.Errorf("failed to load server cert and key: %v", err)
	}

	// Create the credentials and return it.
	config := &tls.Config{
		ClientCAs:    certPool, // this option restrict this server to mutual TLS.
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.NoClientCert,
		MinVersion:   tls.VersionTLS13,
	}

	return credentials.NewTLS(config), nil
}
