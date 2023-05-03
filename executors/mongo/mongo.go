package mongo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/ovh/venom"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/yaml.v3"
)

const Name = "mongo"

func New() venom.Executor {
	return &Executor{}
}

type Executor struct {
	URI        string           `json:"uri,omitempty" yaml:"uri,omitempty"`
	Database   string           `json:"database,omitempty" yaml:"database,omitempty"`
	Collection string           `json:"collection,omitempty" yaml:"collection,omitempty"`
	Actions    []map[string]any `json:"actions,omitempty" yaml:"actions,omitempty"`
}

type Result struct {
	Actions []map[string]any `json:"actions,omitempty" yaml:"actions,omitempty"`
}

func (e Executor) Run(ctx context.Context, step venom.TestStep) (any, error) {
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	venom.Debug(ctx, "connecting to database: %s\n", e.URI)
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(e.URI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer mongoClient.Disconnect(ctx)

	results := make([]map[string]any, len(e.Actions))

	for i, action := range e.Actions {
		actionType := fmt.Sprintf("%v", action["type"])
		switch actionType {
		case "loadFixtures":
			var loadFixturesAction LoadFixturesAction
			if err := mapstructure.Decode(action, &loadFixturesAction); err != nil {
				return nil, err
			}

			if loadFixturesAction.Folder == "" {
				return nil, fmt.Errorf("folder is required")
			}

			// First, drop the existing collections in the database to start clean
			collections, err := mongoClient.Database(e.Database).ListCollectionNames(ctx, bson.M{})
			if err != nil {
				return nil, fmt.Errorf("failed to list collections: %w", err)
			}

			for _, collection := range collections {
				if strings.HasPrefix(collection, "system.") {
					continue
				}

				if err := mongoClient.Database(e.Database).Collection(collection).Drop(ctx); err != nil {
					return nil, fmt.Errorf("failed to drop collection %s: %w", collection, err)
				}
			}

			dirEntries, err := os.ReadDir(path.Join(venom.StringVarFromCtx(ctx, "venom.testsuite.workdir"), loadFixturesAction.Folder))
			if err != nil {
				return nil, err
			}

			fixtures := make([]string, 0, len(dirEntries))
			for _, file := range dirEntries {
				if file.IsDir() {
					continue
				}

				extension := path.Ext(file.Name())
				if extension != ".yaml" && extension != ".yml" {
					continue
				}
				fixtures = append(fixtures, file.Name())
			}

			for _, fixture := range fixtures {
				collectionName := strings.TrimSuffix(fixture, path.Ext(fixture))
				filePath := path.Join(venom.StringVarFromCtx(ctx, "venom.testsuite.workdir"), loadFixturesAction.Folder, fixture)

				file, err := os.Open(filePath)
				if err != nil {
					return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
				}

				var documents []any
				if err := yaml.NewDecoder(file).Decode(&documents); err != nil {
					return nil, fmt.Errorf("failed to decode fixture %s: %w", filePath, err)
				}

				if _, err := mongoClient.Database(e.Database).Collection(collectionName).InsertMany(ctx, documents); err != nil {
					return nil, fmt.Errorf("failed to insert documents in collection %s: %w", collectionName, err)
				}
			}

		case "dropCollection":
			if err := mongoClient.Database(e.Database).Collection(e.Collection).Drop(ctx); err != nil {
				return nil, err
			}

		case "createCollection":
			if err := mongoClient.Database(e.Database).CreateCollection(ctx, e.Collection); err != nil {
				return nil, err
			}

		case "count":
			var countAction CountAction
			if err := mapstructure.Decode(action, &countAction); err != nil {
				return nil, err
			}

			filter := bson.M{}
			if countAction.Filter != "" {
				if err := bson.UnmarshalExtJSON([]byte(countAction.Filter), false, &filter); err != nil {
					return nil, err
				}
			}

			var countOptions []*options.CountOptions
			if countAction.Options.Limit != nil {
				countOptions = append(countOptions, options.Count().SetLimit(*countAction.Options.Limit))
			}

			count, err := mongoClient.Database(e.Database).Collection(e.Collection).CountDocuments(ctx, filter, countOptions...)
			if err != nil {
				return nil, err
			}

			results[i] = map[string]any{"count": count}

		case "insert":
			var insertAction InsertAction
			if err := mapstructure.Decode(action, &insertAction); err != nil {
				return nil, err
			}

			var documents []any

			if insertAction.File != "" {
				filePath := path.Join(venom.StringVarFromCtx(ctx, "venom.testsuite.workdir"), insertAction.File)
				file, err := os.Open(filePath)
				if err != nil {
					return nil, err
				}

				decoder := json.NewDecoder(file)
				for {
					var documentJSON any
					if err := decoder.Decode(&documentJSON); err != nil {
						if err == io.EOF {
							break
						}
						return nil, err
					}

					documentBytes, err := json.Marshal(documentJSON)
					if err != nil {
						return nil, err
					}

					var document bson.M
					if err := bson.UnmarshalExtJSON(documentBytes, false, &document); err != nil {
						return nil, err
					}
					documents = append(documents, document)
				}
			}

			for _, documentJSON := range insertAction.Documents {
				var document bson.M
				if err := bson.UnmarshalExtJSON([]byte(documentJSON), false, &document); err != nil {
					return nil, err
				}
				documents = append(documents, document)
			}

			insertManyResult, err := mongoClient.Database(e.Database).Collection(e.Collection).InsertMany(ctx, documents)
			if err != nil {
				return nil, err
			}

			var result map[string]any
			if err := mapstructure.Decode(insertManyResult, &result); err != nil {
				return nil, err
			}
			results[i] = result

		case "find":
			var findAction FindAction
			if err := mapstructure.Decode(action, &findAction); err != nil {
				return nil, err
			}

			filter := bson.M{}
			if findAction.Filter != "" {
				if err := bson.UnmarshalExtJSON([]byte(findAction.Filter), false, &filter); err != nil {
					return nil, err
				}
			}

			var findOptions []*options.FindOptions
			if findAction.Options.Limit != nil {
				findOptions = append(findOptions, options.Find().SetLimit(*findAction.Options.Limit))
			}
			if findAction.Options.Skip != nil {
				findOptions = append(findOptions, options.Find().SetSkip(*findAction.Options.Skip))
			}
			if findAction.Options.Sort != "" {
				var sort bson.M
				if err := bson.UnmarshalExtJSON([]byte(findAction.Options.Sort), false, &sort); err != nil {
					return nil, err
				}
				findOptions = append(findOptions, options.Find().SetSort(sort))
			}
			if findAction.Options.Projection != "" {
				var projection bson.M
				if err := bson.UnmarshalExtJSON([]byte(findAction.Options.Projection), false, &projection); err != nil {
					return nil, err
				}
				findOptions = append(findOptions, options.Find().SetSort(projection))
			}

			cursor, err := mongoClient.Database(e.Database).Collection(e.Collection).Find(ctx, filter, findOptions...)
			if err != nil {
				return nil, err
			}

			var result []map[string]any
			if err := cursor.All(ctx, &result); err != nil {
				return nil, err
			}
			results[i] = map[string]any{"results": result}

		case "update":
			var updateAction UpdateAction
			if err := mapstructure.Decode(action, &updateAction); err != nil {
				return nil, err
			}

			filter := bson.M{}
			if updateAction.Filter != "" {
				if err := bson.UnmarshalExtJSON([]byte(updateAction.Filter), false, &filter); err != nil {
					return nil, err
				}
			}

			update := bson.M{}
			if updateAction.Update != "" {
				if err := bson.UnmarshalExtJSON([]byte(updateAction.Update), false, &update); err != nil {
					return nil, err
				}
			}

			var updateOptions []*options.UpdateOptions
			if updateAction.Options.Upsert != nil {
				updateOptions = append(updateOptions, options.Update().SetUpsert(*updateAction.Options.Upsert))
			}

			updateResult, err := mongoClient.Database(e.Database).Collection(e.Collection).UpdateMany(ctx, filter, update, updateOptions...)
			if err != nil {
				return nil, err
			}

			var result map[string]any
			if err := mapstructure.Decode(updateResult, &result); err != nil {
				return nil, err
			}
			results[i] = result

		case "delete":
			var deleteAction DeleteAction
			if err := mapstructure.Decode(action, &deleteAction); err != nil {
				return nil, err
			}

			filter := bson.M{}
			if deleteAction.Filter != "" {
				if err := bson.UnmarshalExtJSON([]byte(deleteAction.Filter), false, &filter); err != nil {
					return nil, err
				}
			}

			deleteResult, err := mongoClient.Database(e.Database).Collection(e.Collection).DeleteMany(ctx, filter)
			if err != nil {
				return nil, err
			}

			var result map[string]any
			if err := mapstructure.Decode(deleteResult, &result); err != nil {
				return nil, err
			}
			results[i] = result

		case "aggregate":
			var aggregateAction AggregateAction
			if err := mapstructure.Decode(action, &aggregateAction); err != nil {
				return nil, err
			}

			pipeline := bson.A{}
			for _, pipelineItemJSON := range aggregateAction.Pipeline {
				var pipelineItem bson.M
				if err := bson.UnmarshalExtJSON([]byte(pipelineItemJSON), false, &pipelineItem); err != nil {
					return nil, err
				}
				pipeline = append(pipeline, pipelineItem)
			}

			cursor, err := mongoClient.Database(e.Database).Collection(e.Collection).Aggregate(ctx, pipeline)
			if err != nil {
				return nil, err
			}

			var result []map[string]any
			if err := cursor.All(ctx, &result); err != nil {
				return nil, err
			}
			results[i] = map[string]any{"results": result}
		}
	}

	return Result{Actions: results}, nil
}

type Action struct {
	Type string
}

type CountAction struct {
	Action
	Filter  string
	Options struct {
		Limit *int64
	}
}

type LoadFixturesAction struct {
	Action
	Folder string
}

type InsertAction struct {
	Action
	File      string
	Documents []string
}

type FindAction struct {
	Action
	Filter  string
	Options struct {
		Limit      *int64
		Skip       *int64
		Sort       string
		Projection string
	}
}

type UpdateAction struct {
	Action
	Filter  string
	Update  string
	Options struct {
		Upsert *bool
	}
}

type DeleteAction struct {
	Action
	Filter string
}

type AggregateAction struct {
	Action
	Pipeline []string
}
