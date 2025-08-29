package cli

import (
	"fmt"

	"github.com/davarch/ci-watcher/internal/infrastructure/config"
	"github.com/spf13/cobra"
)

var disableCmd = &cobra.Command{
	Use:   "disable <project_name>",
	Short: "Disable project by name in config.yaml",
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
				if cfg.Poll.Projects[i].Enabled {
					cfg.Poll.Projects[i].Enabled = false
					changed = true
				}
			}
		}

		if !changed {
			fmt.Printf("no change (project %q already disabled or not found)\n", name)
			return nil
		}

		if err := config.Save(cfgPath, cfg); err != nil {
			return err
		}
		fmt.Printf("disabled: %s\n", name)

		return nil
	},
}

func init() {
	disableCmd.ValidArgsFunction = enableCmd.ValidArgsFunction

	rootCmd.AddCommand(disableCmd)
}
