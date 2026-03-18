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

// startContainerParams bundles all inputs for buildStartContainerRequest.
// Using a struct avoids an ever-growing positional argument list and makes
// call sites self-documenting.
type startContainerParams struct {
	// Core identity
	ImageName    string
	Tag          string
	Cmd          string
	InstanceName string

	// Networking
	Network string
	Ports   []string // "internal:external"

	// Runtime environment
	Env     []string // "KEY=VALUE"
	Volumes []string // "name:mountpoint[:ro]"
	Labels  []string // "KEY=VALUE"

	// Capabilities: add/remove Linux capabilities
	CapAdd    []string
	CapRemove []string

	// RunAs: "user[:group]" under which the container process runs.
	// Group is optional; parsed from a single "user[:group]" string.
	RunAs string

	// Resource limits
	LimitCPU     float64
	LimitSoftMem int64
	LimitHardMem int64

	// Devices: "src_path:dst_path[:perms]" where perms is any combo of r/w/m
	Devices []string

	// Restart policy
	RestartPolicy string

	// Placement (only for StartContainer, not UpdateContainer params)
	Location string
}

// parseRestartPolicy converts a restart policy string (e.g. "on-failure:5")
// into a StartContainerRequest_Restart message.
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

// parseDevicePerms converts a permission string (any combination of 'r', 'w',
// 'm') into a slice of Device_Permission enum values.
func parseDevicePerms(perms string) []containerz.Device_Permission {
	var out []containerz.Device_Permission
	seen := map[rune]bool{}
	for _, ch := range strings.ToLower(perms) {
		if seen[ch] {
			continue
		}
		seen[ch] = true
		switch ch {
		case 'r':
			out = append(out, containerz.Device_READ)
		case 'w':
			out = append(out, containerz.Device_WRITE)
		case 'm':
			out = append(out, containerz.Device_MKNOD)
		}
	}
	return out
}

// buildStartContainerRequest assembles a StartContainerRequest from a
// startContainerParams struct. It is shared by StartContainer and the
// params sub-message of UpdateContainer.
//
// Tag defaulting: "latest" is applied automatically when ImageName is
// non-empty and Tag is empty. When ImageName is empty (restart-by-name case)
// the tag is left blank.
func buildStartContainerRequest(p startContainerParams) (*containerz.StartContainerRequest, error) {
	// Apply tag default only when an image was specified.
	if p.ImageName != "" && p.Tag == "" {
		p.Tag = "latest"
	}

	req := &containerz.StartContainerRequest{
		ImageName:    p.ImageName,
		Tag:          p.Tag,
		Cmd:          p.Cmd,
		InstanceName: p.InstanceName,
		Environment:  parseKVMap(p.Env),
		Labels:       parseKVMap(p.Labels),
		Network:      p.Network,
	}

	// Location: where on the device the container should run.
	switch strings.ToLower(p.Location) {
	case "primary":
		req.Location = containerz.StartContainerRequest_L_PRIMARY
	case "backup":
		req.Location = containerz.StartContainerRequest_L_BACKUP
	case "all":
		req.Location = containerz.StartContainerRequest_L_ALL
	// "" or "unknown" → leave at zero value (L_UNKNOWN)
	}

	// Port mappings "internal:external"
	for _, port := range p.Ports {
		parts := strings.SplitN(port, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid port mapping %q, expected internal:external", port)
		}
		var internal, external uint32
		if _, err := fmt.Sscanf(parts[0], "%d", &internal); err != nil {
			return nil, fmt.Errorf("invalid internal port in %q: %v", port, err)
		}
		if _, err := fmt.Sscanf(parts[1], "%d", &external); err != nil {
			return nil, fmt.Errorf("invalid external port in %q: %v", port, err)
		}
		req.Ports = append(req.Ports, &containerz.StartContainerRequest_Port{
			Internal: internal,
			External: external,
		})
	}

	// Volume mounts "name:mountpoint[:ro]"
	for _, vol := range p.Volumes {
		parts := strings.SplitN(vol, ":", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid volume %q, expected name:mountpoint[:ro]", vol)
		}
		v := &containerz.Volume{
			Name:       parts[0],
			MountPoint: parts[1],
		}
		if len(parts) == 3 && strings.EqualFold(parts[2], "ro") {
			v.ReadOnly = true
		}
		req.Volumes = append(req.Volumes, v)
	}

	// Restart policy
	restart, err := parseRestartPolicy(p.RestartPolicy)
	if err != nil {
		return nil, err
	}
	req.Restart = restart

	// Capabilities
	if len(p.CapAdd) > 0 || len(p.CapRemove) > 0 {
		req.Cap = &containerz.StartContainerRequest_Capabilities{
			Add:    p.CapAdd,
			Remove: p.CapRemove,
		}
	}

	// RunAs "user[:group]"
	if p.RunAs != "" {
		user, group, _ := strings.Cut(p.RunAs, ":")
		req.RunAs = &containerz.StartContainerRequest_RunAs{
			User:  user,
			Group: group,
		}
	}

	// Resource limits
	if p.LimitCPU != 0 || p.LimitSoftMem != 0 || p.LimitHardMem != 0 {
		req.Limits = &containerz.StartContainerRequest_Limits{
			MaxCpu:       p.LimitCPU,
			SoftMemBytes: p.LimitSoftMem,
			HardMemBytes: p.LimitHardMem,
		}
	}

	// Devices "src:dst[:perms]"
	// perms is any combination of: r (read), w (write), m (mknod)
	// e.g. "/dev/sda:/dev/sda:rw" or "/dev/gpio:/dev/gpio:rm"
	for _, d := range p.Devices {
		parts := strings.SplitN(d, ":", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid device %q, expected src:dst[:perms]", d)
		}
		dev := &containerz.Device{
			SrcPath: parts[0],
			DstPath: parts[1],
		}
		if len(parts) == 3 {
			dev.Permissions = parseDevicePerms(parts[2])
		}
		req.Devices = append(req.Devices, dev)
	}

	return req, nil
}

