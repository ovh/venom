// +build !yaml

package tap

import (
	"encoding/json"
)

// yaml serializes a message to YAML.  This implementation uses JSON,
// which is a subset of YAML [1] and is implemented by Go's standard
// library.
//
// [1]: http://www.yaml.org/spec/1.2/spec.html#id2759572
func yaml(message interface{}, prefix string) (marshaled []byte, err error) {
	marshaled, err = json.MarshalIndent(message, prefix, "  ")
	if err != nil {
		return marshaled, err
	}

	marshaled = append(marshaled, []byte("\n")...)
	return marshaled, err
}
