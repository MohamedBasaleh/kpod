package main

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/shortcut"
	"github.com/spf13/cobra"
)

func newShortcutCommand(configDir *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shortcut",
		Short: "Manage optional shell shortcuts (k-dc, k-sc, …)",
	}
	cmd.AddCommand(
		shortcutListCmd(configDir),
		shortcutAddCmd(configDir),
		shortcutRemoveCmd(configDir),
		shortcutEnableCmd(configDir),
		shortcutDisableCmd(configDir),
		shortcutGenerateCmd(configDir),
		shortcutInstallCmd(configDir),
		shortcutUninstallCmd(configDir),
		shortcutStatusCmd(configDir),
	)
	return cmd
}

func shortcutSvc(configDir *string) (shortcut.Service, error) {
	paths, err := resolvePaths(*configDir)
	if err != nil {
		return shortcut.Service{}, err
	}
	return shortcut.New(paths), nil
}

func loadShortcutCfg(s shortcut.Service) (config.Config, error) {
	cfg, err := s.Load()
	if err != nil {
		return cfg, fmt.Errorf("could not load kpod configuration: %w", err)
	}
	return cfg, nil
}

func shortcutListCmd(configDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured shortcuts",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := shortcutSvc(configDir)
			if err != nil {
				return err
			}
			cfg, err := loadShortcutCfg(s)
			if err != nil {
				return err
			}
			ids := shortcut.SortedIDs(cfg)
			if len(ids) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No shortcuts configured.\n\nTry: kpod shortcut add dc dev console")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%-10s  %s\n", "NAME", "COMMAND")
			for _, id := range ids {
				sc := cfg.Shortcuts[id]
				line := fmt.Sprintf("%-10s  %s", shortcut.CommandName(id), shortcut.Invocation(sc))
				if err := shortcut.CheckRefs(cfg, id, sc); err != nil {
					line += "  (" + err.Error() + ")"
				}
				fmt.Fprintln(cmd.OutOrStdout(), line)
			}
			return nil
		},
	}
}

func shortcutAddCmd(configDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "add <id> <environment> <action>",
		Short: "Add a shortcut (stored without the k- prefix)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := shortcutSvc(configDir)
			if err != nil {
				return err
			}
			cfg, err := loadShortcutCfg(s)
			if err != nil {
				return err
			}
			cfg, err = s.Add(cfg, args[0], args[1], args[2])
			if err != nil {
				return err
			}
			if err := s.Save(cfg); err != nil {
				return err
			}
			if cfg.Settings.ShellShortcuts {
				_ = s.Generate(cfg)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added %s → %s\n", shortcut.CommandName(args[0]), shortcut.Invocation(cfg.Shortcuts[args[0]]))
			return nil
		},
	}
}

func shortcutRemoveCmd(configDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a shortcut",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := shortcutSvc(configDir)
			if err != nil {
				return err
			}
			cfg, err := loadShortcutCfg(s)
			if err != nil {
				return err
			}
			cfg, err = s.Remove(cfg, args[0])
			if err != nil {
				return err
			}
			if err := s.Save(cfg); err != nil {
				return err
			}
			if cfg.Settings.ShellShortcuts {
				_ = s.Generate(cfg)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed %s\n", shortcut.CommandName(args[0]))
			return nil
		},
	}
}

func shortcutEnableCmd(configDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable shell shortcuts and regenerate alias files",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := shortcutSvc(configDir)
			if err != nil {
				return err
			}
			cfg, err := loadShortcutCfg(s)
			if err != nil {
				return err
			}
			cfg = s.Enable(cfg)
			if err := s.Save(cfg); err != nil {
				return err
			}
			if err := s.Generate(cfg); err != nil {
				return err
			}
			st := s.Status(cfg)
			fmt.Fprintf(cmd.OutOrStdout(), "Shortcuts enabled.\nWrote %s\n", st.AliasFile)
			if !st.Installed {
				fmt.Fprintf(cmd.OutOrStdout(), "\nTo load them in your shell, run:\n  kpod shortcut install\n")
			}
			return nil
		},
	}
}

