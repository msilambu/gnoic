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

// ListImage

// InitContainerzListImageFlags registers flags for the 'image list' command.
func (a *App) InitContainerzListImageFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.Flags().Int32Var(&a.Config.ContainerzImageListLimit, "limit", 0, "max number of images to return (0 = unlimited)")
	cmd.Flags().StringSliceVar(&a.Config.ContainerzImageListFilter, "filter", []string{}, "filters in key=value format (repeatable)")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

// RunEContainerzListImage implements 'gnoic containerz image list'.
func (a *App) RunEContainerzListImage(cmd *cobra.Command, args []string) error {
	return a.containerzDo(func(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
		return a.containerzListImage(ctx, t, c)
	})
}

func (a *App) containerzListImage(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
	req := &containerz.ListImageRequest{
		Limit:  a.Config.ContainerzImageListLimit,
		Filter: parseKVFilter(a.Config.ContainerzImageListFilter),
	}
	stream, err := c.ListImage(ctx, req)
	if err != nil {
		return fmt.Errorf("ListImage RPC failed: %v", err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"TARGET", "ID", "IMAGE NAME", "TAG"})
	formatTable(table)

	count := 0
	for {
		rsp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ListImage stream error: %v", err)
		}
		a.printMsg(t.Config.Address, rsp)
		table.Append([]string{t.Config.Address, rsp.Id, rsp.ImageName, rsp.Tag})
		count++
	}
	if count > 0 {
		table.Render()
	} else {
		fmt.Printf("[%s] No images found\n", t.Config.Address)
	}
	return nil
}

// RemoveImage

// InitContainerzRemoveImageFlags registers flags for the 'image remove' command.
func (a *App) InitContainerzRemoveImageFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.Flags().StringVar(&a.Config.ContainerzImageRemoveName, "name", "", "image name to remove (required)")
	cmd.Flags().StringVar(&a.Config.ContainerzImageRemoveTag, "tag", "", "image tag to remove")
	cmd.Flags().BoolVar(&a.Config.ContainerzImageRemoveForce, "force", false, "force removal even if image is in use")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

// RunEContainerzRemoveImage implements 'gnoic containerz image remove'.
func (a *App) RunEContainerzRemoveImage(cmd *cobra.Command, args []string) error {
	if a.Config.ContainerzImageRemoveName == "" {
		return fmt.Errorf("--name is required")
	}
	return a.containerzDo(func(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
		return a.containerzRemoveImage(ctx, t, c)
	})
}

func (a *App) containerzRemoveImage(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error {
	req := &containerz.RemoveImageRequest{
		Name:  a.Config.ContainerzImageRemoveName,
		Tag:   a.Config.ContainerzImageRemoveTag,
		Force: a.Config.ContainerzImageRemoveForce,
	}
	rsp, err := c.RemoveImage(ctx, req)
	if err != nil {
		return fmt.Errorf("RemoveImage RPC failed: %v", err)
	}
	a.printMsg(t.Config.Address, rsp)
	fmt.Printf("[%s] Image %s:%s removed successfully\n",
		t.Config.Address, a.Config.ContainerzImageRemoveName, a.Config.ContainerzImageRemoveTag)
	return nil
}
