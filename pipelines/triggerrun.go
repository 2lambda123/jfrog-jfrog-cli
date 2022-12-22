package pipelines

import (
	status "github.com/jfrog/jfrog-cli-core/v2/pipelines/commands"
	"github.com/jfrog/jfrog-cli-core/v2/utils/coreutils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	clientlog "github.com/jfrog/jfrog-client-go/utils/log"
	"github.com/urfave/cli"
)

// triggerNewRun triggers a new run for supplied flag values
func triggerNewRun(c *cli.Context) error {
	// read flags for trigger run command
	b := c.String("branch")
	p := c.String("pipelineName")
	s := c.String("server-id")
	multiBranch := getMultiBranch(c)

	coreutils.PrintTitle("🐸🐸🐸 triggering pipeline run ")
	clientlog.Output("triggering pipelineName", p, "for branch ", b)

	// get service config details
	serviceDetails, servErr := getServiceDetails(s)
	if servErr != nil {
		return errorutils.CheckError(servErr)
	}

	tc := status.NewTriggerCommand()
	tc.SetBranch(b).
		SetPipeline(p).
		SetServerDetails(serviceDetails).
		SetMultiBranch(multiBranch)

	run, runErr := tc.Run()
	if runErr != nil {
		return errorutils.CheckError(runErr)
	}
	clientlog.Output(run)
	return nil
}
