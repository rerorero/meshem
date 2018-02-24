package repository

import (
	"fmt"

	"github.com/rerorero/meshem/src/model"
)

type inventoryHeap struct {
	services []*model.Service
	hosts    []*model.Host
}

// NewInventoryHeap creates a heap inventory instance.
func NewInventoryHeap() InventoryRepository {
	return &inventoryHeap{}
}

func (inv *inventoryHeap) PutHost(host model.Host) error {
	filtered := model.FilterHosts(inv.hosts, func(h *model.Host) bool { return h.Name == host.Name })
	if len(filtered) == 1 {
		*filtered[0] = host
	} else if len(filtered) == 0 {
		inv.hosts = append(inv.hosts, &host)
	} else {
		return fmt.Errorf("duplicate hosts in the heap inventory: %s", host.Name)
	}
	return nil
}

func (inv *inventoryHeap) SelectHostByName(name string) (host model.Host, ok bool, err error) {
	filtered := model.FilterHosts(inv.hosts, func(h *model.Host) bool { return h.Name == name })
	if len(filtered) == 1 {
		host = *filtered[0]
		return host, true, nil
	} else if len(filtered) == 0 {
		return host, false, nil
	}
	return host, false, fmt.Errorf("duplicate hosts in the heap inventory: %s", name)
}

func (inv *inventoryHeap) DeleteHost(name string) (bool, error) {
	filtered := model.FilterHosts(inv.hosts, func(h *model.Host) bool { return h.Name == name })
	if len(filtered) == 1 {
		after := []*model.Host{}
		for _, h := range inv.hosts {
			if filtered[0] != h {
				after = append(after, h)
			}
		}
		inv.hosts = after
		return true, nil
	} else if len(filtered) == 0 {
		return false, nil
	}
	return false, fmt.Errorf("duplicate hosts in the heap inventory: %s", name)
}

func (inv *inventoryHeap) SelectAllHostNames() ([]string, error) {
	return model.MapHostsToString(inv.hosts, func(h *model.Host) string { return h.Name }), nil
}

func (inv *inventoryHeap) SelectAllHosts() (hosts []model.Host, err error) {
	for _, p := range inv.hosts {
		hosts = append(hosts, *p)
	}
	return hosts, nil
}

func (inv *inventoryHeap) SelectHostsOfService(service string) (hosts []model.Host, err error) {
	svc, ok, err := inv.SelectServiceByName(service)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("No such service: %s", service)
	}

	for _, hostname := range svc.HostNames {
		host, ok, err := inv.SelectHostByName(hostname)
		if err != nil {
			return nil, err
		}
		if ok {
			hosts = append(hosts, host)
		}
	}

	return hosts, nil
}

func (inv *inventoryHeap) PutService(svc model.Service, version model.Version) error {
	svc.Version = version
	filtered := model.FilterServices(inv.services, func(h *model.Service) bool { return h.Name == svc.Name })
	if len(filtered) == 1 {
		*filtered[0] = svc
	} else if len(filtered) == 0 {
		inv.services = append(inv.services, &svc)
	} else {
		return fmt.Errorf("duplicate services in the heap inventory: %s", svc.Name)
	}
	return nil
}

func (inv *inventoryHeap) SelectServiceByName(name string) (model.Service, bool, error) {
	filtered := model.FilterServices(inv.services, func(h *model.Service) bool { return h.Name == name })
	if len(filtered) == 1 {
		return *filtered[0], true, nil
	} else if len(filtered) == 0 {
		return model.Service{}, false, nil
	}
	return model.Service{}, false, fmt.Errorf("duplicate services in the heap inventory: %s", name)
}

func (inv *inventoryHeap) DeleteService(name string) (bool, error) {
	filtered := model.FilterServices(inv.services, func(h *model.Service) bool { return h.Name == name })
	if len(filtered) == 1 {
		after := []*model.Service{}
		for _, h := range inv.services {
			if filtered[0] != h {
				after = append(after, h)
			}
		}
		inv.services = after
		return true, nil
	} else if len(filtered) == 0 {
		return false, nil
	}
	return false, fmt.Errorf("duplicate services in the heap inventory: %s", name)
}

func (inv *inventoryHeap) SelectAllServiceNames() ([]string, error) {
	return model.MapServicesToString(inv.services, func(h *model.Service) string { return h.Name }), nil
}

func (inv *inventoryHeap) SelectAllServices() (services []model.Service, err error) {
	for _, p := range inv.services {
		services = append(services, *p)
	}
	return services, nil
}

func (inv *inventoryHeap) AddServiceDependency(serviceName string, depend model.DependentService, version model.Version) error {
	svc, ok, err := inv.SelectServiceByName(serviceName)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("No such service: %s", serviceName)
	}

	// update dependency list
	err = svc.AppendDependent(depend)
	if err != nil {
		return err
	}
	return inv.PutService(svc, version)
}

// SelectReferringServiceNamesTo taks names of all services which dependes on the service.
func (inv *inventoryHeap) SelectReferringServiceNamesTo(service string) (referrers []string, err error) {
	for _, svc := range inv.services {
		ok, _ := svc.FindDependentServiceName(service)
		if ok {
			referrers = append(referrers, svc.Name)
		}
	}
	return referrers, nil
}

// RemoveServiceDependency removes a service dependency from the service.
func (inv *inventoryHeap) RemoveServiceDependency(serviceName string, depend string, version model.Version) (bool, error) {
	svc, ok, err := inv.SelectServiceByName(serviceName)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	// update dependency list
	removed := svc.RemoveDependent(depend)
	return removed, inv.PutService(svc, version)
}
