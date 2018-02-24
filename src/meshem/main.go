package main

import (
	"context"
	"flag"
	"net/url"
	"os"

	"github.com/rerorero/meshem/src"

	"github.com/pkg/errors"
	"github.com/rerorero/meshem/src/core"
	"github.com/rerorero/meshem/src/core/ctlapi"
	"github.com/rerorero/meshem/src/core/xds"
	"github.com/rerorero/meshem/src/repository"
	"github.com/rerorero/meshem/src/utils"

	"github.com/rerorero/meshem/src/model"
	"github.com/sirupsen/logrus"
)

var (
	confPath = flag.String("conf.file", "/etc/meshem.yaml", "Path to configuration file.")
	logger   = logrus.New()
)

func main() {
	flag.Parse()
	ctx := context.Background()

	logger.Infof("meshem server version=%s", src.ServerVersion())

	// read config file
	conf, err := model.NewMeshemConfFile(*confPath)
	if err != nil {
		ExitError(errors.Wrapf(err, "failed to read config file: %s", *confPath))
	}

	// consul
	consul, err := newConsulFromConf(&conf.Consul)
	if err != nil {
		ExitError(err)
	}

	// inventory repository
	inventoryRepo := repository.NewInventoryConsul(consul)

	// service discovery repository
	var discoveryRepo repository.DiscoveryRepository
	if conf.Discovery != nil {
		switch conf.Discovery.Type {
		case model.DiscoveryTypeConsul:
			discoveryConsul, err := newConsulFromConf(conf.Discovery.Consul)
			if err != nil {
				ExitError(err)
			}
			discoveryRepo = repository.NewDiscoveryConsul(discoveryConsul, repository.DefaultGlobalServiceName)
		}
	}

	// inventory
	versionGen := core.NewCurrentTimeGenerator()
	inventoryService := core.NewInventoryService(inventoryRepo, discoveryRepo, versionGen, logger)
	xdsServer := xds.NewXDSServer(inventoryService, versionGen, *conf, ctx, logger)

	// start control api server
	apiServer := ctlapi.NewServer(inventoryService, conf.CtlAPI, logger)
	err = apiServer.Run()
	if err != nil {
		ExitError(errors.Wrap(err, "failed to strat control API server"))
	}

	// start snapshot collector
	xdsServer.RunSnapshotCollector()

	// start xds server
	grpcServer, err := xdsServer.RunXDS()
	if err != nil {
		ExitError(errors.Wrap(err, "failed to run xds server."))
	}

	<-ctx.Done()
	grpcServer.GracefulStop()
	logger.Info("xds shutdown")

	os.Exit(0)
}

// ExitError exits on error
func ExitError(err error) {
	logger.Error(err)
	os.Exit(1)
}

func newConsulFromConf(conf *model.ConsulConf) (*utils.Consul, error) {
	consulURL, err := url.Parse(conf.URL)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid consul url: %s", conf.URL)
	}
	consul, err := utils.NewConsul(consulURL, conf.Token, conf.Datacenter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize consul")
	}
	return consul, nil
}
