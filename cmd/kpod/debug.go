package main

import (
	"fmt"
	"time"

	"github.com/MohamedBasaleh/kpod/internal/kube"
	"github.com/spf13/cobra"
)

func newDebugCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "debug",
		Short:  "Development diagnostics for kubectl discovery (not the public CLI)",
		Hidden: true,
	}
	cmd.AddCommand(
		newDebugContextsCommand(),
		newDebugNamespacesCommand(),
		newDebugPodsCommand(),
		newDebugCanICommand(),
	)
	return cmd
}

func kubeServiceFromCmd(cmd *cobra.Command) (*kube.Service, error) {
	debug, _ := cmd.Root().PersistentFlags().GetBool("debug")
	opts := []kube.Option{}
	if debug {
		opts = append(opts, kube.WithDebug(func(info kube.DebugInfo) {
			fmt.Fprintf(cmd.ErrOrStderr(), "kubectl args=%q exit=%d duration=%s stderr=%q\n",
				info.Args, info.ExitCode, info.Duration, info.Stderr)
		}))
	}
	return kube.New(opts...)
}

func newDebugContextsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "contexts",
		Short: "List kubeconfig contexts via kubectl",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := kubeServiceFromCmd(cmd)
			if err != nil {
				return err
			}
			list, err := svc.ListContexts(cmd.Context())
			if err != nil {
				return err
			}
			if len(list) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(no contexts)")
				return nil
			}
			for _, c := range list {
				fmt.Fprintln(cmd.OutOrStdout(), c.Name)
			}
			return nil
		},
	}
}

func newDebugNamespacesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "namespaces <context>",
		Short: "List namespaces for a Kubernetes context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := kubeServiceFromCmd(cmd)
			if err != nil {
				return err
			}
			list, err := svc.ListNamespaces(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if len(list) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(no namespaces)")
				return nil
			}
			for _, ns := range list {
				fmt.Fprintln(cmd.OutOrStdout(), ns.Name)
			}
			return nil
		},
	}
}

func newDebugPodsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "pods <context> <namespace>",
		Short: "List pods in a namespace",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := kubeServiceFromCmd(cmd)
			if err != nil {
				return err
			}
			list, err := svc.ListPods(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			if len(list) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(no pods)")
				return nil
			}
			for _, p := range list {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%d/%d\t%s\t%d\t%s\t%s\n",
					p.Name, p.Ready, p.Total, p.Status, p.Restarts, formatAge(p.Age), p.Node)
			}
			return nil
		},
	}
}

func newDebugCanICommand() *cobra.Command {
	return &cobra.Command{
		Use:   "can-i <context> <namespace>",
		Short: "Probe pod get access in a namespace",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := kubeServiceFromCmd(cmd)
			if err != nil {
				return err
			}
			access, err := svc.CanAccessNamespace(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			if access.Allowed {
				fmt.Fprintln(cmd.OutOrStdout(), "accessible")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "not allowed")
			return nil
		},
	}
}

func formatAge(d time.Duration) string {
	return d.Round(time.Second).String()
}
