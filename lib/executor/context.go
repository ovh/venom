package executor

import (
	"context"
	"os"
	"strings"

	"github.com/ovh/venom"
)

var _ venom.TestContext = new(executorContext)

type executorContext struct {
	context.Context
	bag              venom.HH
	workingDirectory string
}

func NewContextFromEnv() venom.TestContext {
	wd, _ := os.Getwd()
	ctx := executorContext{
		Context:          context.Background(),
		bag:              venom.HH{},
		workingDirectory: wd,
	}

	environ := os.Environ()
	for _, env := range environ {
		splittedEnv := strings.SplitN(env, "=", 2)
		k := splittedEnv[0]
		v := splittedEnv[1]
		if strings.HasPrefix(k, "VENOM_CTX_") {
			k = strings.Replace(k, "VENOM_CTX_", "", 1)
			log.Debugf("context value: %s:%s", k, v)
			ctx.bag.Add(k, v)
		}
	}

	return &ctx
}

func (e *executorContext) Bag() venom.HH {
	return e.bag
}

func (e *executorContext) SetWorkingDirectory(s string) {
	e.workingDirectory = s
}
func (e *executorContext) GetWorkingDirectory() string {
	return e.workingDirectory
}
