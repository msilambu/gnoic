package cmd

import "github.com/spf13/cobra"

// newContainerzCmd represents the top-level containerz command.
// All sub-commands are registered directly here (2-level hierarchy):
//
//	gnoic containerz <group-action> [flags]
func newContainerzCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "containerz",
		Short:        "run Containerz gNOI RPCs",
		SilenceUsage: true,
	}
	gApp.InitContainerzFlags(cmd)
	cmd.AddCommand(
		// Image RPCs
		newContainerzImagePushCmd(),
		newContainerzImageListCmd(),
		newContainerzImageRemoveCmd(),

		// Container RPCs
		newContainerzContainerStartCmd(),
		newContainerzContainerStopCmd(),
		newContainerzContainerListCmd(),
		newContainerzContainerRemoveCmd(),
		newContainerzContainerUpdateCmd(),

		// Log RPC
		newContainerzLogCmd(),

		// Volume RPCs
		newContainerzVolumeCreateCmd(),
		newContainerzVolumeListCmd(),
		newContainerzVolumeRemoveCmd(),

		// Plugin RPCs
		newContainerzPluginStartCmd(),
		newContainerzPluginStopCmd(),
		newContainerzPluginListCmd(),
		newContainerzPluginRemoveCmd(),
	)
	return cmd
}

// Deploy

func newContainerzImagePushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image-push",
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
		Use:   "image-list",
		Short: "run containerz ListImage gNOI RPC",
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
		Use:   "image-remove",
		Short: "run containerz RemoveImage gNOI RPC",
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
		Use:   "container-start",
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
		Use:   "container-stop",
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
		Use:   "container-list",
		Short: "run containerz ListContainer gNOI RPC",
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
		Use:   "container-remove",
		Short: "run containerz RemoveContainer gNOI RPC",
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
		Use:   "container-update",
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
		Use:   "volume-create",
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
		Use:   "volume-list",
		Short: "run containerz ListVolume gNOI RPC",
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
		Use:   "volume-remove",
		Short: "run containerz RemoveVolume gNOI RPC",
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
		Use:   "plugin-start",
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
		Use:   "plugin-stop",
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
		Use:   "plugin-list",
		Short: "run containerz ListPlugins gNOI RPC",
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
		Use:   "plugin-remove",
		Short: "run containerz RemovePlugin gNOI RPC",
		PreRun: func(cmd *cobra.Command, _ []string) {
			gApp.Config.SetLocalFlagsFromFile(cmd)
		},
		RunE:         gApp.RunEContainerzRemovePlugin,
		SilenceUsage: true,
	}
	gApp.InitContainerzRemovePluginFlags(cmd)
	return cmd
}
