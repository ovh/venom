package couchbase

import (
	"context"

	"github.com/couchbase/gocb/v2"
	"github.com/ovh/venom"
)

func (e *Executor) getCollection(ctx context.Context) (
	collection *gocb.Collection,
	done func(),
	err error,
) {
	cluster, err := e.Cluster.New()
	if err != nil {
		return nil, nil, err
	}

	done = func() {
		if cerr := cluster.Close(nil); cerr != nil {
			venom.Error(ctx, "closing cluster return error: %v", cerr)
		}
	}

	bucket, err := e.Bucket.New(ctx, cluster)
	if err != nil {
		defer done()

		return nil, nil, err
	}

	if !e.MustPingService {
		return collection, done, nil
	}

	err = pingBucket(ctx, bucket)
	if err != nil {
		defer done()

		return nil, nil, err
	}

	collection = bucket.DefaultCollection()
	if e.Collection != "" {
		scope := bucket.DefaultScope()
		if e.Scope != "" {
			scope = bucket.Scope(e.Scope)
		}

		collection = scope.Collection(e.Collection)
	}

	return collection, done, nil
}
