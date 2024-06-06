package nettests

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

// STUNReachability nettest implementation.
type STUNReachability struct{}

func (n STUNReachability) lookupURLs(ctl *Controller) ([]model.ExperimentTarget, error) {
	targetloader := &targetloading.Loader{
		CheckInConfig: &model.OOAPICheckInConfig{
			// not needed because we have default static input in the engine
		},
		ExperimentName: "stunreachability",
		InputPolicy:    model.InputOrStaticDefault,
		Session:        ctl.Session,
		SourceFiles:    ctl.InputFiles,
		StaticInputs:   ctl.Inputs,
	}
	testlist, err := targetloader.Load(context.Background())
	if err != nil {
		return nil, err
	}
	return ctl.BuildAndSetInputIdxMap(testlist)
}

// Run starts the nettest.
func (n STUNReachability) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("stunreachability")
	if err != nil {
		return err
	}
	urls, err := n.lookupURLs(ctl)
	if err != nil {
		return err
	}
	return ctl.Run(builder, urls)
}
