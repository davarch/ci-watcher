package cli

import (
	"fmt"

	"github.com/davarch/ci-watcher/internal/infrastructure/config"
	"github.com/spf13/cobra"
)

var enableCmd = &cobra.Command{
	Use:   "enable <project_name>",
	Short: "Enable project by name in config.yaml",
	Args:  cobra.MatchAll(cobra.ExactArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}

		changed := false
		for i := range cfg.Poll.Projects {
			if cfg.Poll.Projects[i].Name == name {
				if !cfg.Poll.Projects[i].Enabled {
					cfg.Poll.Projects[i].Enabled = true
					changed = true
				}
			}
		}

		if !changed {
			fmt.Printf("no change (project %q already enabled or not found)\n", name)
			return nil
		}

		if err := config.Save(cfgPath, cfg); err != nil {
			return err
		}

		fmt.Printf("enabled: %s\n", name)
		return nil
	},
}

func init() {
	enableCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		out := make([]string, 0, len(cfg.Poll.Projects))
		for _, p := range cfg.Poll.Projects {
			if p.Name == "" {
				continue
			}

			if toComplete == "" || startsWith(p.Name, toComplete) {
				out = append(out, p.Name)
			}
		}

		return out, cobra.ShellCompDirectiveNoFileComp
	}

	rootCmd.AddCommand(enableCmd)
}

func startsWith(s, pref string) bool {
	if len(pref) > len(s) {
		return false
	}

	return s[:len(pref)] == pref
}