func shortcutDisableCmd(configDir *string) *cobra.Command {
	var uninstall bool
	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable shell shortcuts (definitions are kept)",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := shortcutSvc(configDir)
			if err != nil {
				return err
			}
			cfg, err := loadShortcutCfg(s)
			if err != nil {
				return err
			}
			cfg = s.Disable(cfg)
			if err := s.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Shortcuts disabled. Definitions remain in config.yaml.")
			if uninstall {
				yes, _ := cmd.Flags().GetBool("yes")
				if err := s.UninstallRC(yes || confirm(cmd, s, false)); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Removed the kpod source line from your shell rc file.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&uninstall, "remove-source", false, "also remove the managed source line from the shell rc file")
	cmd.Flags().Bool("yes", false, "confirm rc edits without prompting")
	return cmd
}

func shortcutGenerateCmd(configDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Regenerate kpod-owned alias files from config",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := shortcutSvc(configDir)
			if err != nil {
				return err
			}
			cfg, err := loadShortcutCfg(s)
			if err != nil {
				return err
			}
			if err := s.Generate(cfg); err != nil {
				return err
			}
			st := s.Status(cfg)
			fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", st.AliasFile)
			return nil
		},
	}
}

func shortcutInstallCmd(configDir *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Add one managed source line to your shell rc file",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := shortcutSvc(configDir)
			if err != nil {
				return err
			}
			cfg, err := loadShortcutCfg(s)
			if err != nil {
				return err
			}
			if err := s.Generate(cfg); err != nil {
				return err
			}
			yes, _ := cmd.Flags().GetBool("yes")
			if !yes && !confirm(cmd, s, true) {
				fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
				return nil
			}
			if err := s.InstallRC(true); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Installed. Open a new shell or source your rc file.")
			return nil
		},
	}
	cmd.Flags().Bool("yes", false, "add the source line without prompting")
	return cmd
}

func shortcutUninstallCmd(configDir *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the managed source line from your shell rc file",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := shortcutSvc(configDir)
			if err != nil {
				return err
			}
			yes, _ := cmd.Flags().GetBool("yes")
			if !yes && !confirm(cmd, s, false) {
				fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
				return nil
			}
			if err := s.UninstallRC(true); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Removed the kpod source line.")
			return nil
		},
	}
	cmd.Flags().Bool("yes", false, "remove the source line without prompting")
	return cmd
}

func shortcutStatusCmd(configDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show shortcut enablement and shell integration",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := shortcutSvc(configDir)
			if err != nil {
				return err
			}
			cfg, err := loadShortcutCfg(s)
			if err != nil {
				return err
			}
			st := s.Status(cfg)
			fmt.Fprintf(cmd.OutOrStdout(), "Enabled:    %v\n", st.Enabled)
			fmt.Fprintf(cmd.OutOrStdout(), "Shell:      %s\n", st.Shell)
			fmt.Fprintf(cmd.OutOrStdout(), "Shortcuts:  %d\n", st.Count)
			fmt.Fprintf(cmd.OutOrStdout(), "Alias file: %s\n", st.AliasFile)
			fmt.Fprintf(cmd.OutOrStdout(), "RC file:    %s\n", st.RCPath)
			fmt.Fprintf(cmd.OutOrStdout(), "Installed:  %v\n", st.Installed)
			for _, p := range st.Problems {
				fmt.Fprintf(cmd.OutOrStdout(), "Problem:    %s\n", p)
			}
			return nil
		},
	}
}

func confirm(cmd *cobra.Command, s shortcut.Service, install bool) bool {
	st := s.Status(config.Default())
	st.RCPath = shortcut.RCPath(s.Home, s.Shell)
	_, alias, snippet := shortcut.InstallPlan(s.Paths, s.Home, s.Shell)
	_ = alias
	if install {
		fmt.Fprintf(cmd.OutOrStdout(), "kpod needs to add one line to %s:\n\n%s\nAdd it? [Y/n] ", st.RCPath, strings.TrimSpace(snippet))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Remove the kpod source line from %s? [y/N] ", st.RCPath)
	}
	in := cmd.InOrStdin()
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return false
	}
	ans := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if install {
		return ans == "" || ans == "y" || ans == "yes"
	}
	return ans == "y" || ans == "yes"
}
