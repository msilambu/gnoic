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

// shared helpers

// parseRestartPolicy converts a restart policy string (e.g. "on-failure:5") into
// a StartContainerRequest_Restart message. It is used by both StartContainer and
// the params field of UpdateContainer.
func parseRestartPolicy(input string) (*containerz.StartContainerRequest_Restart, error) {
	v := strings.ToLower(input)
	switch {
	case v == "always":
		return &containerz.StartContainerRequest_Restart{
			Policy: containerz.StartContainerRequest_Restart_ALWAYS,
		}, nil
	case v == "unless-stopped":
		return &containerz.StartContainerRequest_Restart{
			Policy: containerz.StartContainerRequest_Restart_UNLESS_STOPPED,
		}, nil
	case v == "on-failure" || strings.HasPrefix(v, "on-failure:"):
		var attempts uint32
		if idx := strings.Index(v, ":"); idx != -1 {
			suffix := v[idx+1:]
			var n uint32
			if _, err := fmt.Sscanf(suffix, "%d", &n); err != nil {
				return nil, fmt.Errorf("invalid attempts value in restart policy %q: %v", input, err)
			}
			attempts = n
		}
		return &containerz.StartContainerRequest_Restart{
			Policy:   containerz.StartContainerRequest_Restart_ON_FAILURE,
			Attempts: attempts,
		}, nil
	default:
		return &containerz.StartContainerRequest_Restart{
			Policy: containerz.StartContainerRequest_Restart_NONE,
		}, nil
	}
}

// buildStartContainerRequest assembles a StartContainerRequest from the
// supplied field values. It is used directly by StartContainer and as the
// params sub-message of UpdateContainer.
//
// Tag defaulting: if imageName is non-empty and tag is empty, tag is set to
// "latest" automatically. When imageName is empty (restart-by-name case), the
// tag is left empty and not sent to the server.
//
// Location is one of: "" (omit / L_UNKNOWN), "primary", "backup", "all".
func buildStartContainerRequest(
	imageName, tag, cmd, instanceName, network, restartPolicy, location string,
	ports, env, volumes, labels []string,
) (*containerz.StartContainerRequest, error) {
	// Apply the "latest" default only when an image was actually specified.
	if imageName != "" && tag == "" {
		tag = "latest"
	}

	req := &containerz.StartContainerRequest{
		ImageName:    imageName,
		Tag:          tag,
		Cmd:          cmd,
		InstanceName: instanceName,
		Environment:  parseKVMap(env),
		Labels:       parseKVMap(labels),
		Network:      network,
	}

	// Location: where on the device the container should run.
	switch strings.ToLower(location) {
	case "primary":
		req.Location = containerz.StartContainerRequest_L_PRIMARY
	case "backup":
		req.Location = containerz.StartContainerRequest_L_BACKUP
	case "all":
		req.Location = containerz.StartContainerRequest_L_ALL
	// "" or "unknown" → leave at zero value (L_UNKNOWN)
	}

	// Port mappings "internal:external"
	for _, p := range ports {
		parts := strings.SplitN(p, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid port mapping %q, expected internal:external", p)
		}
		var internal, external uint32
		if _, err := fmt.Sscanf(parts[0], "%d", &internal); err != nil {
			return nil, fmt.Errorf("invalid internal port in %q: %v", p, err)
		}
		if _, err := fmt.Sscanf(parts[1], "%d", &external); err != nil {
			return nil, fmt.Errorf("invalid external port in %q: %v", p, err)
		}
		req.Ports = append(req.Ports, &containerz.StartContainerRequest_Port{
			Internal: internal,
			External: external,
		})
	}

	// Volume mounts "name:mountpoint[:ro]"
	for _, v := range volumes {
		parts := strings.SplitN(v, ":", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid volume %q, expected name:mountpoint[:ro]", v)
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

	// Restart policy
	restart, err := parseRestartPolicy(restartPolicy)
	if err != nil {
		return nil, err
	}
	req.Restart = restart

	return req, nil
}

// StartContainer

func (a *App) InitContainerzStartContainerFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	// --instance-name and --image are both optional individually:
	//   * --instance-name alone  -> restart a stopped container
	//   * --image alone          -> start a new container (server assigns instance name)
	//   * both                   -> start a new container with the given name
	// At least one of the two must be provided.
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartInstanceName, "instance-name", "", "name of the container instance; omit to let the server auto-assign one")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartImageName, "image", "", "container image name; omit to restart a stopped container identified by --instance-name")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartTag, "tag", "", "image tag (default: latest when --image is provided)")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartCmd, "cmd", "", "command to run inside the container")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerStartPorts, "port", []string{}, "port mappings in internal:external format (repeatable)")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerStartEnv, "env", []string{}, "environment variables in KEY=VALUE format (repeatable)")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerStartVolumes, "volume", []string{}, "volume mounts in name:mountpoint[:ro] format (repeatable)")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartNetwork, "network", "", "network mode (e.g. host, bridge)")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartRestart, "restart", "none", "restart policy: none|always|unless-stopped|on-failure[:<attempts>]")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerStartLabels, "label", []string{}, "labels in KEY=VALUE format (repeatable)")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartLocation, "location", "", "where to run the container: primary|backup|all (default: unspecified)")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

