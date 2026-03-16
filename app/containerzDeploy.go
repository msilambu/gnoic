package app

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/karimra/gnoic/api"
	"github.com/openconfig/gnoi/containerz"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// InitContainerzDeployFlags registers CLI flags for the deploy command.
func (a *App) InitContainerzDeployFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.Flags().StringVar(&a.Config.ContainerzDeployFile, "file", "", "local path of the container image or plugin file to deploy")
	cmd.Flags().StringVar(&a.Config.ContainerzDeployImageName, "name", "", "image or plugin name to assign on the target")
	cmd.Flags().StringVar(&a.Config.ContainerzDeployTag, "tag", "latest", "image tag to assign on the target")
	cmd.Flags().BoolVar(&a.Config.ContainerzDeployIsPlugin, "is-plugin", false, "set to true when deploying a plugin file instead of a container image")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

// RunEContainerzDeploy implements the 'gnoic containerz deploy' command.
// It streams the image/plugin file to the target using the Deploy RPC.
func (a *App) RunEContainerzDeploy(cmd *cobra.Command, args []string) error {
	if a.Config.ContainerzDeployFile == "" {
		return fmt.Errorf("--file is required")
	}
	if a.Config.ContainerzDeployImageName == "" {
		return fmt.Errorf("--name is required")
	}
	return a.containerzDo(func(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
		return a.containerzDeploy(ctx, t, c)
	})
}

func (a *App) containerzDeploy(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
	f, err := os.Open(a.Config.ContainerzDeployFile)
	if err != nil {
		return fmt.Errorf("failed to open file %q: %v", a.Config.ContainerzDeployFile, err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %v", err)
	}

	stream, err := c.Deploy(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Deploy stream: %v", err)
	}

	// Step 1 – send ImageTransfer metadata.
	if err := stream.Send(&containerz.DeployRequest{
		Request: &containerz.DeployRequest_ImageTransfer{
			ImageTransfer: &containerz.ImageTransfer{
				Name:      a.Config.ContainerzDeployImageName,
				Tag:       a.Config.ContainerzDeployTag,
				ImageSize: uint64(fi.Size()),
				IsPlugin:  a.Config.ContainerzDeployIsPlugin,
			},
		},
	}); err != nil {
		return fmt.Errorf("failed to send ImageTransfer: %v", err)
	}
	a.Logger.Infof("%q: sent ImageTransfer (name=%s, tag=%s, size=%d, plugin=%v)",
		t.Config.Address,
		a.Config.ContainerzDeployImageName,
		a.Config.ContainerzDeployTag,
		fi.Size(),
		a.Config.ContainerzDeployIsPlugin,
	)

	// Step 2 – wait for ImageTransferReady which tells us the chunk size.
	rsp, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive ImageTransferReady: %v", err)
	}
	ready := rsp.GetImageTransferReady()
	if ready == nil {
		return fmt.Errorf("expected ImageTransferReady, got %T", rsp.GetResponse())
	}
	chunkSize := int(ready.ChunkSize)
	if chunkSize <= 0 {
		chunkSize = 4 * 1024 * 1024 // default 4 MB
	}
	a.Logger.Infof("%q: ImageTransferReady, chunk_size=%d", t.Config.Address, chunkSize)

	// Step 3 – stream file content in chunks.
	buf := make([]byte, chunkSize)
	var totalSent int64
	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			if err := stream.Send(&containerz.DeployRequest{
				Request: &containerz.DeployRequest_Content{Content: buf[:n]},
			}); err != nil {
				return fmt.Errorf("failed to send content chunk: %v", err)
			}
			totalSent += int64(n)

			// Read any progress responses (non-blocking style – server may not send for every chunk).
			// We simply log progress if the server sends it.
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("error reading file: %v", readErr)
		}
	}
	a.Logger.Infof("%q: sent %d bytes", t.Config.Address, totalSent)

	// Step 4 – send ImageTransferEnd to signal completion.
	if err := stream.Send(&containerz.DeployRequest{
		Request: &containerz.DeployRequest_ImageTransferEnd{
			ImageTransferEnd: &containerz.ImageTransferEnd{},
		},
	}); err != nil {
		return fmt.Errorf("failed to send ImageTransferEnd: %v", err)
	}

	// Step 5 – drain remaining responses until ImageTransferSuccess or error.
	for {
		rsp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error receiving deploy response: %v", err)
		}
		a.printMsg(t.Config.Address, rsp)
		switch v := rsp.GetResponse().(type) {
		case *containerz.DeployResponse_ImageTransferProgress:
			a.Logger.Infof("%q: transfer progress: %d bytes received by server",
				t.Config.Address, v.ImageTransferProgress.BytesReceived)
		case *containerz.DeployResponse_ImageTransferSuccess:
			a.Logger.Infof("%q: ImageTransferSuccess: name=%s tag=%s size=%d",
				t.Config.Address,
				v.ImageTransferSuccess.Name,
				v.ImageTransferSuccess.Tag,
				v.ImageTransferSuccess.ImageSize,
			)
			fmt.Printf("[%s] Deploy succeeded: %s:%s (%d bytes)\n",
				t.Config.Address,
				v.ImageTransferSuccess.Name,
				v.ImageTransferSuccess.Tag,
				v.ImageTransferSuccess.ImageSize,
			)
			return nil
		case *containerz.DeployResponse_ImageTransferError:
			return fmt.Errorf("image transfer error: %s", v.ImageTransferError.GetMessage())
		}
	}
	return nil
}
