package couchbase

import (
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
)

const (
	defaultConnectTimeout = 10000 * time.Millisecond
	defaultKVTimeout      = 2500 * time.Millisecond
)

type Cluster struct {
	DSN      string `json:"dsn"                yaml:"dsn"                mapstructure:"dsn"`
	Username string `json:"username,omitempty" yaml:"username,omitempty" mapstructure:"username"`
	Password string `json:"password,omitempty" yaml:"password,omitempty" mapstructure:"password"`
}

// SetDefaults method.
func (c *Cluster) SetDefaults() {
}

func (c *Cluster) New() (*gocb.Cluster, error) {
	opts := gocb.ClusterOptions{
		Username: c.Username,
		Password: c.Password,
	}

	err := opts.ApplyProfile(gocb.ClusterConfigProfileWanDevelopment)
	if err != nil {
		return nil, fmt.Errorf("apply profile: %w", err)
	}

	cluster, err := gocb.Connect(c.DSN, opts)
	if err != nil {
		return nil, fmt.Errorf("gocb.Connect(%q): %w", c.DSN, err)
	}

	return cluster, nil
}
