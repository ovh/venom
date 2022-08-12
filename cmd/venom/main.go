package main

import (
	"github.com/ovh/venom/cmd/venom/root"
	log "github.com/sirupsen/logrus"
)

func main() {
	if err := root.New().Execute(); err != nil {
		log.Fatalf("Err:%s", err)
	}
}
