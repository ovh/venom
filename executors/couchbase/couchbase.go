package couchbase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/mitchellh/mapstructure"
	"github.com/ovh/venom"
)

const (
	// Name of executor
	Name = "couchbase"

	defaultMustPingService              = false
	defaultWaitUntilReady               = false
	defaultWaitUntilReadyTimeoutSeconds = 10
)

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

// Executor represents a Test Exec
type Executor struct {
	Cluster         Cluster `json:"cluster"                     yaml:"cluster"                     mapstructure:"cluster"`
	Bucket          Bucket  `json:"bucket"                      yaml:"bucket"                      mapstructure:"bucket"`
	Scope           string  `json:"scope,omitempty"             yaml:"scope,omitempty"             mapstructure:"scope"`
	Collection      string  `json:"collection,omitempty"        yaml:"collection,omitempty"        mapstructure:"collection"`
	MustPingService bool    `json:"must_ping_service,omitempty" yaml:"must_ping_service,omitempty" mapstructure:"must_ping_service"`

	Actions []map[string]any `json:"actions,omitempty" yaml:"actions,omitempty" mapstructure:"actions"`
}

type Result struct {
	Actions []map[string]any `json:"actions,omitempty" yaml:"actions,omitempty"`
}

func (e *Executor) SetDefaults() {
	e.Cluster.SetDefaults()
	e.Bucket.SetDefaults()
	e.MustPingService = defaultMustPingService
}

func decodeStructure(input, output any) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.StringToTimeDurationHookFunc(),
		ZeroFields: false,
		// WeaklyTypedInput: true,
		Result: &output,
	})
	if err != nil {
		return err
	}

	return decoder.Decode(input)
}

// Run execute TestStep
func (e *Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	if err := decodeStructure(step, &e); err != nil {
		return nil, err
	}

	venom.Error(ctx, "executor: %v", e)

	collection, done, err := e.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	defer done()

	results := make([]map[string]any, len(e.Actions))

	for index, action := range e.Actions {
		actionType := fmt.Sprintf("%v", action["type"])

		switch actionType {
		case "exists":
			var existsAction ExistsAction

			if err := decodeStructure(action, &existsAction); err != nil {
				return nil, err
			}

			existsResults := make([]map[string]any, 0, len(existsAction.IDs))

			for _, id := range existsAction.IDs {
				docOut, err := collection.Exists(id, nil)
				if err != nil {
					return nil, err
				}

				existsResults = append(existsResults, map[string]any{
					"id":    id,
					"found": docOut.Exists(),
				})
			}

			results[index] = map[string]any{"results": existsResults}

		case "get":
			var getAction GetAction

			if err := decodeStructure(action, &getAction); err != nil {
				return nil, err
			}

			getResults := make([]map[string]any, 0, len(getAction.IDs))

			opts := &gocb.GetOptions{WithExpiry: true}

			for _, id := range getAction.IDs {
				var docOut interface {
					Content(any) error
					Expiry() *time.Duration
				}

				if expiry := getAction.Expiration; expiry != nil && *expiry > time.Duration(0) {
					docOut, err = collection.GetAndTouch(id, *expiry, nil)
				} else {
					docOut, err = collection.Get(id, &gocb.GetOptions{
						WithExpiry: true,
						Timeout:    5 * time.Second,
					})
				}

				docOut, err := collection.Get(id, opts)
				if errors.Is(err, gocb.ErrDocumentNotFound) {
					getResults = append(getResults, map[string]any{
						"id":    id,
						"found": false,
					})
				} else if err != nil {
					return nil, fmt.Errorf("unable to get document id %q: %w", id, err)
				} else {
					var value any
					if cerr := docOut.Content(&value); cerr != nil {
						return nil, fmt.Errorf("unable to decode document id %q: %w", id, err)
					}
					getResult := map[string]any{
						"id":    id,
						"found": true,
						"value": value,
					}

					if expiry := docOut.Expiry(); expiry != nil {
						getResult["expiry"] = *expiry
					}

					getResults = append(getResults, getResult)
				}
			}

			results[index] = map[string]any{"results": getResults}

		case "insert":
			var insertAction InsertAction

			if err := decodeStructure(action, &insertAction); err != nil {
				return nil, err
			}

			insertResults := make([]map[string]any, 0, len(insertAction.Documents))

			var opts *gocb.InsertOptions
			// if insertAction.Expiration != nil {
			// 	opts = &gocb.InsertOptions{Expiry: *insertAction.Expiration}
			// }

			opts = &gocb.InsertOptions{
				Timeout: 5 * time.Second,
			}

			for id, val := range insertAction.Documents {
				_, err := collection.Insert(id, val, opts)
				if errors.Is(err, gocb.ErrDocumentExists) {
					insertResults = append(insertResults, map[string]any{
						"id":       id,
						"inserted": false,
					})
				} else if err != nil {
					return nil, fmt.Errorf("unable to insert document id %q: %w", id, err)
				} else {
					insertResults = append(insertResults, map[string]any{
						"id":       id,
						"inserted": true,
					})
				}
			}

			results[index] = map[string]any{"results": insertResults}

		case "update":
			var updateAction UpdateAction

			if err := decodeStructure(action, &updateAction); err != nil {
				return nil, err
			}

			updateResults := make([]map[string]any, 0, len(updateAction.Documents))

			var opts *gocb.ReplaceOptions
			if updateAction.Expiration != nil {
				opts = &gocb.ReplaceOptions{Expiry: *updateAction.Expiration}
			}

			for id, val := range updateAction.Documents {
				_, err := collection.Replace(id, val, opts)
				if errors.Is(err, gocb.ErrDocumentNotFound) {
					updateResults = append(updateResults, map[string]any{
						"id":      id,
						"updated": false,
					})
				} else if err != nil {
					return nil, fmt.Errorf("unable to update document id %q: %w", id, err)
				} else {
					updateResults = append(updateResults, map[string]any{
						"id":      id,
						"updated": true,
					})
				}
			}

			results[index] = map[string]any{"results": updateResults}

		case "upsert":
			var upsertAction UpsertAction

			if err := decodeStructure(action, &upsertAction); err != nil {
				return nil, err
			}

			upsertResults := make([]map[string]any, 0, len(upsertAction.Documents))

			var opts *gocb.UpsertOptions
			if upsertAction.Expiration != nil {
				opts = &gocb.UpsertOptions{Expiry: *upsertAction.Expiration}
			}

			for id, val := range upsertAction.Documents {
				_, err := collection.Upsert(id, val, opts)
				if err != nil {
					return nil, fmt.Errorf("unable to update document id %q: %w", id, err)
				} else {
					upsertResults = append(upsertResults, map[string]any{
						"id":       id,
						"upserted": true,
					})
				}
			}

			results[index] = map[string]any{"results": upsertResults}

		case "delete":
			var removeAction RemoveAction

			if err := decodeStructure(action, &removeAction); err != nil {
				return nil, err
			}

			removeResults := make([]map[string]any, 0, len(removeAction.IDs))

			for _, id := range removeAction.IDs {
				_, err := collection.Remove(id, nil)
				if errors.Is(err, gocb.ErrDocumentNotFound) {
					removeResults = append(removeResults, map[string]any{
						"id":      id,
						"deleted": false,
					})
				} else if err != nil {
					return nil, fmt.Errorf("unable to delete document id %q: %w", id, err)
				} else {
					removeResults = append(removeResults, map[string]any{
						"id":      id,
						"deleted": true,
					})
				}
			}

			results[index] = map[string]any{"results": removeResults}

		default:
			return nil, fmt.Errorf("action type %q not supported", actionType)
		}
	}

	return Result{
		Actions: results,
	}, nil
}

