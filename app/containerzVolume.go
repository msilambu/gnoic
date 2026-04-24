package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/karimra/gnoic/api"
	"github.com/olekukonko/tablewriter"
	"github.com/openconfig/gnoi/containerz"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// parseVolumeOptions parses a string of the form
// "type=<val>,options=<opt>,mountpoint=<path>" into a LocalDriverOptions message.
// All keys are optional. The "options" key may appear multiple times separated
// by comma within its value is not supported; use multiple comma-separated
// key=value pairs instead (e.g. "options=opt1,options=opt2").
func parseVolumeOptions(input string) (*containerz.LocalDriverOptions, error) {
	out := &containerz.LocalDriverOptions{}
	for _, part := range strings.Split(input, ",") {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("invalid options token %q, expected key=value", part)
		}
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "type":
			switch strings.ToLower(strings.TrimSpace(v)) {
			case "none":
				out.Type = containerz.LocalDriverOptions_TYPE_NONE
			default:
				out.Type = containerz.LocalDriverOptions_TYPE_UNSPECIFIED
			}
		case "options":
			out.Options = append(out.Options, strings.TrimSpace(v))
		case "mountpoint":
			out.Mountpoint = strings.TrimSpace(v)
		default:
			return nil, fmt.Errorf("unknown options key %q", k)
		}
	}
	return out, nil
}

// CreateVolume

func (a *App) InitContainerzCreateVolumeFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.Flags().StringVar(&a.Config.ContainerzVolumeCreateName, "name", "", "volume name; omit to let the server auto-assign one")
	cmd.Flags().StringVar(&a.Config.ContainerzVolumeCreateDriver, "driver", "local", "volume driver: local|custom")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzVolumeCreateLabels, "label", []string{}, "labels in KEY=VALUE format (repeatable)")
	cmd.Flags().StringVar(&a.Config.ContainerzVolumeCreateOptions, "options", "", `local mount options in type=<val>,options=<opt>,mountpoint=<path> format`)
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

func (a *App) RunEContainerzCreateVolume(cmd *cobra.Command, args []string) error {
	return a.containerzDo(func(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
		return a.containerzCreateVolume(ctx, t, c)
	})
}

// CreateVolume
// If volume dirver is either empty or 'local', set Driver to DS_LOCAL type.
// Everything else is assumed to be DS_CUSTOM type.
func (a *App) containerzCreateVolume(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
	req := &containerz.CreateVolumeRequest{
		Name:   a.Config.ContainerzVolumeCreateName,
		Labels: parseKVMap(a.Config.ContainerzVolumeCreateLabels),
	}

	if a.Config.ContainerzVolumeCreateOptions != "" {
		opts, err := parseVolumeOptions(a.Config.ContainerzVolumeCreateOptions)
		if err != nil {
			return err
		}
		req.Options = &containerz.CreateVolumeRequest_LocalMountOptions{LocalMountOptions: opts}
	}

	switch a.Config.ContainerzVolumeCreateDriver {
	case "local", "":
		req.Driver = containerz.Driver_DS_LOCAL
	default:
		req.Driver = containerz.Driver_DS_CUSTOM
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
