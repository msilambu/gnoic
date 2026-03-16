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

//  StartContainer

func (a *App) InitContainerzStartContainerFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartImageName, "image", "", "container image name (required)")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartTag, "tag", "latest", "image tag")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartCmd, "cmd", "", "command to run inside the container")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartInstanceName, "instance-name", "", "name for the running container (auto-assigned if empty)")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerStartPorts, "port", []string{}, "port mappings in internal:external format (repeatable)")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerStartEnv, "env", []string{}, "environment variables in KEY=VALUE format (repeatable)")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerStartVolumes, "volume", []string{}, "volume mounts in name:mountpoint[:ro] format (repeatable)")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartNetwork, "network", "", "network mode (e.g. host, bridge)")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartRestart, "restart", "none", "restart policy: none|always|unless-stopped|on-failure")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerStartLabels, "label", []string{}, "labels in KEY=VALUE format (repeatable)")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

func (a *App) RunEContainerzStartContainer(cmd *cobra.Command, args []string) error {
	if a.Config.ContainerzContainerStartImageName == "" {
		return fmt.Errorf("--image is required")
	}
	return a.containerzDo(func(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
		return a.containerzStartContainer(ctx, t, c)
	})
}

func (a *App) containerzStartContainer(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
	req := &containerz.StartContainerRequest{
		ImageName:    a.Config.ContainerzContainerStartImageName,
		Tag:          a.Config.ContainerzContainerStartTag,
		Cmd:          a.Config.ContainerzContainerStartCmd,
		InstanceName: a.Config.ContainerzContainerStartInstanceName,
		Environment:  parseKVMap(a.Config.ContainerzContainerStartEnv),
		Labels:       parseKVMap(a.Config.ContainerzContainerStartLabels),
		Network:      a.Config.ContainerzContainerStartNetwork,
	}

	// Parse port mappings "internal:external"
	for _, p := range a.Config.ContainerzContainerStartPorts {
		parts := strings.SplitN(p, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid port mapping %q, expected internal:external", p)
		}
		var internal, external uint32
		if _, err := fmt.Sscanf(parts[0], "%d", &internal); err != nil {
			return fmt.Errorf("invalid internal port in %q: %v", p, err)
		}
		if _, err := fmt.Sscanf(parts[1], "%d", &external); err != nil {
			return fmt.Errorf("invalid external port in %q: %v", p, err)
		}
		req.Ports = append(req.Ports, &containerz.StartContainerRequest_Port{
			Internal: internal,
			External: external,
		})
	}

	// Parse volume mounts "name:mountpoint[:ro]"
	for _, v := range a.Config.ContainerzContainerStartVolumes {
		parts := strings.SplitN(v, ":", 3)
		if len(parts) < 2 {
			return fmt.Errorf("invalid volume %q, expected name:mountpoint[:ro]", v)
		}
		vol := &containerz.Volume{
			Name:       parts[0],
			MountPoint: parts[1],
		}
		if len(parts) == 3 && strings.EqualFold(parts[2], "ro") {
			vol.ReadOnly = true
		}
		req.Volumes = append(req.Volumes, vol)
	}

	// Parse restart policy.
	// on-failure accepts an optional attempts count: "on-failure:<n>".
	restartInput := strings.ToLower(a.Config.ContainerzContainerStartRestart)
	switch {
	case restartInput == "always":
		req.Restart = &containerz.StartContainerRequest_Restart{
			Policy: containerz.StartContainerRequest_Restart_ALWAYS,
		}
	case restartInput == "unless-stopped":
		req.Restart = &containerz.StartContainerRequest_Restart{
			Policy: containerz.StartContainerRequest_Restart_UNLESS_STOPPED,
		}
	case restartInput == "on-failure" || strings.HasPrefix(restartInput, "on-failure:"):
		var attempts uint32
		if idx := strings.Index(restartInput, ":"); idx != -1 {
			suffix := restartInput[idx+1:]
			var n uint32
			if _, err := fmt.Sscanf(suffix, "%d", &n); err != nil {
				return fmt.Errorf("invalid attempts value in restart policy %q: %v",
					a.Config.ContainerzContainerStartRestart, err)
			}
			attempts = n
		}
		req.Restart = &containerz.StartContainerRequest_Restart{
			Policy: containerz.StartContainerRequest_Restart_ON_FAILURE,
			Attempts: attempts,
		}
	default:
		req.Restart = &containerz.StartContainerRequest_Restart{
			Policy: containerz.StartContainerRequest_Restart_NONE,
		}
	}

	rsp, err := c.StartContainer(ctx, req)
	if err != nil {
		return fmt.Errorf("StartContainer RPC failed: %v", err)
	}
	a.printMsg(t.Config.Address, rsp)
	switch v := rsp.GetResponse().(type) {
	case *containerz.StartContainerResponse_StartOk:
		fmt.Printf("[%s] Container started: instance_name=%s\n", t.Config.Address, v.StartOk.InstanceName)
	case *containerz.StartContainerResponse_StartError:
		return fmt.Errorf("[%s] StartContainer failed", t.Config.Address)
	}
	return nil
}

//  StopContainer 

func (a *App) InitContainerzStopContainerFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStopInstanceName, "instance-name", "", "name of the container instance to stop (required)")
	cmd.Flags().BoolVar(&a.Config.ContainerzContainerStopForce, "force", false, "forcefully kill the container")
	cmd.Flags().BoolVar(&a.Config.ContainerzContainerStopRestart, "restart", false, "restart container immediately after stopping")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