// StartContainer

func (a *App) InitContainerzStartContainerFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	// --instance-name and --image are both optional individually:
	//   * --instance-name alone  -> restart a stopped container
	//   * --image alone          -> start a new container (server assigns name)
	//   * both                   -> start a new container with the given name
	// At least one must be provided.
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartInstanceName, "instance-name", "", "name of the container instance; omit to let the server auto-assign one")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartImageName, "image", "", "container image name; omit to restart a stopped container by --instance-name")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartTag, "tag", "", "image tag (default: latest when --image is provided)")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartCmd, "cmd", "", "command to run inside the container")
	// Networking
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerStartPorts, "port", []string{}, "port mappings in internal:external format (repeatable)")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartNetwork, "network", "", "network mode (e.g. host, bridge)")
	// Environment & labels
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerStartEnv, "env", []string{}, "environment variables in KEY=VALUE format (repeatable)")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerStartLabels, "label", []string{}, "labels in KEY=VALUE format (repeatable)")
	// Volumes
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerStartVolumes, "volume", []string{}, "volume mounts in name:mountpoint[:ro] format (repeatable)")
	// Capabilities
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerStartCapAdd, "cap-add", []string{}, "Linux capabilities to add (repeatable, e.g. NET_ADMIN)")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerStartCapRemove, "cap-remove", []string{}, "Linux capabilities to remove (repeatable, e.g. MKNOD)")
	// RunAs
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartRunAs, "run-as", "", "user[:group] to run the container process as (e.g. appuser or appuser:appgroup)")
	// Resource limits
	cmd.Flags().Float64Var(&a.Config.ContainerzContainerStartLimitCPU, "limit-cpu", 0, "max CPU fraction the container may use (e.g. 0.5 = half a core)")
	cmd.Flags().Int64Var(&a.Config.ContainerzContainerStartLimitSoftMem, "limit-soft-mem", 0, "soft memory limit in bytes (eviction hint; 0 = unlimited)")
	cmd.Flags().Int64Var(&a.Config.ContainerzContainerStartLimitHardMem, "limit-hard-mem", 0, "hard memory limit in bytes (OOM kill threshold; 0 = unlimited)")
	// Devices
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerStartDevices, "device", []string{}, "host devices in src:dst[:perms] format, perms = any of r/w/m (repeatable)")
	// Restart & placement
	cmd.Flags().StringVar(&a.Config.ContainerzContainerStartRestart, "restart", "none", "restart policy: none|always|unless-stopped|on-failure[:<attempts>]")
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
	req, err := buildStartContainerRequest(startContainerParams{
		ImageName:     a.Config.ContainerzContainerStartImageName,
		Tag:           a.Config.ContainerzContainerStartTag,
		Cmd:           a.Config.ContainerzContainerStartCmd,
		InstanceName:  a.Config.ContainerzContainerStartInstanceName,
		Network:       a.Config.ContainerzContainerStartNetwork,
		Ports:         a.Config.ContainerzContainerStartPorts,
		Env:           a.Config.ContainerzContainerStartEnv,
		Volumes:       a.Config.ContainerzContainerStartVolumes,
		Labels:        a.Config.ContainerzContainerStartLabels,
		CapAdd:       a.Config.ContainerzContainerStartCapAdd,
		CapRemove:    a.Config.ContainerzContainerStartCapRemove,
		RunAs:        a.Config.ContainerzContainerStartRunAs,
		LimitCPU:     a.Config.ContainerzContainerStartLimitCPU,
		LimitSoftMem:  a.Config.ContainerzContainerStartLimitSoftMem,
		LimitHardMem:  a.Config.ContainerzContainerStartLimitHardMem,
		Devices:       a.Config.ContainerzContainerStartDevices,
		RestartPolicy: a.Config.ContainerzContainerStartRestart,
		Location:      a.Config.ContainerzContainerStartLocation,
	})
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
	// These use the same flag names as container-start for convenience.
	// If none are provided, params is left nil and the server reuses the
	// existing container configuration.
	cmd.Flags().StringVar(&a.Config.ContainerzContainerUpdateParamsCmd, "cmd", "", "command to run inside the updated container")
	// Networking
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerUpdateParamsPorts, "port", []string{}, "port mappings in internal:external format (repeatable)")
	cmd.Flags().StringVar(&a.Config.ContainerzContainerUpdateParamsNetwork, "network", "", "network mode for the updated container (e.g. host, bridge)")
	// Environment & labels
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerUpdateParamsEnv, "env", []string{}, "environment variables in KEY=VALUE format (repeatable)")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerUpdateParamsLabels, "label", []string{}, "labels in KEY=VALUE format (repeatable)")
	// Volumes
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerUpdateParamsVolumes, "volume", []string{}, "volume mounts in name:mountpoint[:ro] format (repeatable)")
	// Capabilities
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerUpdateParamsCapAdd, "cap-add", []string{}, "Linux capabilities to add (repeatable)")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerUpdateParamsCapRemove, "cap-remove", []string{}, "Linux capabilities to remove (repeatable)")
	// RunAs
	cmd.Flags().StringVar(&a.Config.ContainerzContainerUpdateParamsRunAs, "run-as", "", "user[:group] to run the container process as (e.g. appuser or appuser:appgroup)")
	// Resource limits
	cmd.Flags().Float64Var(&a.Config.ContainerzContainerUpdateParamsLimitCPU, "limit-cpu", 0, "max CPU fraction (e.g. 0.5 = half a core)")
	cmd.Flags().Int64Var(&a.Config.ContainerzContainerUpdateParamsLimitSoftMem, "limit-soft-mem", 0, "soft memory limit in bytes (0 = unlimited)")
	cmd.Flags().Int64Var(&a.Config.ContainerzContainerUpdateParamsLimitHardMem, "limit-hard-mem", 0, "hard memory limit in bytes (0 = unlimited)")
	// Devices
	cmd.Flags().StringSliceVar(&a.Config.ContainerzContainerUpdateParamsDevices, "device", []string{}, "host devices in src:dst[:perms] format (repeatable)")
	// Restart
	cmd.Flags().StringVar(&a.Config.ContainerzContainerUpdateParamsRestart, "restart", "", "restart policy: none|always|unless-stopped|on-failure[:<attempts>]")

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
		params, err := buildStartContainerRequest(startContainerParams{
			ImageName:     a.Config.ContainerzContainerUpdateImageName,
			Tag:           a.Config.ContainerzContainerUpdateImageTag,
			Cmd:           a.Config.ContainerzContainerUpdateParamsCmd,
			InstanceName:  a.Config.ContainerzContainerUpdateInstanceName, // Use update instance name
			Network:       a.Config.ContainerzContainerUpdateParamsNetwork,
			Ports:         a.Config.ContainerzContainerUpdateParamsPorts,
			Env:           a.Config.ContainerzContainerUpdateParamsEnv,
			Volumes:       a.Config.ContainerzContainerUpdateParamsVolumes,
			Labels:        a.Config.ContainerzContainerUpdateParamsLabels,
			CapAdd:        a.Config.ContainerzContainerUpdateParamsCapAdd,
			CapRemove:     a.Config.ContainerzContainerUpdateParamsCapRemove,
			RunAs:        a.Config.ContainerzContainerUpdateParamsRunAs,
			LimitCPU:      a.Config.ContainerzContainerUpdateParamsLimitCPU,
			LimitSoftMem:  a.Config.ContainerzContainerUpdateParamsLimitSoftMem,
			LimitHardMem:  a.Config.ContainerzContainerUpdateParamsLimitHardMem,
			Devices:       a.Config.ContainerzContainerUpdateParamsDevices,
			RestartPolicy: a.Config.ContainerzContainerUpdateParamsRestart,
			// Location is not updatable per the proto spec; omitted here.
		})
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

