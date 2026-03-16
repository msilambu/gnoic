package app

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/karimra/gnoic/api"
	"github.com/olekukonko/tablewriter"
	"github.com/openconfig/gnoi/containerz"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// CreateVolume

func (a *App) InitContainerzCreateVolumeFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.Flags().StringVar(&a.Config.ContainerzVolumeCreateName, "name", "", "volume name (required)")
	cmd.Flags().StringVar(&a.Config.ContainerzVolumeCreateDriver, "driver", "local", "volume driver: local|custom")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzVolumeCreateLabels, "label", []string{}, "labels in KEY=VALUE format (repeatable)")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

func (a *App) RunEContainerzCreateVolume(cmd *cobra.Command, args []string) error {
	if a.Config.ContainerzVolumeCreateName == "" {
		return fmt.Errorf("--name is required")
	}
	return a.containerzDo(func(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
		return a.containerzCreateVolume(ctx, t, c)
	})
}

func (a *App) containerzCreateVolume(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
	req := &containerz.CreateVolumeRequest{
		Name:   a.Config.ContainerzVolumeCreateName,
		Labels: parseKVMap(a.Config.ContainerzVolumeCreateLabels),
	}

	switch a.Config.ContainerzVolumeCreateDriver {
	case "custom":
		req.Driver = containerz.Driver_DS_CUSTOM
	default:
		req.Driver = containerz.Driver_DS_LOCAL
	}

	rsp, err := c.CreateVolume(ctx, req)
	if err != nil {
		return fmt.Errorf("CreateVolume RPC failed: %v", err)
	}
	a.printMsg(t.Config.Address, rsp)
	fmt.Printf("[%s] Volume created: %s\n", t.Config.Address, rsp.Name)
	return nil
}

// ListVolume

func (a *App) InitContainerzListVolumeFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.Flags().StringSliceVar(&a.Config.ContainerzVolumeListFilter, "filter", []string{}, "filters in key=value format (repeatable)")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

func (a *App) RunEContainerzListVolume(cmd *cobra.Command, args []string) error {
	return a.containerzDo(func(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
		return a.containerzListVolume(ctx, t, c)
	})
}

func (a *App) containerzListVolume(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
	req := &containerz.ListVolumeRequest{
		Filter: parseVolumeFilter(a.Config.ContainerzVolumeListFilter),
	}
	stream, err := c.ListVolume(ctx, req)
	if err != nil {
		return fmt.Errorf("ListVolume RPC failed: %v", err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"TARGET", "NAME", "DRIVER", "CREATED"})
	formatTable(table)

	count := 0
	for {
		rsp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ListVolume stream error: %v", err)
		}
		a.printMsg(t.Config.Address, rsp)
		created := ""
		if rsp.Created != nil {
			created = rsp.Created.AsTime().Format("2006-01-02 15:04:05")
		}
		table.Append([]string{t.Config.Address, rsp.Name, rsp.Driver, created})
		count++
	}
	if count > 0 {
		table.Render()
	} else {
		fmt.Printf("[%s] No volumes found\n", t.Config.Address)
	}
	return nil
}

// RemoveVolume

func (a *App) InitContainerzRemoveVolumeFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.Flags().StringVar(&a.Config.ContainerzVolumeRemoveName, "name", "", "volume name to remove (required)")
	cmd.Flags().BoolVar(&a.Config.ContainerzVolumeRemoveForce, "force", false, "force removal")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

func (a *App) RunEContainerzRemoveVolume(cmd *cobra.Command, args []string) error {
	if a.Config.ContainerzVolumeRemoveName == "" {
		return fmt.Errorf("--name is required")
	}
	return a.containerzDo(func(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
		return a.containerzRemoveVolume(ctx, t, c)
	})
}

func (a *App) containerzRemoveVolume(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
	req := &containerz.RemoveVolumeRequest{
		Name:  a.Config.ContainerzVolumeRemoveName,
		Force: a.Config.ContainerzVolumeRemoveForce,
	}
	rsp, err := c.RemoveVolume(ctx, req)
	if err != nil {
		return fmt.Errorf("RemoveVolume RPC failed: %v", err)
	}
	a.printMsg(t.Config.Address, rsp)
	fmt.Printf("[%s] Volume %q removed\n", t.Config.Address, a.Config.ContainerzVolumeRemoveName)
	return nil
}
