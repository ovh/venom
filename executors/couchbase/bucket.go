package couchbase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/ovh/venom"
)

const (
	defaultCreateBucketFlushEnabled      = false
	defaultCreatebucketQueryPrimaryIndex = false
	defaultCreateIfNotExists             = false
	minCreateBucketQuota                 = 100
	minCreateBucketNumReplicas           = 0
	maxCreateBucketNumReplicas           = 3
)

type CreateBucketOptions struct {
	FlushEnabled      bool
	QueryPrimaryIndex bool
	Quota             uint64
	NumReplicas       uint32
}

func (cbo *CreateBucketOptions) SetDefaults() {
	cbo.FlushEnabled = defaultCreateBucketFlushEnabled
	cbo.QueryPrimaryIndex = defaultCreatebucketQueryPrimaryIndex
	cbo.Quota = minCreateBucketQuota
	cbo.NumReplicas = minCreateBucketNumReplicas
}

// coerceValues fix some out of range values.
func (cbo *CreateBucketOptions) coerceValues() {
	// num replicas must between 0 and 3
	cbo.NumReplicas = min(cbo.NumReplicas, maxCreateBucketNumReplicas)
	// couchbase server 6.5.0 has a minimum of 100 MB
	cbo.Quota = max(cbo.Quota, minCreateBucketQuota)
}

type Bucket struct {
	Name string `json:"name" yaml:"name" mapstructure:"name"`

	CreateIfNotExists bool
	CreateBucketOptions

	WaitUntilReady               bool    `json:"wait_until_ready,omitempty"                 yaml:"wait_until_ready,omitempty"                 mapstructure:"wait_until_ready"`
	WaitUntilReadyTimeoutSeconds float64 `json:"wait_until_ready_timeout_seconds,omitempty" yaml:"wait_until_ready_timeout_seconds,omitempty" mapstructure:"wait_until_ready_timeout_seconds"`
}

func (b *Bucket) SetDefaults() {
	b.CreateIfNotExists = defaultCreateIfNotExists

	b.CreateBucketOptions.SetDefaults()

	b.WaitUntilReady = defaultWaitUntilReady
	b.WaitUntilReadyTimeoutSeconds = defaultWaitUntilReadyTimeoutSeconds
}

func (b *Bucket) New(ctx context.Context, cluster *gocb.Cluster) (*gocb.Bucket, error) {
	if b.CreateIfNotExists {
		venom.Error(ctx, "trying to create bucket %s", b.Name)

		err := b.tryCreateBucketIfNotExists(ctx, cluster)
		if err != nil {
			return nil, err
		}
	}

	bucket := cluster.Bucket(b.Name)

	if !b.WaitUntilReady {
		return bucket, nil
	}

	timeout := time.Duration(b.WaitUntilReadyTimeoutSeconds * float64(time.Second))
	err := bucket.WaitUntilReady(timeout, nil)
	if err != nil {
		return nil, fmt.Errorf("bucket.WaitUntilReady(): %w", err)
	}

	return bucket, nil
}

func (b *Bucket) tryCreateBucketIfNotExists(ctx context.Context, cluster *gocb.Cluster) error {
	b.coerceValues()

	mgmg, err := gocb.Connect("couchbase://localhost:11210", gocb.ClusterOptions{
		Username: "venom",
		Password: "password",
	})
	if err != nil {
		return fmt.Errorf("unable to connect to management port: %w", err)
	}

	err = mgmg.WaitUntilReady(5*time.Second,
		&gocb.WaitUntilReadyOptions{
			DesiredState: gocb.ClusterStateOnline,
			ServiceTypes: []gocb.ServiceType{gocb.ServiceTypeManagement},
		},
	)
	if err != nil {
		return fmt.Errorf("management cluster not ready: %w", err)
	}

	err = mgmg.Buckets().CreateBucket(gocb.CreateBucketSettings{
		BucketSettings: gocb.BucketSettings{
			Name:         b.Name,
			FlushEnabled: b.FlushEnabled,
			RAMQuotaMB:   b.Quota,
			NumReplicas:  b.NumReplicas,
			BucketType:   gocb.CouchbaseBucketType,
		},
	}, nil)
	if err != nil && !errors.Is(err, gocb.ErrBucketExists) {
		return fmt.Errorf("unable to create bucket %q: %w", b.Name, err)
	}

	venom.Error(ctx, "does not need to create bucket err=%v", err)

	if !b.QueryPrimaryIndex {
		return nil
	}

	err = cluster.QueryIndexes().CreatePrimaryIndex(b.Name, nil)
	if err != nil {
		return fmt.Errorf("unable to create primary index for bucket %q: %w", b.Name, err)
	}

	return nil
}

// ErrNoKeyValueService error.
var ErrNoKeyValueService = errors.New("ping: no key-value service found")

func pingBucket(ctx context.Context, bucket *gocb.Bucket) error {
	result, err := bucket.Ping(&gocb.PingOptions{ //nolint:exhaustruct // sane default values.
		ServiceTypes: []gocb.ServiceType{
			gocb.ServiceTypeKeyValue,
		},
	})
	if err != nil {
		return fmt.Errorf("bucket.Ping(): %w", err)
	}

	if pings, ok := result.Services[gocb.ServiceTypeKeyValue]; !ok {
		return ErrNoKeyValueService
	} else {
		for i, ping := range pings {
			venom.Error(ctx, "ping bucket %d %v", i, ping)
		}
	}

	return nil
}
