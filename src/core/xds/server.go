package xds

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
	"github.com/pkg/errors"
	mcore "github.com/rerorero/meshem/src/core"
	"github.com/rerorero/meshem/src/model"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type XDSServer interface {
	RunXDS() (*grpc.Server, error)
	RunSnapshotCollector()
}

type xdss struct {
	inventory     mcore.InventoryService
	snapshotCache cache.SnapshotCache
	snapshotGen   SnapshotGen
	conf          model.XDSConf
	ctx           context.Context
	logger        *logrus.Logger
}

// NewXDSServer creates a xds server.
func NewXDSServer(inventory mcore.InventoryService, vb mcore.VersionGenerator, conf model.MeshemConf, ctx context.Context, logger *logrus.Logger) XDSServer {
	return &xdss{
		inventory:     inventory,
		snapshotCache: cache.NewSnapshotCache(conf.XDS.IsADSMode, Hasher{}, &snapshotLogger{logger}),
		snapshotGen:   NewSnapshotGen(inventory, logger, vb, conf.Envoy),
		conf:          conf.XDS,
		ctx:           ctx,
		logger:        logger,
	}
}

type snapshotLogger struct {
	logger *logrus.Logger
}

// Infof logs a formatted informational message.
func (l *snapshotLogger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

// Errorf logs a formatted error message.
func (l *snapshotLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

func (s *xdss) RunXDS() (*grpc.Server, error) {
	grpcServer := grpc.NewServer()
	server := xds.NewServer(s.snapshotCache, nil)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.conf.Port))
	if err != nil {
		return nil, err
	}
	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	v2.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	v2.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	v2.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	v2.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	s.logger.Infof("xDS server listening on %d", s.conf.Port)

	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			s.logger.Error(err)
		}
	}()

	return grpcServer, nil
}

func (s *xdss) RunSnapshotCollector() {
	ticker := time.NewTicker(time.Duration(s.conf.CacheCollectionIntervalMS) * time.Millisecond)
	go func() {
		s.logger.Info("snapshot collector started.")
		for {
			select {
			case <-ticker.C:
				s.saveSnapshots()
			case <-s.ctx.Done():
				ticker.Stop()
				s.logger.Info("snapshot collector finished.")
				return
			}
		}
	}()
}

func (s *xdss) saveSnapshots() error {
	// TODO: Copy all data from the datastore to the heap(repository) to reduce the access to the datastore (and to read consistently when we use an ACID datastore).

	allsvc, err := s.inventory.GetServiceNames()
	if err != nil {
		return err
	}

	for _, svc := range allsvc {
		snapshots, err := s.snapshotGen.MakeSnapshotsOfService(svc)
		if err != nil {
			s.logger.Errorf("failed to generate snapshot: %s", svc)
			s.logger.Error(err)
			continue
		}
		for host, snapshot := range snapshots {
			s.logger.Infof("set snapshot %s: %+v", host.Name, *snapshot)
			err = s.snapshotCache.SetSnapshot(host.Name, *snapshot)
			if err != nil {
				return errors.Wrapf(err, "snapshot failed: %s=%+v of", host.Name, snapshot)
			}
		}
	}

	return nil
}