func (a *App) RunEContainerzStopContainer(cmd *cobra.Command, args []string) error {
	if a.Config.ContainerzContainerStopInstanceName == "" {
		return fmt.Errorf("--instance-name is required")
	}
	return a.containerzDo(func(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
		return a.containerzStopContainer(ctx, t, c)
	})
}

func (a *App) containerzStopContainer(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
	req := &containerz.StopContainerRequest{
		InstanceName: a.Config.ContainerzContainerStopInstanceName,
		Force:        a.Config.ContainerzContainerStopForce,
		Restart:      a.Config.ContainerzContainerStopRestart,
	}
	rsp, err := c.StopContainer(ctx, req)
	if err != nil {
		return fmt.Errorf("StopContainer RPC failed: %v", err)
	}
	a.printMsg(t.Config.Address, rsp)
	fmt.Printf("[%s] Container %q stopped\n", t.Config.Address, a.Config.ContainerzContainerStopInstanceName)
	return nil
}

//  ListContainer 

func (a *App) InitContainerzListContainerFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.Flags().BoolVar(&a.Config.ContainerzContainerListAll, "all", false, "list all containers including stopped ones")
	cmd.Flags().Int32Var(&a.Config.ContainerzContainerListLimit, "limit", 0, "max number of containers to return (0 = unlimited)")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerListFilter, "filter", []string{}, "filters in key=value format (repeatable)")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

func (a *App) RunEContainerzListContainer(cmd *cobra.Command, args []string) error {
	return a.containerzDo(func(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
		return a.containerzListContainer(ctx, t, c)
	})
}

func (a *App) containerzListContainer(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
	req := &containerz.ListContainerRequest{
		All:    a.Config.ContainerzContainerListAll,
		Limit:  a.Config.ContainerzContainerListLimit,
		Filter: parseContainerFilter(a.Config.ContainerzContainerListFilter),
	}
	stream, err := c.ListContainer(ctx, req)
	if err != nil {
		return fmt.Errorf("ListContainer RPC failed: %v", err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"TARGET", "ID", "NAME", "IMAGE", "STATUS"})
	formatTable(table)

	count := 0
	for {
		rsp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ListContainer stream error: %v", err)
		}
		a.printMsg(t.Config.Address, rsp)
		table.Append([]string{
			t.Config.Address,
			rsp.Id,
			rsp.Name,
			rsp.ImageName,
			rsp.Status.String(),
		})
		count++
	}
	if count > 0 {
		table.Render()
	} else {
		fmt.Printf("[%s] No containers found\n", t.Config.Address)
	}
	return nil
}

//  RemoveContainer

func (a *App) InitContainerzRemoveContainerFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.Flags().StringVar(&a.Config.ContainerzContainerRemoveName, "name", "", "container instance name to remove (required)")
	cmd.Flags().BoolVar(&a.Config.ContainerzContainerRemoveForce, "force", false, "force removal of a running container")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

func (a *App) RunEContainerzRemoveContainer(cmd *cobra.Command, args []string) error {
	if a.Config.ContainerzContainerRemoveName == "" {
		return fmt.Errorf("--name is required")
	}
	return a.containerzDo(func(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
		return a.containerzRemoveContainer(ctx, t, c)
	})
}

func (a *App) containerzRemoveContainer(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
	req := &containerz.RemoveContainerRequest{
		Name:  a.Config.ContainerzContainerRemoveName,
		Force: a.Config.ContainerzContainerRemoveForce,
	}
	rsp, err := c.RemoveContainer(ctx, req)
	if err != nil {
		return fmt.Errorf("RemoveContainer RPC failed: %v", err)
	}
	a.printMsg(t.Config.Address, rsp)
	fmt.Printf("[%s] Container %q removed\n", t.Config.Address, a.Config.ContainerzContainerRemoveName)
	return nil
}

//  UpdateContainer

func (a *App) InitContainerzUpdateContainerFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.Flags().StringVar(&a.Config.ContainerzContainerUpdateInstanceName, "instance-name", "", "name of the running container to update (required)")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerUpdateImageName, "image", "", "new image name to update to (required)")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerUpdateImageTag, "tag", "latest", "new image tag")
	cmd.Flags().BoolVar(&a.Config.ContainerzContainerUpdateAsync, "async", false, "perform the update asynchronously")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

func (a *App) RunEContainerzUpdateContainer(cmd *cobra.Command, args []string) error {
	if a.Config.ContainerzContainerUpdateInstanceName == "" {
		return fmt.Errorf("--instance-name is required")
	}
	if a.Config.ContainerzContainerUpdateImageName == "" {
		return fmt.Errorf("--image is required")
	}
	return a.containerzDo(func(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
		return a.containerzUpdateContainer(ctx, t, c)
	})
}

func (a *App) containerzUpdateContainer(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
	req := &containerz.UpdateContainerRequest{
		InstanceName: a.Config.ContainerzContainerUpdateInstanceName,
		ImageName:    a.Config.ContainerzContainerUpdateImageName,
		ImageTag:     a.Config.ContainerzContainerUpdateImageTag,
		Async:        a.Config.ContainerzContainerUpdateAsync,
	}
	rsp, err := c.UpdateContainer(ctx, req)
	if err != nil {
		return fmt.Errorf("UpdateContainer RPC failed: %v", err)
	}
	a.printMsg(t.Config.Address, rsp)
	switch v := rsp.GetResponse().(type) {
	case *containerz.UpdateContainerResponse_UpdateOk:
		fmt.Printf("[%s] Container updated: instance=%s async=%v\n",
			t.Config.Address, v.UpdateOk.InstanceName, v.UpdateOk.IsAsync)
	case *containerz.UpdateContainerResponse_UpdateError:
		return fmt.Errorf("[%s] UpdateContainer failed", t.Config.Address)
	}
	return nil
}
