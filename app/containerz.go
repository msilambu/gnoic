package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/karimra/gnoic/api"
	"github.com/openconfig/gnoi/containerz"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/grpc/metadata"
)

// InitContainerzFlags initializes the top-level containerz command flags.
func (a *App) InitContainerzFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.LocalFlags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

// containerzDo is a helper that dials each target, runs fn concurrently, and
// collects errors.  fn receives a ready ContainerzClient for one target.
func (a *App) containerzDo(fn func(ctx context.Context, t *api.Target, c containerz.ContainerzClient) error) error {
	targets, err := a.GetTargets()
	if err != nil {
		return err
	}

	type result struct {
		TargetError
	}

	numTargets := len(targets)
	responseChan := make(chan result, numTargets)

	a.wg.Add(numTargets)
	for _, t := range targets {
		go func(t *api.Target) {
			defer a.wg.Done()
			ctx, cancel := context.WithCancel(a.ctx)
			defer cancel()
			ctx = metadata.AppendToOutgoingContext(ctx,
				"username", *t.Config.Username,
				"password", *t.Config.Password,
			)
			if err := t.CreateGrpcClient(ctx, a.createBaseDialOpts()...); err != nil {
				responseChan <- result{TargetError{TargetName: t.Config.Address, Err: err}}
				return
			}
			defer t.Close()
			c := t.ContainerzClient()
			responseChan <- result{TargetError{
				TargetName: t.Config.Address,
				Err:        fn(ctx, t, c),
			}}
		}(t)
	}
	a.wg.Wait()
	close(responseChan)

	errs := make([]error, 0, numTargets)
	for r := range responseChan {
		if r.Err != nil {
			wErr := fmt.Errorf("%q containerz RPC failed: %v", r.TargetName, r.Err)
			a.Logger.Error(wErr)
			errs = append(errs, wErr)
		}
	}
	return a.handleErrs(errs)
}

// parseKVFilter converts "key=value" strings into containerz ListImageRequest Filter messages.
func parseKVFilter(filters []string) []*containerz.ListImageRequest_Filter {
	result := make([]*containerz.ListImageRequest_Filter, 0, len(filters))
	for _, f := range filters {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) != 2 {
			continue
		}
		result = append(result, &containerz.ListImageRequest_Filter{
			Key:   parts[0],
			Value: []string{parts[1]},
		})
	}
	return result
}

// parseContainerFilter converts "key=value" strings into ListContainerRequest Filter messages.
func parseContainerFilter(filters []string) []*containerz.ListContainerRequest_Filter {
	result := make([]*containerz.ListContainerRequest_Filter, 0, len(filters))
	for _, f := range filters {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) != 2 {
			continue
		}
		result = append(result, &containerz.ListContainerRequest_Filter{
			Key:   parts[0],
			Value: []string{parts[1]},
		})
	}
	return result
}

// parseVolumeFilter converts "key=value" strings into ListVolumeRequest Filter messages.
func parseVolumeFilter(filters []string) []*containerz.ListVolumeRequest_Filter {
	result := make([]*containerz.ListVolumeRequest_Filter, 0, len(filters))
	for _, f := range filters {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) != 2 {
			continue
		}
		result = append(result, &containerz.ListVolumeRequest_Filter{
			Key:   parts[0],
			Value: []string{parts[1]},
		})
	}
	return result
}

// parseKVMap converts "key=value" strings into a map.
func parseKVMap(pairs []string) map[string]string {
	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}
