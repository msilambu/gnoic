package cmd

import "github.com/spf13/cobra"

// newContainerzCmd represents the top-level containerz command
func newContainerzCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "containerz",
		Short: "run Containerz gNOI RPCs",
		SilenceUsage: true,
	}
	gApp.InitContainerzFlags(cmd)
	cmd.AddCommand(
		newContainerzDeployCmd(),
		newContainerzImageCmd(),
		newContainerzContainerCmd(),
		newContainerzLogCmd(),
		newContainerzVolumeCmd(),
		newContainerzPluginCmd(),
	)
	return cmd
}

// Image sub-group

func newContainerzImageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "image",
		Short:        "run Containerz Image gNOI RPCs",
		SilenceUsage: true,
	}
	cmd.AddCommand(
		newContainerzImageListCmd(),
		newContainerzImageRemoveCmd(),
	)
	return cmd
}

// Container sub-group

func newContainerzContainerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "container",
		Short:        "run Containerz Container gNOI RPCs",
		SilenceUsage: true,
	}
	cmd.AddCommand(
		newContainerzContainerStartCmd(),
		newContainerzContainerStopCmd(),
		newContainerzContainerListCmd(),
		newContainerzContainerRemoveCmd(),
		newContainerzContainerUpdateCmd(),
	)
	return cmd
}

// Volume sub-group

func newContainerzVolumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "volume",
		Short:        "run Containerz Volume gNOI RPCs",
		SilenceUsage: true,
	}
	cmd.AddCommand(
		newContainerzVolumeCreateCmd(),
		newContainerzVolumeListCmd(),
		newContainerzVolumeRemoveCmd(),
	)
	return cmd
}

// Plugin sub-group

func newContainerzPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "plugin",
		Short:        "run Containerz Plugin gNOI RPCs",
		SilenceUsage: true,
	}
	cmd.AddCommand(
		newContainerzPluginStartCmd(),
		newContainerzPluginStopCmd(),
		newContainerzPluginListCmd(),
		newContainerzPluginRemoveCmd(),
	)
	return cmd
}

// Deploy

func newContainerzDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "run containerz Deploy gNOI RPC (upload image or plugin to target)",
		PreRun: func(cmd *cobra.Command, _ []string) {
			gApp.Config.SetLocalFlagsFromFile(cmd)
		},
		RunE:         gApp.RunEContainerzDeploy,
		SilenceUsage: true,
	}
	gApp.InitContainerzDeployFlags(cmd)
	return cmd
}

// Image List

func newContainerzImageListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "run containerz ListImage gNOI RPC",
		PreRun: func(cmd *cobra.Command, _ []string) {
			gApp.Config.SetLocalFlagsFromFile(cmd)
		},
		RunE:         gApp.RunEContainerzListImage,
		SilenceUsage: true,
	}
	gApp.InitContainerzListImageFlags(cmd)
	return cmd
}

// Image Remove

func newContainerzImageRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm"},
		Short:   "run containerz RemoveImage gNOI RPC",
		PreRun: func(cmd *cobra.Command, _ []string) {
			gApp.Config.SetLocalFlagsFromFile(cmd)
		},
		RunE:         gApp.RunEContainerzRemoveImage,
		SilenceUsage: true,
	}
	gApp.InitContainerzRemoveImageFlags(cmd)
	return cmd
}

// Container Start

func newContainerzContainerStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "run containerz StartContainer gNOI RPC",
		PreRun: func(cmd *cobra.Command, _ []string) {
			gApp.Config.SetLocalFlagsFromFile(cmd)
		},
		RunE:         gApp.RunEContainerzStartContainer,
		SilenceUsage: true,
	}
	gApp.InitContainerzStartContainerFlags(cmd)
	return cmd
}

// Container Stop

func newContainerzContainerStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "run containerz StopContainer gNOI RPC",
		PreRun: func(cmd *cobra.Command, _ []string) {
			gApp.Config.SetLocalFlagsFromFile(cmd)
		},
		RunE:         gApp.RunEContainerzStopContainer,
		SilenceUsage: true,
	}
	gApp.InitContainerzStopContainerFlags(cmd)
	return cmd
}

// Container List

func newContainerzContainerListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "run containerz ListContainer gNOI RPC",
		PreRun: func(cmd *cobra.Command, _ []string) {
			gApp.Config.SetLocalFlagsFromFile(cmd)
		},
		RunE:         gApp.RunEContainerzListContainer,
		SilenceUsage: true,
	}
	gApp.InitContainerzListContainerFlags(cmd)
	return cmd
}

