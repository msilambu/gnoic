package app

// containerzMockServer provides a static, in-process mock implementation of
// the gNOI Containerz service. It is intentionally simple: every RPC returns
// a canned static reply so that the gnoic client commands can be exercised
// end-to-end without a real container runtime.
//
// Start the mock server via:
//
//	gnoic server --address :9339 --containerz
//
// Then invoke any containerz command against it, for example:
//
//	gnoic -a localhost:9339 --insecure containerz image-list
//	gnoic -a localhost:9339 --insecure containerz container-list
//	gnoic -a localhost:9339 --insecure containerz volume-list
//	gnoic -a localhost:9339 --insecure containerz plugin-list

import (
	"context"
	"fmt"
	"io"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/openconfig/gnoi/containerz"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// containerzMockServer satisfies containerz.ContainerzServer.
type containerzMockServer struct {
	containerz.UnimplementedContainerzServer
	logger *log.Entry
}

// Deploy

func (s *containerzMockServer) Deploy(stream containerz.Containerz_DeployServer) error {
	s.logger.Info("Deploy: stream opened")

	// Expect the first message to be an ImageTransfer.
	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Internal, "failed to receive initial message: %v", err)
	}
	it := req.GetImageTransfer()
	if it == nil {
		return status.Errorf(codes.InvalidArgument, "first message must be ImageTransfer")
	}
	s.logger.Infof("Deploy: ImageTransfer name=%s tag=%s size=%d plugin=%v",
		it.Name, it.Tag, it.ImageSize, it.IsPlugin)

	// Inform the client of the preferred chunk size (4 MB).
	if err := stream.Send(&containerz.DeployResponse{
		Response: &containerz.DeployResponse_ImageTransferReady{
			ImageTransferReady: &containerz.ImageTransferReady{ChunkSize: 4 * 1024 * 1024},
		},
	}); err != nil {
		return err
	}

	var totalBytes uint64
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "stream recv error: %v", err)
		}
		switch v := req.GetRequest().(type) {
		case *containerz.DeployRequest_Content:
			totalBytes += uint64(len(v.Content))
			// Emit a progress update after each ~1 MB boundary.
			if totalBytes%(1024*1024) < uint64(len(v.Content)) {
				_ = stream.Send(&containerz.DeployResponse{
					Response: &containerz.DeployResponse_ImageTransferProgress{
						ImageTransferProgress: &containerz.ImageTransferProgress{
							BytesReceived: totalBytes,
						},
					},
				})
			}
		case *containerz.DeployRequest_ImageTransferEnd:
			s.logger.Infof("Deploy: ImageTransferEnd – total %d bytes", totalBytes)
		}
	}

	return stream.Send(&containerz.DeployResponse{
		Response: &containerz.DeployResponse_ImageTransferSuccess{
			ImageTransferSuccess: &containerz.ImageTransferSuccess{
				Name:      it.Name,
				Tag:       it.Tag,
				ImageSize: it.ImageSize,
			},
		},
	})
}

// ListImage

func (s *containerzMockServer) ListImage(req *containerz.ListImageRequest, stream containerz.Containerz_ListImageServer) error {
	s.logger.Infof("ListImage: limit=%d filters=%v", req.Limit, req.Filter)
	staticImages := []struct{ id, name, tag string }{
		{"sha256:aabbcc001122", "nginx", "1.25"},
		{"sha256:ddeeff334455", "alpine", "3.18"},
		{"sha256:112233aabbcc", "my-app", "v1.0.0"},
	}
	for _, img := range staticImages {
		if err := stream.Send(&containerz.ListImageResponse{
			Id:        img.id,
			ImageName: img.name,
			Tag:       img.tag,
		}); err != nil {
			return err
		}
	}
	return nil
}

// RemoveImage

func (s *containerzMockServer) RemoveImage(_ context.Context, req *containerz.RemoveImageRequest) (*containerz.RemoveImageResponse, error) {
	s.logger.Infof("RemoveImage: name=%s tag=%s force=%v", req.Name, req.Tag, req.Force)
	return &containerz.RemoveImageResponse{}, nil
}

// StartContainer

func (s *containerzMockServer) StartContainer(_ context.Context, req *containerz.StartContainerRequest) (*containerz.StartContainerResponse, error) {
	s.logger.Infof("StartContainer: image=%s:%s instance=%s", req.ImageName, req.Tag, req.InstanceName)
	name := req.InstanceName
	if name == "" {
		name = fmt.Sprintf("mock-%s-%d", req.ImageName, time.Now().Unix())
	}
	return &containerz.StartContainerResponse{
		Response: &containerz.StartContainerResponse_StartOk{
			StartOk: &containerz.StartOK{InstanceName: name},
		},
	}, nil
}

// StopContainer

func (s *containerzMockServer) StopContainer(_ context.Context, req *containerz.StopContainerRequest) (*containerz.StopContainerResponse, error) {
	s.logger.Infof("StopContainer: instance=%s force=%v restart=%v", req.InstanceName, req.Force, req.Restart)
	return &containerz.StopContainerResponse{}, nil
}

// ListContainer

func (s *containerzMockServer) ListContainer(req *containerz.ListContainerRequest, stream containerz.Containerz_ListContainerServer) error {
	s.logger.Infof("ListContainer: all=%v limit=%d", req.All, req.Limit)
	staticContainers := []struct {
		id, name, image string
		st              containerz.ListContainerResponse_Status
	}{
		{"ctr-001", "web-server", "nginx:1.25", containerz.ListContainerResponse_RUNNING},
		{"ctr-002", "db-primary", "postgres:15", containerz.ListContainerResponse_RUNNING},
		{"ctr-003", "cache", "redis:7", containerz.ListContainerResponse_STOPPED},
	}
	for _, c := range staticContainers {
		if !req.All && c.st != containerz.ListContainerResponse_RUNNING {
			continue
		}
		if err := stream.Send(&containerz.ListContainerResponse{
			Id:        c.id,
			Name:      c.name,
			ImageName: c.image,
			Status:    c.st,
		}); err != nil {
			return err
		}
	}
	return nil
}