func (a *App) RunEContainerzStartContainer(cmd *cobra.Command, args []string) error {
	if a.Config.ContainerzContainerStartImageName == "" && a.Config.ContainerzContainerStartInstanceName == "" {
		return fmt.Errorf("at least one of --image or --instance-name is required")
	}
	return a.containerzDo(func(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
		return a.containerzStartContainer(ctx, t, c)
	})
}

func (a *App) containerzStartContainer(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
	req, err := buildStartContainerRequest(
		a.Config.ContainerzContainerStartImageName,
		a.Config.ContainerzContainerStartTag,
		a.Config.ContainerzContainerStartCmd,
		a.Config.ContainerzContainerStartInstanceName,
		a.Config.ContainerzContainerStartNetwork,
		a.Config.ContainerzContainerStartRestart,
		a.Config.ContainerzContainerStartLocation,
		a.Config.ContainerzContainerStartPorts,
		a.Config.ContainerzContainerStartEnv,
		a.Config.ContainerzContainerStartVolumes,
		a.Config.ContainerzContainerStartLabels,
	)
	if err != nil {
		return err
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

// StopContainer

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

// ListContainer

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

// RemoveContainer

func (a *App) InitContainerzRemoveContainerFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.Flags().StringVar(&a.Config.ContainerzContainerRemoveName, "instance-name", "", "name of the container instance to remove (required)")
	cmd.Flags().BoolVar(&a.Config.ContainerzContainerRemoveForce, "force", false, "force removal of a running container")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

func (a *App) RunEContainerzRemoveContainer(cmd *cobra.Command, args []string) error {
	if a.Config.ContainerzContainerRemoveName == "" {
		return fmt.Errorf("--instance-name is required")
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

// UpdateContainer

func (a *App) InitContainerzUpdateContainerFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	// Identity of the container being updated and the new image target.
	cmd.Flags().StringVar(&a.Config.ContainerzContainerUpdateInstanceName, "instance-name", "", "name of the running container to update (required)")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerUpdateImageName, "image", "", "new image name to update to (required)")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerUpdateImageTag, "tag", "", "new image tag (default: latest when --image is provided)")
	cmd.Flags().BoolVar(&a.Config.ContainerzContainerUpdateAsync, "async", false, "perform the update asynchronously")
	// params field – runtime parameters for the updated container.
	// These use the same flag names as 'container start' for convenience.
	// If none are provided, params is left nil and the server reuses the
	// existing container configuration.
	cmd.Flags().StringVar(&a.Config.ContainerzContainerUpdateParamsCmd, "cmd", "", "command to run inside the updated container")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerUpdateParamsPorts, "port", []string{}, "port mappings in internal:external format (repeatable)")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerUpdateParamsEnv, "env", []string{}, "environment variables in KEY=VALUE format (repeatable)")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerUpdateParamsVolumes, "volume", []string{}, "volume mounts in name:mountpoint[:ro] format (repeatable)")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerUpdateParamsNetwork, "network", "", "network mode for the updated container (e.g. host, bridge)")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerUpdateParamsRestart, "restart", "", "restart policy: none|always|unless-stopped|on-failure[:<attempts>]")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerUpdateParamsLabels, "label", []string{}, "labels in KEY=VALUE format (repeatable)")
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
	// Apply the same "default to latest" tag rule as container-start.
	imageTag := a.Config.ContainerzContainerUpdateImageTag
	if imageTag == "" {
		imageTag = "latest"
	}
	req := &containerz.UpdateContainerRequest{
		InstanceName: a.Config.ContainerzContainerUpdateInstanceName,
		ImageName:    a.Config.ContainerzContainerUpdateImageName,
		ImageTag:     imageTag,
		Async:        a.Config.ContainerzContainerUpdateAsync,
	}

	// Populate params only when at least one params flag was supplied.
	// An empty params field tells the server to keep the existing runtime config.
	if a.paramsProvided() {
		params, err := buildStartContainerRequest(
			a.Config.ContainerzContainerUpdateImageName,
			a.Config.ContainerzContainerUpdateImageTag,
			a.Config.ContainerzContainerUpdateParamsCmd,
			a.Config.ContainerzContainerUpdateInstanceName,  // Use same instance name 
			a.Config.ContainerzContainerUpdateParamsNetwork,
			a.Config.ContainerzContainerUpdateParamsRestart,
			"", // location is not applicable for container-update params
			a.Config.ContainerzContainerUpdateParamsPorts,
			a.Config.ContainerzContainerUpdateParamsEnv,
			a.Config.ContainerzContainerUpdateParamsVolumes,
			a.Config.ContainerzContainerUpdateParamsLabels,
		)
		if err != nil {
			return fmt.Errorf("invalid params: %v", err)
		}
		req.Params = params
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

// paramsProvided returns true if the caller set at least one params flag
// (--cmd, --port, --env, --volume, --network, --restart, --label,
// --new-instance-name), meaning req.Params should be populated.
func (a *App) paramsProvided() bool {
	return a.Config.ContainerzContainerUpdateParamsCmd != "" ||
		a.Config.ContainerzContainerUpdateInstanceName != "" ||
		a.Config.ContainerzContainerUpdateParamsNetwork != "" ||
		a.Config.ContainerzContainerUpdateParamsRestart != "" ||
		len(a.Config.ContainerzContainerUpdateParamsPorts) > 0 ||
		len(a.Config.ContainerzContainerUpdateParamsEnv) > 0 ||
		len(a.Config.ContainerzContainerUpdateParamsVolumes) > 0 ||
		len(a.Config.ContainerzContainerUpdateParamsLabels) > 0
}
