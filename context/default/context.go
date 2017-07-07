package defaultctx

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/runabove/venom"
)

// Context Type name
const Name = "default"

// New returns a new TestCaseContext
func New() venom.TestCaseContext {
	ctx := &DefaultTestCaseContext{}
	ctx.Name = Name
	return ctx
}

// TestCaseContext represents the context of a testcase
type DefaultTestCaseContext struct {
	venom.CommonTestCaseContext
	datas map[string]interface{}
}

// Init Initialize the context
func (tcc *DefaultTestCaseContext) Init() error {
	tcc.datas = tcc.TestCase.Context
	return nil
}

// Close the context
func (tcc *DefaultTestCaseContext) Close() error {
	return nil
}

func (tcc *DefaultTestCaseContext) GetString(key string) (string, error) {
	if tcc.datas[key] == nil {
		return "", NotFound(key)
	}

	if result, ok := tcc.datas[key].(string); !ok {
		return "", errors.New(fmt.Sprintf("arguments %s invalid", key))
	} else {
		return result, nil
	}
}

func (tcc *DefaultTestCaseContext) GetFloat(key string) (float64, error) {
	if tcc.datas[key] == nil {
		return 0, NotFound(key)
	}

	if result, ok := tcc.datas[key].(float64); !ok {
		return 0, errors.New(fmt.Sprintf("arguments %s invalid", key))
	} else {
		return result, nil
	}
}

func (tcc *DefaultTestCaseContext) GetInt(key string) (int, error) {
	res, err := tcc.GetFloat(key)
	if err != nil {
		return 0, err
	}

	return int(res), nil
}

func (tcc *DefaultTestCaseContext) GetBool(key string) (bool, error) {
	if tcc.datas[key] == nil {
		return false, NotFound(key)
	}

	if result, ok := tcc.datas[key].(bool); !ok {
		return false, errors.New(fmt.Sprintf("arguments %s invalid", key))
	} else {
		return result, nil
	}
}

func (tcc *DefaultTestCaseContext) GetStringSlice(key string) ([]string, error) {
	if tcc.datas[key] == nil {
		return nil, NotFound(key)
	}

	stringSlice, ok := tcc.datas[key].([]string)
	if ok {
		return stringSlice, nil
	}

	slice, ok := tcc.datas[key].([]interface{})
	if !ok {
		return nil, errors.New(fmt.Sprintf("arguments %s invalid", key))
	}

	res := make([]string, len(slice))

	for k, v := range slice {
		s, ok := v.(string)
		if !ok {
			return nil, errors.New("cannot cast to string")
		}

		res[k] = s
	}

	return res, nil
}

func (tcc *DefaultTestCaseContext) GetComplex(key string, arg interface{}) error {
	if tcc.datas[key] == nil {
		return NotFound(key)
	}

	val, err := json.Marshal(tcc.datas[key])
	if err != nil {
		return err
	}

	err = json.Unmarshal(val, arg)
	if err != nil {
		return err
	}
	return nil
}

// NotFound is error returned when trying to get missing argument
func NotFound(key string) error { return fmt.Errorf("missing context argument '%s'", key) }
