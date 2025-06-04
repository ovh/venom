package couchbase

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/mitchellh/mapstructure"
	"github.com/ovh/venom"
)

const (
	// Name of executor
	Name = "couchbase"

	defaultDSN        = "couchbase://localhost"
	defaultTranscoder = "legacy"

	defaultWaitUntilReadyTimeout = 5 * time.Second
)

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

// Executor represents a Test Exec
type Executor struct {
	DSN        string   `json:"dsn"                  yaml:"dsn"                  mapstructure:"dsn"`
	Username   string   `json:"username,omitempty"   yaml:"username,omitempty"   mapstructure:"username"`
	Password   string   `json:"password,omitempty"   yaml:"password,omitempty"   mapstructure:"password"`
	Bucket     string   `json:"bucket"               yaml:"bucket"               mapstructure:"bucket"`
	Scope      string   `json:"scope,omitempty"      yaml:"scope,omitempty"      mapstructure:"scope"`
	Collection string   `json:"collection,omitempty" yaml:"collection,omitempty" mapstructure:"collection"`
	Transcoder string   `json:"transcoder,omitempty" yaml:"transcoder,omitempty" mapstructure:"transcoder"`
	Expiry     *float64 `json:"expiry,omitempty"     yaml:"expiry,omitempty"     mapstructure:"expiry"`

	WaitUntilReadyTimeout float64 `json:"wait_until_ready_timeout,omitempty" yaml:"wait_until_ready_timeout,omitempty" mapstructure:"wait_until_ready_timeout"`

	ProfileWanDevelopment bool `json:"profile_wan_development,omitempty" yaml:"profile_wan_development,omitempty" mapstructure:"profile_wan_development"`

	Actions []map[string]any `json:"actions,omitempty" yaml:"actions,omitempty" mapstructure:"actions"`

	buckets map[string]*gocb.Bucket
}

func (e *Executor) SetDefaults() {
	e.DSN = defaultDSN
	e.Transcoder = defaultTranscoder
	e.WaitUntilReadyTimeout = defaultWaitUntilReadyTimeout.Seconds()
}

type Result struct {
	Actions []any `json:"actions,omitempty" yaml:"actions,omitempty"`
}

// Run execute TestStep
func (e *Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	e.SetDefaults()

	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	cluster, err := e.getCluster(ctx)
	if err != nil {
		return nil, err
	}

	defer cluster.Close(&gocb.ClusterCloseOptions{})

	defer e.clearCache()

	results := make([]any, len(e.Actions))

	for index, action := range e.Actions {
		actionType := fmt.Sprintf("%v", action["type"])

		switch actionType {
		case "touch":
			results[index], err = e.doTouch(ctx, action, cluster)
			if err != nil {
				return nil, err
			}

		case "exists":
			results[index], err = e.doExists(ctx, action, cluster)
			if err != nil {
				return nil, err
			}

		case "delete":
			results[index], err = e.doDelete(ctx, action, cluster)
			if err != nil {
				return nil, err
			}

		case "get":
			results[index], err = e.doGet(ctx, action, cluster)
			if err != nil {
				return nil, err
			}

		case "upsert":
			results[index], err = e.doUpsert(ctx, action, cluster)
			if err != nil {
				return nil, err
			}

		case "insert":
			results[index], err = e.doInsert(ctx, action, cluster)
			if err != nil {
				return nil, err
			}

		case "replace":
			results[index], err = e.doReplace(ctx, action, cluster)
			if err != nil {
				return nil, err
			}

		default:
			return nil, fmt.Errorf("action type %q not supported", actionType)
		}
	}

	return Result{
		Actions: results,
	}, nil
}

func (e *Executor) clearCache() {
	clear(e.buckets)
}

var (
	errMissingCouchbaseDSN = errors.New("missing couchbase dsn")
	errMissingBucketName   = errors.New("missing couchbase bucket name")
)

