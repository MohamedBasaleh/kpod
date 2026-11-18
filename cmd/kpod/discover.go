package main

import (
	"context"
	"fmt"
	"time"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/discover"
	"github.com/MohamedBasaleh/kpod/internal/kube"
	"github.com/MohamedBasaleh/kpod/internal/state"
	"github.com/spf13/cobra"
)

func newDiscoverCommand(configDir *string) *cobra.Command {
	var refresh bool
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Show newly visible Kubernetes contexts and namespaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			paths, err := resolvePaths(*configDir)
			if err != nil {
				return err
			}
			cfg, err := config.NewRepository(paths).Load()
			if err != nil {
				return fmt.Errorf("could not load kpod configuration: %w", err)
			}
			st, err := state.NewRepository(paths).LoadOrDefault()
			if err != nil {
				return err
			}
			svc, err := kube.New()
			if err != nil {
				fmt.Fprintln(cmd.OutOrStdout(), "Unable to check Kubernetes updates.")
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 25*time.Second)
			defer cancel()
			out := discover.Run(ctx, svc, cfg, st, discover.Options{Refresh: refresh})
			if out.StateDirty {
				_ = state.NewRepository(paths).Mutate(func(cur *state.State) error {
					cur.DiscoveryReady = out.State.DiscoveryReady
					cur.SeenContexts = out.State.SeenContexts
					cur.SeenNamespaces = out.State.SeenNamespaces
					return nil
				})
			}
			fmt.Fprint(cmd.OutOrStdout(), discover.Format(out.Result))
			if out.Result.ContextsErr != nil && !out.Result.Empty() {
				return nil
			}
			if out.Result.ContextsErr != nil {
				return out.Result.ContextsErr
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&refresh, "refresh", false, "bypass kubectl result cache")
	cmd.AddCommand(discoverIgnoreContextCmd(configDir), discoverIgnoreNamespaceCmd(configDir))
	return cmd
}

func discoverIgnoreContextCmd(configDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "ignore-context <name>",
		Short: "Stop notifying about a Kubernetes context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			paths, err := resolvePaths(*configDir)
			if err != nil {
				return err
			}
			repo := config.NewRepository(paths)
			cfg, err := repo.Load()
			if err != nil {
				return fmt.Errorf("could not load kpod configuration: %w", err)
			}
			cfg = config.IgnoreContext(cfg, args[0])
			if err := repo.Save(cfg); err != nil {
				return err
			}
			_ = state.NewRepository(paths).Mutate(func(st *state.State) error {
				state.MarkContextSeen(st, args[0])
				return nil
			})
			fmt.Fprintf(cmd.OutOrStdout(), "Ignored context %s.\n", args[0])
			return nil
		},
	}
}

func discoverIgnoreNamespaceCmd(configDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "ignore-namespace <context> <namespace>",
		Short: "Stop notifying about a namespace in a context",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			paths, err := resolvePaths(*configDir)
			if err != nil {
				return err
			}
			repo := config.NewRepository(paths)
			cfg, err := repo.Load()
			if err != nil {
				return fmt.Errorf("could not load kpod configuration: %w", err)
			}
			cfg = config.IgnoreNamespace(cfg, args[0], args[1])
			if err := repo.Save(cfg); err != nil {
				return err
			}
			_ = state.NewRepository(paths).Mutate(func(st *state.State) error {
				state.MarkNamespaceSeen(st, args[0], args[1])
				return nil
			})
			fmt.Fprintf(cmd.OutOrStdout(), "Ignored namespace %s in %s.\n", args[1], args[0])
			return nil
		},
	}
}
