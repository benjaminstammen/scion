package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/ptone/scion-agent/pkg/config"
	"github.com/ptone/scion-agent/pkg/runtime"
	"github.com/spf13/cobra"
)

var (
	listAll bool
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List running scion agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		rt := runtime.GetRuntime()
		filters := map[string]string{
			"scion.agent": "true",
		}

		if !listAll {
			projectDir, err := config.GetResolvedProjectDir(grovePath)
			if err != nil {
				return err
			}
			groveName := config.GetGroveName(projectDir)
			filters["scion.grove"] = groveName
		}

		agents, err := rt.List(context.Background(), filters)
		if err != nil {
			return err
		}

		if len(agents) == 0 {
			if listAll {
				fmt.Println("No active agents found across any groves.")
			} else {
				fmt.Println("No active agents found in the current grove.")
			}
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tGROVE\tAGENT STATUS\tCONTAINER\tID\tIMAGE")
		for _, a := range agents {
			agentStatus := "unknown"
			if a.GrovePath != "" {
				// agent home: <GrovePath>/agents/<AgentName>/home/scion.json
				agentScionJSON := filepath.Join(a.GrovePath, "agents", a.Name, "home", "scion.json")
				data, err := os.ReadFile(agentScionJSON)
				if err == nil {
					var cfg config.ScionConfig
					if err := json.Unmarshal(data, &cfg); err == nil && cfg.Agent != nil {
						agentStatus = cfg.Agent.Status
						if agentStatus == "" {
							agentStatus = "IDLE"
						}
					}
				}
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", a.Name, a.Grove, agentStatus, a.Status, a.ID, a.Image)
		}
		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "List all agents across all groves")
}