func (e *Executor) getCluster(ctx context.Context) (*gocb.Cluster, error) {
	if e.DSN == "" {
		return nil, errMissingCouchbaseDSN
	}

	opts := gocb.ClusterOptions{}

	if e.Username != "" {
		opts.Username = e.Username
		opts.Password = e.Password
	}

	switch e.Transcoder {
	case "json":
		opts.Transcoder = gocb.NewJSONTranscoder()
	case "raw":
		opts.Transcoder = gocb.NewRawBinaryTranscoder()
	case "rawjson":
		opts.Transcoder = gocb.NewRawJSONTranscoder()
	case "rawstring":
		opts.Transcoder = gocb.NewRawStringTranscoder()
	case "legacy":
		opts.Transcoder = gocb.NewLegacyTranscoder()
	default:
		return nil, fmt.Errorf("invalid couchbase transcoder %q (valid values are %v)",
			e.Transcoder, []string{"json", "raw", "rawjson", "rawstring", "legacy"})
	}

	venom.Debug(ctx, "setting couchbase transcoder %q", e.Transcoder)

	if e.ProfileWanDevelopment {
		venom.Debug(ctx, "will connect to cluster using config profile wan development")

		opts.ApplyProfile(gocb.ClusterConfigProfileWanDevelopment)
	}

	cluster, err := gocb.Connect(e.DSN, opts)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to couchbase cluster %q: %w", e.DSN, err)
	}

	return cluster, nil
}

func (e *Executor) getBucket(ctx context.Context,
	cluster *gocb.Cluster,
	bucketName string,
) (*gocb.Bucket, error) {
	bucketName = cmp.Or(bucketName, e.Bucket)

	if bucketName == "" {
		return nil, errMissingBucketName
	}

	bucket, found := e.buckets[bucketName]
	if found {
		venom.Debug(ctx, "return bucket %q from cache", bucketName)

		return bucket, nil
	}

	bucket = cluster.Bucket(bucketName)

	if e.buckets == nil {
		e.buckets = map[string]*gocb.Bucket{}
	}

	e.buckets[bucketName] = bucket

	if e.WaitUntilReadyTimeout == 0.0 {
		venom.Debug(ctx, "skip wait until bucket ready")

		return bucket, nil
	}

	waitUntilReadyTimeout := float64ToDuration(e.WaitUntilReadyTimeout)
	err := bucket.WaitUntilReady(waitUntilReadyTimeout, nil)
	if err != nil {
		delete(e.buckets, bucketName)

		return nil, fmt.Errorf("couchbase bucket %q not ready: %w", bucketName, err)
	}

	return bucket, nil
}

func (e *Executor) getCollection(ctx context.Context,
	cluster *gocb.Cluster,
	bucketName string,
	collectionName string,
	scopeName string,
) (
	collection *gocb.Collection,
	err error,
) {
	bucket, err := e.getBucket(ctx, cluster, bucketName)
	if err != nil {
		return nil, err
	}

	collectionName = cmp.Or(collectionName, e.Collection)
	scopeName = cmp.Or(scopeName, e.Scope)

	collection = bucket.DefaultCollection()
	if collectionName != "" {
		scope := bucket.DefaultScope()
		if scopeName != "" {
			scope = bucket.Scope(scopeName)
		}

		collection = scope.Collection(collectionName)
	}

	venom.Debug(ctx, "return collection %q scope %q from bucket %q",
		collection.Name(), collection.ScopeName(), collection.Bucket().Name())

	return collection, nil
}

// baseAction basic structure.
type baseAction struct {
	Type       string `json:"type"                 yaml:"type"                 mapstructure:"type"`
	Bucket     string `json:"bucket,omitempty"     yaml:"bucket,omitempty"     mapstructure:"bucket"`
	Scope      string `json:"scope,omitempty"      yaml:"scope,omitempty"      mapstructure:"scope"`
	Collection string `json:"collection,omitempty" yaml:"collection,omitempty" mapstructure:"collection"`
}

// touchAction represents an exists in couchbase
type touchAction struct {
	baseAction

	Expiry *float64 `json:"expiry" yaml:"expiry" mapstructure:"expiry"`
	IDs    []string `json:"ids"    yaml:"ids"    mapstructure:"ids"`
}

func (e *Executor) doTouch(ctx context.Context,
	rawAction any,
	cluster *gocb.Cluster,
) (any, error) {
	var action touchAction

	if err := mapstructure.Decode(rawAction, &action); err != nil {
		return nil, fmt.Errorf("unable to decode exists action: %w", err)
	}

	collection, err := e.getCollection(ctx, cluster,
		action.Bucket, action.Collection, action.Scope)
	if err != nil {
		return nil, err
	}

	results := map[string]map[string]any{}

	expiry, ok := e.tryGetExpiry(action.Expiry)
	if !ok {
		return nil, errors.New("unable to perform touch operation: missing 'expiry'")
	}

	for _, id := range action.IDs {
		touched := false
		_, err := collection.Touch(id, expiry, nil)
		if errors.Is(err, gocb.ErrDocumentNotFound) {
			touched = false
		} else if err != nil {
			return nil, err
		}

		results[id] = map[string]any{
			"touched": touched,
		}
	}

	return results, nil
}

