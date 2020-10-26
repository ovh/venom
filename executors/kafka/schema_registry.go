package kafka

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net/http"
	"strconv"
	"time"

	schemaregistry "github.com/landoop/schema-registry"
	cache "github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
)

const (
	magicByte    byte  = 0x0
	schemaIDSize int32 = 4
)

type (
	SchemaRegistry interface {
		GetSchemaByID(id int) (string, error)
		RegisterNewSchema(subject, schema string) (int, error)
		IsSchemaCached(id int) bool
	}

	client struct {
		client *schemaregistry.Client
		cache  *cache.Cache
	}
)

func NewSchemaRegistry(schemaRegistryHost string) (SchemaRegistry, error) {
	return NewWithClient(schemaRegistryHost, &http.Client{})
}

func NewWithClient(schemaRegistryHost string, httpClient *http.Client) (SchemaRegistry, error) {
	schemaRegistryClient, err := schemaregistry.NewClient(schemaRegistryHost, schemaregistry.UsingClient(httpClient))
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to schema registry")
	}

	c := cache.New(10*time.Minute, 20*time.Minute)

	return &client{
		client: schemaRegistryClient,
		cache:  c,
	}, nil
}

func (c client) GetSchemaByID(id int) (string, error) {
	idString := strconv.Itoa(id)
	cachedSchema, found := c.cache.Get(idString)
	if found {
		return cachedSchema.(string), nil
	}

	schema, err := c.client.GetSchemaByID(id)
	if err != nil {
		return "", errors.Wrapf(err, "could not get schema id %q from schema registry", id)
	}

	c.cache.Set(idString, schema, cache.DefaultExpiration)

	return schema, nil
}

// RegisterNewSchema either register a new schema and return the ID or get the ID of an already created schema.
func (c client) RegisterNewSchema(topic, schema string) (int, error) {
	cacheKey := fmt.Sprintf("%s:%s", topic, schema)

	cachedSchemaID, found := c.cache.Get(cacheKey) //NOTE: Not thread safe
	if found {
		return cachedSchemaID.(int), nil
	}

	schemaID, err := c.client.RegisterNewSchema(topic, schema)
	if err != nil {
		return 0, errors.Wrap(err, "failed to register new schema or fetch already created schema ID")
	}

	c.cache.Set(cacheKey, schemaID, cache.DefaultExpiration)

	return schemaID, nil
}

// IsSchemaCached would check if schema with id is already in our cache
func (c client) IsSchemaCached(id int) bool {
	if _, found := c.cache.Get(strconv.Itoa(id)); !found {
		return false
	}
	return true
}

func getMessageByte(messageValue []byte) (value []byte, schemaID int) {
	// Remove magic byte, get ID and remove ID before deserialisation
	value = bytes.TrimPrefix(messageValue, []byte{magicByte})
	schemaID = int(binary.BigEndian.Uint32(value[:schemaIDSize]))
	value = value[schemaIDSize:]

	return
}