// Action basic structure.
type Action struct {
	Type string `json:"type" yaml:"type" mapstructure:"type"`
}

// ExistsAction represents an exists in couchbase
type ExistsAction struct {
	Action

	IDs []string `json:"ids" yaml:"ids" mapstructure:"ids"`
}

// GetAction represent an get/fetch in couchbase
type GetAction struct {
	Action

	Expiration *time.Duration `json:"expiration,omitempty" yaml:"expiration,omitempty" mapstructure:"expiration"`
	IDs        []string       `json:"ids"                   yaml:"ids"                   mapstructure:"ids"`
}

// RemoveAction represents an remove/delete in couchbase
type RemoveAction struct {
	Action

	IDs []string `json:"ids" yaml:"ids" mapstructure:"ids"`
}

// InsertAction represents an insert/creation in couchbase.
type InsertAction struct {
	Action

	// Expiration *time.Duration `json:"expiration,omitempty" yaml:"expiration,omitempty" mapstructure:"expiration"`
	Documents map[string]any `json:"documents"            yaml:"documents"            mapstructure:"documents"`
}

// UpdateAction represents an update/replace in couchbase.
type UpdateAction struct {
	Action

	Expiration *time.Duration `json:"expiration,omitempty" yaml:"expiration,omitempty" mapstructure:"expiration"`
	Documents  map[string]any `json:"documents"            yaml:"documents"            mapstructure:"documents"`
}

// UpsertAction represents an upsert/insert or replace in couchbase.
type UpsertAction struct {
	Action

	Expiration *time.Duration `json:"expiration,omitempty" yaml:"expiration,omitempty" mapstructure:"expiration"`
	Documents  map[string]any `json:"documents"            yaml:"documents"            mapstructure:"documents"`
}