// existsAction represents an exists in couchbase
type existsAction struct {
	baseAction

	IDs []string `json:"ids" yaml:"ids" mapstructure:"ids"`
}

func (e *Executor) doExists(ctx context.Context,
	rawAction any,
	cluster *gocb.Cluster,
) (any, error) {
	var action existsAction

	if err := mapstructure.Decode(rawAction, &action); err != nil {
		return nil, fmt.Errorf("unable to decode exists action: %w", err)
	}

	collection, err := e.getCollection(ctx, cluster,
		action.Bucket, action.Collection, action.Scope)
	if err != nil {
		return nil, err
	}

	results := map[string]map[string]any{}

	for _, id := range action.IDs {
		docOut, err := collection.Exists(id, nil)
		if err != nil {
			return nil, err
		}

		results[id] = map[string]any{
			"found": docOut.Exists(),
		}
	}

	return results, nil
}

// deleteAction represents a delete/remove in couchbase
type deleteAction struct {
	baseAction

	IDs []string `json:"ids" yaml:"ids" mapstructure:"ids"`
}

func (e *Executor) doDelete(ctx context.Context,
	rawAction any,
	cluster *gocb.Cluster,
) (any, error) {
	var action deleteAction

	if err := mapstructure.Decode(rawAction, &action); err != nil {
		return nil, fmt.Errorf("unable to decode exists action: %w", err)
	}

	collection, err := e.getCollection(ctx, cluster,
		action.Bucket, action.Collection, action.Scope)
	if err != nil {
		return nil, err
	}

	results := map[string]map[string]any{}

	for _, id := range action.IDs {
		deleted := true

		_, err := collection.Remove(id, nil)
		if errors.Is(err, gocb.ErrDocumentNotFound) {
			deleted = false
		} else if err != nil {
			return nil, err
		}

		results[id] = map[string]any{
			"deleted": deleted,
		}
	}

	return results, nil
}

// getAction represents a get/fetch in couchbase
type getAction struct {
	baseAction

	WithExpiry bool     `json:"with_expiry,omitempty" yaml:"with_expiry,omitempty" mapstructure:"with_expiry"`
	Expiry     *float64 `json:"expiry,omitempty"      yaml:"expiry,omitempty"      mapstructure:"expiry"`
	IDs        []string `json:"ids"                   yaml:"ids"                   mapstructure:"ids"`
}

func (e *Executor) tryGetExpiry(actionExpiry *float64) (time.Duration, bool) {
	expiry := cmp.Or(actionExpiry, e.Expiry)

	if expiry != nil {
		return float64ToDuration(*expiry), true
	}

	return time.Duration(0), false
}

func (e *Executor) getExpiry(actionExpiry *float64) time.Duration {
	expiry, _ := e.tryGetExpiry(actionExpiry)

	return expiry
}

func (e *Executor) doGet(ctx context.Context,
	rawAction any,
	cluster *gocb.Cluster,
) (any, error) {
	var action getAction

	if err := mapstructure.Decode(rawAction, &action); err != nil {
		return nil, fmt.Errorf("unable to decode exists action: %w", err)
	}

	collection, err := e.getCollection(ctx, cluster,
		action.Bucket, action.Collection, action.Scope)
	if err != nil {
		return nil, err
	}

	results := map[string]map[string]any{}

	opts := &gocb.GetOptions{
		WithExpiry: action.WithExpiry,
	}

	for _, id := range action.IDs {
		var (
			data   any
			found  = true
			docOut *gocb.GetResult
			err    error
		)

		if expiry, ok := e.tryGetExpiry(action.Expiry); ok {
			docOut, err = collection.GetAndTouch(id, expiry, nil)
		} else {
			docOut, err = collection.Get(id, opts)
		}

		if errors.Is(err, gocb.ErrDocumentNotFound) {
			found = false
		} else if err != nil {
			return nil, err
		} else {
			if terr := docOut.Content(&data); terr != nil {
				return nil, fmt.Errorf("error while transcoding content of entry id=%q: %w", id, terr)
			}
		}

		getResult := map[string]any{
			"found": found,
			"data":  data,
		}

		if docOut != nil {
			if expiry := docOut.Expiry(); expiry != nil {
				getResult["expiry"] = *expiry
			}
		}

		results[id] = getResult
	}

	return results, nil
}

