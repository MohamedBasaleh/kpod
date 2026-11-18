package main

import (
	"os"

	"github.com/MohamedBasaleh/kpod/internal/uninstall"
	"github.com/spf13/cobra"
)

func newUninstallCommand(configDir *string) *cobra.Command {
	var purge bool
	var yes bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the kpod binary (and optionally kpod config)",
		Long: `Remove kpod from this machine.

Default uninstall removes the installed binary from known locations
(~/.local/bin/kpod, ~/bin/kpod) and keeps ~/.config/kpod.

  kpod uninstall --purge

also deletes kpod-owned config, state, generated shortcuts, and
migration backups under the kpod config directory.

kubeconfig, Kubernetes contexts, and AWS credentials are never modified.
Shell rc files are not rewritten; kpod prints any leftover source line.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			paths, err := resolvePaths(*configDir)
			if err != nil {
				return err
			}
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			_, err = uninstall.Execute(uninstall.Options{
				Home:           home,
				Paths:          paths,
				Purge:          purge,
				Yes:            yes,
				ExplicitConfig: *configDir != "" || os.Getenv("KPOD_CONFIG_DIR") != "",
				In:             cmd.InOrStdin(),
				Out:            cmd.OutOrStdout(),
				Err:            cmd.ErrOrStderr(),
			})
			return err
		},
	}
	cmd.Flags().BoolVar(&purge, "purge", false, "also delete kpod config and generated files (not kubeconfig)")
	cmd.Flags().BoolVar(&yes, "yes", false, "do not ask for confirmation")
	return cmd
}
