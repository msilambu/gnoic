package app

import (
	"context"
	"fmt"
	"os"

	"github.com/karimra/gnoic/api"
	"github.com/olekukonko/tablewriter"
	"github.com/openconfig/gnoi/containerz"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// StartPlugin

func (a *App) InitContainerzStartPluginFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.Flags().StringVar(&a.Config.ContainerzPluginStartName, "name", "", "plugin file name as deployed (required)")
	cmd.Flags().StringVar(&a.Config.ContainerzPluginStartInstanceName, "instance-name", "", "name for the running plugin instance (required)")
	cmd.Flags().StringVar(&a.Config.ContainerzPluginStartConfig, "config", "", "JSON configuration string for the plugin")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

func (a *App) RunEContainerzStartPlugin(cmd *cobra.Command, args []string) error {
	if a.Config.ContainerzPluginStartName == "" {
		return fmt.Errorf("--name is required")
	}
	if a.Config.ContainerzPluginStartInstanceName == "" {
		return fmt.Errorf("--instance-name is required")
	}
	return a.containerzDo(func(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
		return a.containerzStartPlugin(ctx, t, c)
	})
}

func (a *App) containerzStartPlugin(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
	req := &containerz.StartPluginRequest{
		Name:         a.Config.ContainerzPluginStartName,
		InstanceName: a.Config.ContainerzPluginStartInstanceName,
		Config:       a.Config.ContainerzPluginStartConfig,
	}
	rsp, err := c.StartPlugin(ctx, req)
	if err != nil {
		return fmt.Errorf("StartPlugin RPC failed: %v", err)
	}
	a.printMsg(t.Config.Address, rsp)
	fmt.Printf("[%s] Plugin started: instance_name=%s\n", t.Config.Address, rsp.InstanceName)
	return nil
}

// StopPlugin

func (a *App) InitContainerzStopPluginFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.Flags().StringVar(&a.Config.ContainerzPluginStopInstanceName, "instance-name", "", "plugin instance name to stop (required)")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

func (a *App) RunEContainerzStopPlugin(cmd *cobra.Command, args []string) error {
	if a.Config.ContainerzPluginStopInstanceName == "" {
		return fmt.Errorf("--instance-name is required")
	}
	return a.containerzDo(func(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
		return a.containerzStopPlugin(ctx, t, c)
	})
}

func (a *App) containerzStopPlugin(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
	req := &containerz.StopPluginRequest{
		InstanceName: a.Config.ContainerzPluginStopInstanceName,
	}
	rsp, err := c.StopPlugin(ctx, req)
	if err != nil {
		return fmt.Errorf("StopPlugin RPC failed: %v", err)
	}
	a.printMsg(t.Config.Address, rsp)
	fmt.Printf("[%s] Plugin %q stopped\n", t.Config.Address, a.Config.ContainerzPluginStopInstanceName)
	return nil
}

// ListPlugins

func (a *App) InitContainerzListPluginsFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.Flags().StringVar(&a.Config.ContainerzPluginListInstanceName, "instance-name", "", "filter by plugin instance name (optional)")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

func (a *App) RunEContainerzListPlugins(cmd *cobra.Command, args []string) error {
	return a.containerzDo(func(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
		return a.containerzListPlugins(ctx, t, c)
	})
}

func (a *App) containerzListPlugins(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
	req := &containerz.ListPluginsRequest{
		InstanceName: a.Config.ContainerzPluginListInstanceName,
	}
	rsp, err := c.ListPlugins(ctx, req)
	if err != nil {
		return fmt.Errorf("ListPlugins RPC failed: %v", err)
	}
	a.printMsg(t.Config.Address, rsp)

	if len(rsp.Plugins) == 0 {
		fmt.Printf("[%s] No plugins found\n", t.Config.Address)
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"TARGET", "ID", "INSTANCE NAME", "CONFIG"})
	formatTable(table)
	for _, p := range rsp.Plugins {
		table.Append([]string{t.Config.Address, p.Id, p.InstanceName, p.Config})
	}
	table.Render()
	return nil
}

// RemovePlugin

func (a *App) InitContainerzRemovePluginFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.Flags().StringVar(&a.Config.ContainerzPluginRemoveInstanceName, "instance-name", "", "plugin instance name to remove (required)")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

func (a *App) RunEContainerzRemovePlugin(cmd *cobra.Command, args []string) error {
	if a.Config.ContainerzPluginRemoveInstanceName == "" {
		return fmt.Errorf("--instance-name is required")
	}
	return a.containerzDo(func(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
		return a.containerzRemovePlugin(ctx, t, c)
	})
}

func (a *App) containerzRemovePlugin(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
	req := &containerz.RemovePluginRequest{
		InstanceName: a.Config.ContainerzPluginRemoveInstanceName,
	}
	rsp, err := c.RemovePlugin(ctx, req)
	if err != nil {
		return fmt.Errorf("RemovePlugin RPC failed: %v", err)
	}
	a.printMsg(t.Config.Address, rsp)
	fmt.Printf("[%s] Plugin %q removed\n", t.Config.Address, a.Config.ContainerzPluginRemoveInstanceName)
	return nil
}
