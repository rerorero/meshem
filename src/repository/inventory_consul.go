package repository

// TODO: Use cas and lock to achieve stronger consistency

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rerorero/meshem/src/model"
	"github.com/rerorero/meshem/src/utils"
)

type inventoryConsul struct {
	consul *utils.Consul
}

const (
	hostPrefix    = "hosts"
	servicePrefix = "services"
)

func NewInventoryConsul(consul *utils.Consul) InventoryRepository {
	return &inventoryConsul{consul: consul}
}

// Put Host object to Consul
func (inventory *inventoryConsul) PutHost(host model.Host) error {
	js, err := json.Marshal(host)
	if err != nil {
		return errors.Wrapf(err, "Failed to marshal Host: %+v", host)
	}
	err = inventory.consul.PutKV(withHostPrefix(host.Name), string(js))
	if err != nil {
		return err
	}
	return nil
}

func (inventory *inventoryConsul) SelectHostByName(name string) (host model.Host, ok bool, err error) {
	js, ok, err := inventory.consul.GetKV(withHostPrefix(name))
	if err != nil {
		return host, false, err
	}
	if !ok {
		return host, false, nil
	}

	err = json.Unmarshal([]byte(js), &host)
	if err != nil {
		return host, false, errors.Wrapf(err, "Host object may be broken: %s", js)
	}

	return host, true, nil
}

// returns (true, nil) if it is deleted
func (inventory *inventoryConsul) DeleteHost(name string) (bool, error) {
	return inventory.consul.DeleteTreeIfExists(withHostPrefix(name))
}

func (inventory *inventoryConsul) SelectAllHostNames() ([]string, error) {
	names, err := inventory.consul.GetSubKeyNames(hostPrefix)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list host names")
	}
	return names, nil
}

func (inventory *inventoryConsul) SelectAllHosts() (hosts []model.Host, err error) {
	names, err := inventory.SelectAllHostNames()
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		host, ok, err := inventory.SelectHostByName(name)
		if err != nil {
			return nil, err
		}
		if ok {
			hosts = append(hosts, host)
		}
	}
	return hosts, nil
}

func (inventory *inventoryConsul) SelectHostsOfService(service string) (hosts []model.Host, err error) {
	svc, ok, err := inventory.SelectServiceByName(service)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("No such service: %s", service)
	}

	for _, hostname := range svc.HostNames {
		host, ok, err := inventory.SelectHostByName(hostname)
		if err != nil {
			return nil, err
		}
		if ok {
			hosts = append(hosts, host)
		}
	}

	return hosts, nil
}

// Put Service object to Consul
func (inventory *inventoryConsul) PutService(svc model.Service, version model.Version) error {
	svc.Version = version
	js, err := json.Marshal(svc)
	if err != nil {
		return errors.Wrapf(err, "Failed to marshal Service: %+v", svc)
	}
	err = inventory.consul.PutKV(withServicePrefix(svc.Name), string(js))
	if err != nil {
		return err
	}
	return nil
}

func (inventory *inventoryConsul) SelectServiceByName(name string) (service model.Service, ok bool, err error) {
	js, ok, err := inventory.consul.GetKV(withServicePrefix(name))
	if err != nil {
		return service, false, err
	}
	if !ok {
		return service, false, nil
	}

	err = json.Unmarshal([]byte(js), &service)
	if err != nil {
		return service, false, errors.Wrapf(err, "Service object may be broken: %s", js)
	}

	return service, true, nil
}

// returns (true, nil) if it is deleted
func (inventory *inventoryConsul) DeleteService(name string) (bool, error) {
	return inventory.consul.DeleteTreeIfExists(withServicePrefix(name))
}

func (inventory *inventoryConsul) SelectAllServiceNames() ([]string, error) {
	names, err := inventory.consul.GetSubKeyNames(servicePrefix)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list service names")
	}
	return names, nil
}

func (inventory *inventoryConsul) SelectAllServices() (services []model.Service, err error) {
	names, err := inventory.SelectAllServiceNames()
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		service, ok, err := inventory.SelectServiceByName(name)
		if err != nil {
			return nil, err
		}
		if ok {
			services = append(services, service)
		}
	}
	return services, nil
}

// AddServiceDependency appends a service dependencey.
func (inventory *inventoryConsul) AddServiceDependency(serviceName string, dependent model.DependentService, version model.Version) error {
	svc, ok, err := inventory.SelectServiceByName(serviceName)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("No such service: %s", serviceName)
	}

	// update dependency list
	err = svc.AppendDependent(dependent)
	if err != nil {
		return err
	}

	return inventory.PutService(svc, version)
}

// RemoveServiceDependency removes a service depndency from service.
func (inventory *inventoryConsul) RemoveServiceDependency(serviceName string, depend string, version model.Version) (bool, error) {
	svc, ok, err := inventory.SelectServiceByName(serviceName)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	// update dependency list
	removed := svc.RemoveDependent(depend)
	return removed, inventory.PutService(svc, version)
}

// SelectReferringServiceNamesTo taks names of all services which dependes on the service.
func (inventory *inventoryConsul) SelectReferringServiceNamesTo(service string) (referrers []string, err error) {
	all, err := inventory.SelectAllServices()
	if err != nil {
		return nil, err
	}
	var i int
	for i = 0; i < len(all); i++ {
		ok, _ := all[i].FindDependentServiceName(service)
		if ok {
			referrers = append(referrers, all[i].Name)
		}
	}
	return referrers, nil
}

// returns an error if key doesn't exist
func (inventory *inventoryConsul) getKVAddressExactly(key string) (*model.Address, error) {
	str, err := inventory.consul.GetKVExactly(key)
	if err != nil {
		return nil, err
	}

	addr, err := model.ParseAddress(str)
	if err != nil {
		return nil, errors.Wrapf(err, "%s=%s is invalid address", key, str)
	}
	return addr, nil
}

func (inventory *inventoryConsul) putKVAddress(key string, addr *model.Address) error {
	return inventory.consul.PutKV(key, addr.String())
}

func withServicePrefix(sub string) string {
	return fmt.Sprintf("%s/%s", servicePrefix, sub)
}

func withHostPrefix(sub string) string {
	return fmt.Sprintf("%s/%s", hostPrefix, sub)
}
