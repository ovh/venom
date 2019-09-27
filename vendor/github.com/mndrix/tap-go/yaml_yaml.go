// +build yaml

package tap

import (
	"bytes"

	goyaml "gopkg.in/yaml.v2"
)

// yaml serializes a message to YAML.  This implementation uses
// non-JSON YAML, which has better prove support [1].
//
// [1]: https://rt.cpan.org/Public/Bug/Display.html?id=121606
func yaml(message interface{}, prefix string) (marshaled []byte, err error) {
	marshaled, err = goyaml.Marshal(message)
	if err != nil {
		return marshaled, err
	}

	marshaled = bytes.Replace(marshaled, []byte("\n"), []byte("\n"+prefix), -1)
	return marshaled[:len(marshaled)-len(prefix)], err
}
