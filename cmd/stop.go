package cmd

import (
	"context"
	"fmt"

	"github.com/ptone/scion-agent/pkg/runtime"
	"github.com/spf13/cobra"
)

var stopRm bool

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop <agent>",
	Short: "Stop an agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]
		rt := runtime.GetRuntime()
		
		fmt.Printf("Stopping agent '%s'...\n", agentName)
		if err := rt.Stop(context.Background(), agentName); err != nil {
			return err
		}

		if stopRm {
			if err := rt.Delete(context.Background(), agentName); err != nil {
				return err
			}
			if err := DeleteAgentFiles(agentName); err != nil {
				return err
			}
			fmt.Printf("Agent '%s' stopped and removed.\n", agentName)
		} else {
			fmt.Printf("Agent '%s' stopped.\n", agentName)
		}
		
		return nil
	},
}

func init() {
	stopCmd.Flags().BoolVar(&stopRm, "rm", false, "Remove the agent after stopping")
	rootCmd.AddCommand(stopCmd)
}