// paramsProvided returns true if the caller supplied at least one runtime
// params flag, meaning req.Params should be populated.
func (a *App) paramsProvided() bool {
	return a.Config.ContainerzContainerUpdateParamsCmd != "" ||
		a.Config.ContainerzContainerUpdateInstanceName != "" ||
		a.Config.ContainerzContainerUpdateParamsNetwork != "" ||
		a.Config.ContainerzContainerUpdateParamsRestart != "" ||
		a.Config.ContainerzContainerUpdateParamsRunAs != "" ||
		a.Config.ContainerzContainerUpdateParamsLimitCPU != 0 ||
		a.Config.ContainerzContainerUpdateParamsLimitSoftMem != 0 ||
		a.Config.ContainerzContainerUpdateParamsLimitHardMem != 0 ||
		len(a.Config.ContainerzContainerUpdateParamsPorts) > 0 ||
		len(a.Config.ContainerzContainerUpdateParamsEnv) > 0 ||
		len(a.Config.ContainerzContainerUpdateParamsVolumes) > 0 ||
		len(a.Config.ContainerzContainerUpdateParamsLabels) > 0 ||
		len(a.Config.ContainerzContainerUpdateParamsCapAdd) > 0 ||
		len(a.Config.ContainerzContainerUpdateParamsCapRemove) > 0 ||
		len(a.Config.ContainerzContainerUpdateParamsDevices) > 0
}
