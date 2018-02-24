package repository

import "github.com/rerorero/meshem/src/model"

// InventoryRepository provides interface to control storage which stores the inventories information.
type InventoryRepository interface {
	PutHost(host model.Host) error
	SelectHostByName(name string) (model.Host, bool, error)
	DeleteHost(name string) (bool, error)
	SelectAllHostNames() ([]string, error)
	SelectAllHosts() ([]model.Host, error)
	SelectHostsOfService(service string) ([]model.Host, error)
	PutService(svc model.Service, version model.Version) error
	SelectServiceByName(name string) (model.Service, bool, error)
	DeleteService(name string) (bool, error)
	SelectAllServiceNames() ([]string, error)
	SelectAllServices() ([]model.Service, error)
	AddServiceDependency(serviceName string, depend model.DependentService, version model.Version) error
	RemoveServiceDependency(serviceName string, depend string, version model.Version) (bool, error)
	SelectReferringServiceNamesTo(service string) ([]string, error)
}

type DiscoveryInfo struct {
	Name    string
	Address model.Address
	Tags    map[string]string
}

// DiscoveryRepository provides functions for discovery service registration
type DiscoveryRepository interface {
	Register(host model.Host, tags map[string]string) error
	Unregister(hostname string) error
	FindByName(hostname string) (*DiscoveryInfo, bool, error)
}
