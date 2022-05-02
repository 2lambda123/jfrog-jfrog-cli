package envsetup

import (
	"github.com/jfrog/jfrog-cli-core/v2/general/envsetup"
	"github.com/jfrog/jfrog-cli-core/v2/utils/coreutils"
	"github.com/jfrog/jfrog-cli/utils/progressbar"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

const (
	registrationPageURL = "https://jfrog.com/start-free/cli/"
)

func RunEnvSetupCmd() error {
	log.Output()
	log.Output()
	log.Output(coreutils.PrintTitle("Thank you for installing JFrog CLI! 🐸"))
	log.Output(coreutils.PrintTitle("We'll now set up a FREE JFrog environment in the cloud for you, and configure your local machine to use it."))
	log.Output("Your environment will be ready in less than a minute.")
	setupCmd := envsetup.NewEnvSetupCommand(registrationPageURL)
	return progressbar.ExecWithProgress(setupCmd, false)
}
