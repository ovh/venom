package kafka

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/linkedin/goavro/v2"
)

const (
	magicByte    byte  = 0x0
	schemaIDSize int32 = 4
)

// Convert2Avro will convert value to Avro encoded binary with help of schema
func Convert2Avro(value []byte, schema string) ([]byte, error) {
	// https://github.com/linkedin/goavro
	codec, err := goavro.NewCodec(string(schema))
	if err != nil {
		return nil, fmt.Errorf("failed to create Avro schema: %w", err)
	}
	// Convert textual Avro data (in Avro JSON format) to native Go form
	native, _, err := codec.NativeFromTextual(value)
	if err != nil {
		return nil, fmt.Errorf("failed to convert value %s 2 native Avro: %w", value, err)
	}
	// Convert native Go form to binary Avro data
	binary, err := codec.BinaryFromNative(nil, native)
	if err != nil {
		return nil, fmt.Errorf("failed to convert 2 native Avro: %w", err)
	}
	return binary, nil
}

// ConvertFromAvro will convert value from Avro encoded binary with help of schema to string
func ConvertFromAvro(binary []byte, schema string) (string, error) {
	// https://github.com/linkedin/goavro
	codec, err := goavro.NewCodec(schema)
	if err != nil {
		return "", fmt.Errorf("failed to create Avro schema: %w", err)
	}
	// Convert binary Avro data back to native Go form
	native, _, err := codec.NativeFromBinary(binary)
	if err != nil {
		return "", fmt.Errorf("failed to convert from binary 2 native with Avro schema: %w", err)
	}
	// Convert native Go form to textual Avro data
	textual, err := codec.TextualFromNative(nil, native)
	if err != nil {
		return "", fmt.Errorf("failed to convert from native 2 textual representation: %w", err)
	}
	return string(textual), nil
}

// CreateMessage will convert Avro message to one, which can be sent to Kafka
func CreateMessage(message []byte, schemaID int) ([]byte, error) {
	var value bytes.Buffer

	// Add magic byte.
	_, err := value.Write([]byte{magicByte})
	if err != nil {
		return nil, err
	}

	// Add schema ID.
	schemaIDByte := make([]byte, schemaIDSize)
	binary.BigEndian.PutUint32(schemaIDByte, uint32(schemaID))

	_, err = value.Write(schemaIDByte)
	if err != nil {
		return nil, err
	}

	// Add message.
	_, err = value.Write(message)
	if err != nil {
		return nil, fmt.Errorf("failed to write serialized bytes: %w", err)
	}

	return value.Bytes(), nil
}

// GetMessageAvroID will try to get encoded message Avro ID
func GetMessageAvroID(messageValue []byte) ([]byte, int) {
	// Remove magic byte, get ID and remove ID before deserialisation
	value := bytes.TrimPrefix(messageValue, []byte{magicByte})
	schemaID := int(binary.BigEndian.Uint32(value[:schemaIDSize]))
	value = value[schemaIDSize:]

	return value, schemaID
}
