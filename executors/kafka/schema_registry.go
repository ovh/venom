package kafka

import (
	"fmt"
	"net/http"
	"time"

	schemaregistry "github.com/landoop/schema-registry"
)

type (
	// SchemaRegistry will provide interface to SchemaRegistry implementation
	SchemaRegistry interface {
		GetSchemaByID(id int) (string, error)
		RegisterNewSchema(subject, schema string) (int, error)
		GetLatestSchema(subject string) (int, string, error)
	}

	client struct {
		client *schemaregistry.Client
	}
)

// NewSchemaRegistry will create new Schema Registry interface
func NewSchemaRegistry(schemaRegistryHost string) (SchemaRegistry, error) {
	// Adding new Schema Registry client with http client which has timeout
	return NewWithClient(schemaRegistryHost, &http.Client{Timeout: time.Second * 10})
}

// NewWithClient will add SchemaRegistry with client
func NewWithClient(schemaRegistryHost string, httpClient *http.Client) (SchemaRegistry, error) {
	schemaRegistryClient, err := schemaregistry.NewClient(schemaRegistryHost, schemaregistry.UsingClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to schema registry: %w", err)
	}
	return &client{
		client: schemaRegistryClient,
	}, nil
}

// GetSchemaByID will return schema from SchemaRegistry by it's ID (if it exists there)
func (c client) GetSchemaByID(id int) (string, error) {
	schema, err := c.client.GetSchemaByID(id)
	if err != nil {
		return "", fmt.Errorf("could not get schema id %q from schema registry: %w", id, err)
	}

	return schema, nil
}

// RegisterNewSchema either register a new schema and return the ID or get the ID of an already created schema.
func (c client) RegisterNewSchema(subject, schema string) (int, error) {
	schemaID, err := c.client.RegisterNewSchema(subject, schema)
	if err != nil {
		return 0, fmt.Errorf("failed to register new schema or fetch already created schema ID: %w", err)
	}
	return schemaID, nil
}

// GetLatestSchema gets latest schema identifier from the given subject.
func (c client) GetLatestSchema(subject string) (int, string, error) {
	schema, err := c.client.GetLatestSchema(subject)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get latest schema ID: %w", err)
	}
	
	return schema.ID, schema.Schema, nil
}