// upsertAction represents a upsert/insert or update in couchbase
type upsertAction struct {
	baseAction

	PreserveExpiry bool           `json:"preserve_expiry,omitempty" yaml:"preserve_expiry,omitempty" mapstructure:"preserve_expiry"`
	Expiry         *float64       `json:"expiry,omitempty"          yaml:"expiry,omitempty"          mapstructure:"expiry"`
	Entries        map[string]any `json:"entries"                   yaml:"entries"                   mapstructure:"entries"`
}

func (e *Executor) doUpsert(ctx context.Context,
	rawAction any,
	cluster *gocb.Cluster,
) (any, error) {
	var action upsertAction

	if err := mapstructure.Decode(rawAction, &action); err != nil {
		return nil, fmt.Errorf("unable to decode exists action: %w", err)
	}

	collection, err := e.getCollection(ctx, cluster,
		action.Bucket, action.Collection, action.Scope)
	if err != nil {
		return nil, err
	}

	results := map[string]map[string]any{}

	opts := &gocb.UpsertOptions{
		PreserveExpiry: action.PreserveExpiry,
	}

	if expiry, ok := e.tryGetExpiry(action.Expiry); ok {
		opts = &gocb.UpsertOptions{
			Expiry: expiry,
		}
	}

	ids := slices.Sorted(maps.Keys(action.Entries))
	for _, id := range ids {
		val := action.Entries[id]

		_, err := collection.Upsert(id, val, opts)
		if err != nil {
			return nil, err
		}

		results[id] = map[string]any{
			"upserted": true,
		}
	}

	return results, nil
}

// insertAction represents an insert in couchbase
type insertAction struct {
	baseAction

	Expiry  *float64       `json:"expiry,omitempty" yaml:"expiry,omitempty" mapstructure:"expiry"`
	Entries map[string]any `json:"entries"          yaml:"entries"          mapstructure:"entries"`
}

func (e *Executor) doInsert(ctx context.Context,
	rawAction any,
	cluster *gocb.Cluster,
) (any, error) {
	var action insertAction

	if err := mapstructure.Decode(rawAction, &action); err != nil {
		return nil, fmt.Errorf("unable to decode exists action: %w", err)
	}

	collection, err := e.getCollection(ctx, cluster,
		action.Bucket, action.Collection, action.Scope)
	if err != nil {
		return nil, err
	}

	results := map[string]map[string]any{}

	var opts *gocb.InsertOptions

	if expiry, ok := e.tryGetExpiry(action.Expiry); ok {
		opts = &gocb.InsertOptions{
			Expiry: expiry,
		}
	}

	ids := slices.Sorted(maps.Keys(action.Entries))
	for _, id := range ids {
		val := action.Entries[id]
		inserted := true

		_, err := collection.Insert(id, val, opts)
		if errors.Is(err, gocb.ErrDocumentExists) {
			inserted = false
		} else if err != nil {
			return nil, err
		}

		results[id] = map[string]any{
			"inserted": inserted,
		}
	}

	return results, nil
}

// replaceAction represents a replace in couchbase
type replaceAction struct {
	baseAction

	PreserveExpiry bool           `json:"preserve_expiry,omitempty" yaml:"preserve_expiry,omitempty" mapstructure:"preserve_expiry"`
	Expiry         *float64       `json:"expiry,omitempty"          yaml:"expiry,omitempty"          mapstructure:"expiry"`
	Entries        map[string]any `json:"entries"                   yaml:"entries"                   mapstructure:"entries"`
}

func (e *Executor) doReplace(ctx context.Context,
	rawAction any,
	cluster *gocb.Cluster,
) (any, error) {
	var action replaceAction

	if err := mapstructure.Decode(rawAction, &action); err != nil {
		return nil, fmt.Errorf("unable to decode exists action: %w", err)
	}

	collection, err := e.getCollection(ctx, cluster,
		action.Bucket, action.Collection, action.Scope)
	if err != nil {
		return nil, err
	}

	results := map[string]map[string]any{}

	opts := &gocb.ReplaceOptions{
		PreserveExpiry: action.PreserveExpiry,
	}

	if expiry, ok := e.tryGetExpiry(action.Expiry); ok {
		opts = &gocb.ReplaceOptions{
			Expiry: expiry,
		}
	}

	ids := slices.Sorted(maps.Keys(action.Entries))
	for _, id := range ids {
		val := action.Entries[id]
		replaced := true

		_, err := collection.Replace(id, val, opts)
		if errors.Is(err, gocb.ErrDocumentNotFound) {
			replaced = false
		} else if err != nil {
			return nil, err
		}

		results[id] = map[string]any{
			"replaced": replaced,
		}
	}

	return results, nil
}

func float64ToDuration(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}
