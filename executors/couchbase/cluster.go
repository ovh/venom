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

	WaitUntilReady               bool    `json:"wait_until_ready,omitempty"                 yaml:"wait_until_ready,omitempty"                 mapstructure:"wait_until_ready"`
	WaitUntilReadyTimeoutSeconds float64 `json:"wait_until_ready_timeout_seconds,omitempty" yaml:"wait_until_ready_timeout_seconds,omitempty" mapstructure:"wait_until_ready_timeout_seconds"`
}

// SetDefaults method.
func (c *Cluster) SetDefaults() {
	c.WaitUntilReady = defaultWaitUntilReady
	c.WaitUntilReadyTimeoutSeconds = defaultWaitUntilReadyTimeoutSeconds
}

func (c *Cluster) New() (*gocb.Cluster, error) {
	opts := gocb.ClusterOptions{
		Username: c.Username,
		Password: c.Password,
		TimeoutsConfig: gocb.TimeoutsConfig{
			ConnectTimeout: defaultConnectTimeout,
			KVTimeout:      defaultKVTimeout,
		},
	}

	err := opts.ApplyProfile(gocb.ClusterConfigProfileWanDevelopment)
	if err != nil {
		return nil, fmt.Errorf("apply profile: %w", err)
	}

	cluster, err := gocb.Connect(c.DSN, opts)
	if err != nil {
		return nil, fmt.Errorf("gocb.Connect(%q): %w", c.DSN, err)
	}

	if !c.WaitUntilReady {
		return cluster, nil
	}

	timeout := time.Duration(c.WaitUntilReadyTimeoutSeconds * float64(time.Second))

	gocb.SetLogger(gocb.DefaultStdioLogger())

	err = cluster.WaitUntilReady(timeout,
		&gocb.WaitUntilReadyOptions{
			DesiredState: gocb.ClusterStateOnline,
			ServiceTypes: []gocb.ServiceType{gocb.ServiceTypeManagement},
		},
	)
	if err != nil {
		defer cluster.Close(nil)

		return nil, fmt.Errorf("cluster.WaitUntilReady(): %w", err)
	}

	return cluster, nil
}