// RemoveContainer

func (s *containerzMockServer) RemoveContainer(_ context.Context, req *containerz.RemoveContainerRequest) (*containerz.RemoveContainerResponse, error) {
	s.logger.Infof("RemoveContainer: name=%s force=%v", req.Name, req.Force)
	return &containerz.RemoveContainerResponse{}, nil
}

// UpdateContainer

func (s *containerzMockServer) UpdateContainer(_ context.Context, req *containerz.UpdateContainerRequest) (*containerz.UpdateContainerResponse, error) {
	s.logger.Infof("UpdateContainer: instance=%s image=%s:%s async=%v",
		req.InstanceName, req.ImageName, req.ImageTag, req.Async)
	return &containerz.UpdateContainerResponse{
		Response: &containerz.UpdateContainerResponse_UpdateOk{
			UpdateOk: &containerz.UpdateOK{
				InstanceName: req.InstanceName,
				IsAsync:      req.Async,
			},
		},
	}, nil
}

// Log

func (s *containerzMockServer) Log(req *containerz.LogRequest, stream containerz.Containerz_LogServer) error {
	s.logger.Infof("Log: instance=%s follow=%v", req.InstanceName, req.Follow)
	staticLines := []string{
		fmt.Sprintf("[mock] container %s starting up...", req.InstanceName),
		"[mock] listening on :8080",
		"[mock] GET /health 200 12ms",
		"[mock] GET /metrics 200 5ms",
	}
	for _, line := range staticLines {
		if err := stream.Send(&containerz.LogResponse{Msg: line}); err != nil {
			return err
		}
		time.Sleep(30 * time.Millisecond)
	}
	if req.Follow {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-stream.Context().Done():
				return nil
			case t := <-ticker.C:
				i++
				if err := stream.Send(&containerz.LogResponse{
					Msg: fmt.Sprintf("[mock] heartbeat #%d %s", i, t.Format(time.RFC3339)),
				}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// CreateVolume

func (s *containerzMockServer) CreateVolume(_ context.Context, req *containerz.CreateVolumeRequest) (*containerz.CreateVolumeResponse, error) {
	s.logger.Infof("CreateVolume: name=%s driver=%v", req.Name, req.Driver)
	return &containerz.CreateVolumeResponse{Name: req.Name}, nil
}

// RemoveVolume

func (s *containerzMockServer) RemoveVolume(_ context.Context, req *containerz.RemoveVolumeRequest) (*containerz.RemoveVolumeResponse, error) {
	s.logger.Infof("RemoveVolume: name=%s force=%v", req.Name, req.Force)
	return &containerz.RemoveVolumeResponse{}, nil
}

// ListVolume

func (s *containerzMockServer) ListVolume(req *containerz.ListVolumeRequest, stream containerz.Containerz_ListVolumeServer) error {
	s.logger.Infof("ListVolume: filters=%v", req.Filter)
	staticVolumes := []struct{ name, driver string }{
		{"data-vol", "local"},
		{"config-vol", "local"},
		{"plugin-storage", "custom"},
	}
	for _, v := range staticVolumes {
		if err := stream.Send(&containerz.ListVolumeResponse{
			Name:    v.name,
			Driver:  v.driver,
			Created: timestamppb.New(time.Now().Add(-24 * time.Hour)),
		}); err != nil {
			return err
		}
	}
	return nil
}

// StartPlugin

func (s *containerzMockServer) StartPlugin(_ context.Context, req *containerz.StartPluginRequest) (*containerz.StartPluginResponse, error) {
	s.logger.Infof("StartPlugin: name=%s instance=%s", req.Name, req.InstanceName)
	return &containerz.StartPluginResponse{InstanceName: req.InstanceName}, nil
}

//  StopPlugin 

func (s *containerzMockServer) StopPlugin(_ context.Context, req *containerz.StopPluginRequest) (*containerz.StopPluginResponse, error) {
	s.logger.Infof("StopPlugin: instance=%s", req.InstanceName)
	return &containerz.StopPluginResponse{}, nil
}

//  ListPlugins 

func (s *containerzMockServer) ListPlugins(_ context.Context, req *containerz.ListPluginsRequest) (*containerz.ListPluginsResponse, error) {
	s.logger.Infof("ListPlugins: instance_name=%q", req.InstanceName)
	plugins := []*containerz.Plugin{
		{Id: "plg-001", InstanceName: "net-monitor-v1", Config: `{"interval":"30s"}`},
		{Id: "plg-002", InstanceName: "flow-collector", Config: `{"port":9995}`},
	}
	if req.InstanceName != "" {
		filtered := make([]*containerz.Plugin, 0)
		for _, p := range plugins {
			if p.InstanceName == req.InstanceName {
				filtered = append(filtered, p)
			}
		}
		plugins = filtered
	}
	return &containerz.ListPluginsResponse{Plugins: plugins}, nil
}

// RemovePlugin

func (s *containerzMockServer) RemovePlugin(_ context.Context, req *containerz.RemovePluginRequest) (*containerz.RemovePluginResponse, error) {
	s.logger.Infof("RemovePlugin: instance=%s", req.InstanceName)
	return &containerz.RemovePluginResponse{}, nil
}