// Container Remove

func newContainerzContainerRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm"},
		Short:   "run containerz RemoveContainer gNOI RPC",
		PreRun: func(cmd *cobra.Command, _ []string) {
			gApp.Config.SetLocalFlagsFromFile(cmd)
		},
		RunE:         gApp.RunEContainerzRemoveContainer,
		SilenceUsage: true,
	}
	gApp.InitContainerzRemoveContainerFlags(cmd)
	return cmd
}

// Container Update

func newContainerzContainerUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "run containerz UpdateContainer gNOI RPC",
		PreRun: func(cmd *cobra.Command, _ []string) {
			gApp.Config.SetLocalFlagsFromFile(cmd)
		},
		RunE:         gApp.RunEContainerzUpdateContainer,
		SilenceUsage: true,
	}
	gApp.InitContainerzUpdateContainerFlags(cmd)
	return cmd
}

// Log

func newContainerzLogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log",
		Short: "run containerz Log gNOI RPC (stream container logs)",
		PreRun: func(cmd *cobra.Command, _ []string) {
			gApp.Config.SetLocalFlagsFromFile(cmd)
		},
		RunE:         gApp.RunEContainerzLog,
		SilenceUsage: true,
	}
	gApp.InitContainerzLogFlags(cmd)
	return cmd
}

// Volume Create

func newContainerzVolumeCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "run containerz CreateVolume gNOI RPC",
		PreRun: func(cmd *cobra.Command, _ []string) {
			gApp.Config.SetLocalFlagsFromFile(cmd)
		},
		RunE:         gApp.RunEContainerzCreateVolume,
		SilenceUsage: true,
	}
	gApp.InitContainerzCreateVolumeFlags(cmd)
	return cmd
}

// Volume List

func newContainerzVolumeListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "run containerz ListVolume gNOI RPC",
		PreRun: func(cmd *cobra.Command, _ []string) {
			gApp.Config.SetLocalFlagsFromFile(cmd)
		},
		RunE:         gApp.RunEContainerzListVolume,
		SilenceUsage: true,
	}
	gApp.InitContainerzListVolumeFlags(cmd)
	return cmd
}

// Volume Remove

func newContainerzVolumeRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm"},
		Short:   "run containerz RemoveVolume gNOI RPC",
		PreRun: func(cmd *cobra.Command, _ []string) {
			gApp.Config.SetLocalFlagsFromFile(cmd)
		},
		RunE:         gApp.RunEContainerzRemoveVolume,
		SilenceUsage: true,
	}
	gApp.InitContainerzRemoveVolumeFlags(cmd)
	return cmd
}

// Plugin Start

func newContainerzPluginStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "run containerz StartPlugin gNOI RPC",
		PreRun: func(cmd *cobra.Command, _ []string) {
			gApp.Config.SetLocalFlagsFromFile(cmd)
		},
		RunE:         gApp.RunEContainerzStartPlugin,
		SilenceUsage: true,
	}
	gApp.InitContainerzStartPluginFlags(cmd)
	return cmd
}

// Plugin Stop

func newContainerzPluginStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "run containerz StopPlugin gNOI RPC",
		PreRun: func(cmd *cobra.Command, _ []string) {
			gApp.Config.SetLocalFlagsFromFile(cmd)
		},
		RunE:         gApp.RunEContainerzStopPlugin,
		SilenceUsage: true,
	}
	gApp.InitContainerzStopPluginFlags(cmd)
	return cmd
}

// Plugin List

func newContainerzPluginListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "run containerz ListPlugins gNOI RPC",
		PreRun: func(cmd *cobra.Command, _ []string) {
			gApp.Config.SetLocalFlagsFromFile(cmd)
		},
		RunE:         gApp.RunEContainerzListPlugins,
		SilenceUsage: true,
	}
	gApp.InitContainerzListPluginsFlags(cmd)
	return cmd
}

// Plugin Remove

func newContainerzPluginRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm"},
		Short:   "run containerz RemovePlugin gNOI RPC",
		PreRun: func(cmd *cobra.Command, _ []string) {
			gApp.Config.SetLocalFlagsFromFile(cmd)
		},
		RunE:         gApp.RunEContainerzRemovePlugin,
		SilenceUsage: true,
	}
	gApp.InitContainerzRemovePluginFlags(cmd)
	return cmd
}
