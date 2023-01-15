package cmd

import (
	"fmt"
	"github.com/spf13/cobra"

	"github.com/zapier/tfbuddy/internal/logging"
	"github.com/zapier/tfbuddy/pkg/hooks"
)

var enableHookServer bool
var enableHookWorker bool

// tfcHookHandlerCmd represents the run command
var tfcHookHandlerCmd = &cobra.Command{
	Use:   "handler",
	Short: "Start a hooks handler for Gitlab & Terraform cloud.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		logging.SetupLogOutput(resolveLogLevel())
		hooks.StartServer(enableHookWorker, enableHookServer)
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if !enableHookServer && !enableHookWorker {
			return fmt.Errorf("must enable hooks server and/or worker")
		}
		return nil
	},
}

func init() {
	tfcCmd.AddCommand(tfcHookHandlerCmd)
	tfcHookHandlerCmd.PersistentFlags().BoolVar(&enableHookServer, "enable-server", false, "Start with the Hooks Server function")
	tfcHookHandlerCmd.PersistentFlags().BoolVar(&enableHookWorker, "enable-worker", false, "Start with the Hooks Worker function")
}
