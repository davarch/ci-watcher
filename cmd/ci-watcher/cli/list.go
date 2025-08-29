package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/davarch/ci-watcher/internal/infrastructure/config"
	"github.com/spf13/cobra"
)

var (
	listOnlyEnabled  bool
	listOnlyDisabled bool
	listJSON         bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects from config.yaml",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}

		items := make([]config.Project, 0, len(cfg.Poll.Projects))
		for _, p := range cfg.Poll.Projects {
			if listOnlyEnabled && !p.Enabled {
				continue
			}
			if listOnlyDisabled && p.Enabled {
				continue
			}
			items = append(items, p)
		}

		if listJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(items)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "NAME\tPROJECT_ID\tREF\tENABLED")
		for _, p := range items {
			name := p.Name
			if name == "" {
				name = "(unnamed)"
			}
			en := "false"
			if p.Enabled {
				en = "true"
			}
			_, _ = fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", name, p.ProjectID, p.Ref, en)
		}
		_ = w.Flush()
		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&listOnlyEnabled, "enabled", false, "show only enabled projects")
	listCmd.Flags().BoolVar(&listOnlyDisabled, "disabled", false, "show only disabled projects")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "print JSON")

	listCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if listOnlyEnabled && listOnlyDisabled {
			return fmt.Errorf("flags --enabled and --disabled are mutually exclusive")
		}
		return nil
	}

	rootCmd.AddCommand(listCmd)
}
