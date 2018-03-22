package repository

import (
	"fmt"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/rerorero/meshem/src/model"
	"github.com/rerorero/meshem/src/utils"
)

type discoveryConsul struct {
	consul     *utils.Consul
	globalName string
}

const (
	// DefaultGlobalServiceName is default service name
	DefaultGlobalServiceName = "meshem_envoy"
)

// NewDiscoveryConsul creates DiscoverRepository instance which uses a consul as datastore.
func NewDiscoveryConsul(consul *utils.Consul, globalName string) DiscoveryRepository {
	gn := globalName
	if len(gn) == 0 {
		gn = DefaultGlobalServiceName
	}
	return &discoveryConsul{
		consul:     consul,
		globalName: gn,
	}
}

// Register registers an admin endpoint of host to the consul cagtalog
func (dc *discoveryConsul) Register(host model.Host, tags map[string]string) error {
	adminAddr := host.GetAdminAddr()

	service := &api.AgentService{
		Service: dc.globalName,
		Port:    int(adminAddr.Port),
	}

	reg := &api.CatalogRegistration{
		Node:       host.Name,
		Address:    adminAddr.Hostname,
		Datacenter: dc.consul.Datacenter,
		NodeMeta:   tags,
		Service:    service,
	}

	_, err := dc.consul.Client.Catalog().Register(reg, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to register a catalog %+v", host)
	}
	return nil
}

// RegisterInfo finds a registered node by name.
func (dc *discoveryConsul) FindByName(hostname string) (*DiscoveryInfo, bool, error) {
	node, _, err := dc.consul.Client.Catalog().Node(hostname, nil)
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to get a catalog %s", hostname)
	}
	if node == nil {
		return nil, false, nil
	}

	// make address
	addr := model.Address{Hostname: node.Node.Address}
	svc, ok := node.Services[dc.globalName]
	if ok {
		addr.Port = uint32(svc.Port)
	}

	if err != nil {
		return nil, false, fmt.Errorf("unexpected address format %s", node.Node.Address)
	}
	info := DiscoveryInfo{
		Name:    node.Node.Node,
		Address: addr,
		Tags:    node.Node.Meta,
	}
	return &info, true, nil
}

// Unregister deregister host from datastore.
func (dc *discoveryConsul) Unregister(hostname string) error {
	dereg := &api.CatalogDeregistration{
		Node: hostname,
	}
	_, err := dc.consul.Client.Catalog().Deregister(dereg, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to deregister host: %s", hostname)
	}
	return nil
}
