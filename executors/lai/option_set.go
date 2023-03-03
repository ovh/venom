package lai

import (
	"git.orcatech.org/infrastructure/data/backend"
	"git.orcatech.org/infrastructure/data/backend/conn/onprem"
	"git.orcatech.org/sdks/golang/config"
)

type OptionSet struct {
	*backend.MicroServiceOptionSet
	*onprem.OnPremisesOptionSet
}

func NewOptionSet() *OptionSet {
	return &OptionSet{
		MicroServiceOptionSet: backend.NewMicroserviceOptionSet(),
		OnPremisesOptionSet:   onprem.NewOptionSet(),
	}
}

func (o *OptionSet) Configs() []config.Sectioner {
	return backend.AppendOptionSetSections(o.MicroServiceOptionSet, o.OnPremisesOptionSet)
}
