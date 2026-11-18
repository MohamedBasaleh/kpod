package main

import (
	"fmt"

	"github.com/MohamedBasaleh/kpod/internal/cli"
	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/migration"
	"github.com/MohamedBasaleh/kpod/internal/tui"
	"github.com/MohamedBasaleh/kpod/internal/update"
	"github.com/MohamedBasaleh/kpod/internal/version"
	"github.com/spf13/cobra"
)

func newRootCommand() *cobra.Command {
	var configDir string
	var doUpdate bool

	cmd := &cobra.Command{
		Use:   "kpod [environment] [action] [pod]",
		Short: "Fast, interactive Kubernetes access for developers",
		Long: `kpod is a Kubernetes developer CLI.

An environment is a Kubernetes context + namespace with a short name
(for example: dev → my-company-dev / dev-core).

kpod never changes kubectl's global current-context.

Examples:
  kpod
  kpod dev
  kpod dev console
  kpod staging logs api
  kpod dev bash
  kpod shortcut generate
  kpod discover
  kpod uninstall
  kpod uninstall --purge
`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeRoot(configDir, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if doUpdate {
				_, err := update.Run(update.Options{Stdout: cmd.OutOrStdout(), Stderr: cmd.ErrOrStderr()})
				return err
			}
			paths, err := resolvePaths(configDir)
			if err != nil {
				return err
			}
			debug, _ := cmd.Flags().GetBool("debug")
			inv, err := cli.Parse(args)
			if err != nil {
				return err
			}
			if inv.BrowseOnly {
				return tui.Start(tui.Options{
					Paths:    paths,
					Out:      cmd.OutOrStdout(),
					Env:      inv.Env,
					Debug:    debug,
					OpenPods: inv.OpenPods,
				})
			}
			return cli.Run(cli.Options{
				Paths: paths,
				Inv:   inv,
				Out:   cmd.OutOrStdout(),
				Err:   cmd.ErrOrStderr(),
				Debug: debug,
			})
		},
	}
	cmd.Version = version.Display()
	cmd.SetVersionTemplate("kpod {{.Version}}\n")
	cmd.PersistentFlags().StringVar(&configDir, "config-dir", "", "kpod configuration directory (default: $KPOD_CONFIG_DIR or ~/.config/kpod)")
	cmd.PersistentFlags().Bool("debug", false, "print kubectl invocation diagnostics (no credentials)")
	cmd.Flags().BoolVar(&doUpdate, "update", false, "update kpod from the latest GitHub Release")

	cmd.AddCommand(newVersionCommand())
	cmd.AddCommand(newMigrateCommand(&configDir))
	cmd.AddCommand(newShortcutCommand(&configDir))
	cmd.AddCommand(newDiscoverCommand(&configDir))
	cmd.AddCommand(newUninstallCommand(&configDir))
	cmd.AddCommand(newDebugCommand())
	cmd.CompletionOptions.HiddenDefaultCmd = false
	return cmd
}

func newVersionCommand() *cobra.Command {
	var verbose bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print kpod version",
		RunE: func(cmd *cobra.Command, args []string) error {
			if verbose {
				fmt.Fprint(cmd.OutOrStdout(), version.Verbose())
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), version.Line())
			return nil
		},
	}
	cmd.Flags().BoolVar(&verbose, "verbose", false, "include commit and build date")
	return cmd
}

func newMigrateCommand(configDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Migrate legacy Bash contexts file to v2 config.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			paths, err := resolvePaths(*configDir)
			if err != nil {
				return err
			}
			result, err := migration.Migrate(paths)
			if err != nil {
				return err
			}
			if result.Skipped != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "migration skipped: %s\n", result.Skipped)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "migrated legacy contexts to %s\n", paths.ConfigFile())
			if result.Backup != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "backup: %s\n", result.Backup)
			}
			for ctx, c := range result.Config.Contexts {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", ctx)
				for alias, env := range c.Environments {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s → %s\n", alias, env.Namespace)
				}
			}
			return nil
		},
	}
}

func resolvePaths(configDir string) (config.Paths, error) {
	if configDir != "" {
		return config.NewPaths(configDir), nil
	}
	return config.DefaultPaths()
}
