package app

import (
	"context"
	"fmt"
	"io"

	"github.com/karimra/gnoic/api"
	"github.com/openconfig/gnoi/containerz"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// InitContainerzLogFlags registers flags for the 'log' command.
func (a *App) InitContainerzLogFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.Flags().StringVar(&a.Config.ContainerzLogInstanceName, "instance-name", "", "container instance name to stream logs from (required)")
	cmd.Flags().BoolVar(&a.Config.ContainerzLogFollow, "follow", false, "keep the log stream open (like 'docker logs -f')")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

// RunEContainerzLog implements 'gnoic containerz log'.
func (a *App) RunEContainerzLog(cmd *cobra.Command, args []string) error {
	if a.Config.ContainerzLogInstanceName == "" {
		return fmt.Errorf("--instance-name is required")
	}
	return a.containerzDo(func(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
		return a.containerzLog(ctx, t, c)
	})
}

func (a *App) containerzLog(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
	req := &containerz.LogRequest{
		InstanceName: a.Config.ContainerzLogInstanceName,
		Follow:       a.Config.ContainerzLogFollow,
	}
	stream, err := c.Log(ctx, req)
	if err != nil {
		return fmt.Errorf("Log RPC failed: %v", err)
	}
	for {
		rsp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("log stream error: %v", err)
		}
		fmt.Printf("[%s] %s\n", t.Config.Address, rsp.Msg)
	}
	return nil
}
