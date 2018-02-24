package core

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	"github.com/rerorero/meshem/src/model"
	"github.com/rerorero/meshem/src/repository"
	"github.com/rerorero/meshem/src/utils"
	"github.com/sirupsen/logrus"
)

// InventoryService is domain service shich manages meshem inventories.
type InventoryService interface {
	RegisterService(name string, protocol string) (model.Service, error)
	UnregisterService(name string) (deleted bool, referer []string, err error)
	GetService(name string) (model.Service, bool, error)
	GetServiceNames() ([]string, error)
	AddServiceDependency(serviceName string, dependServiceNames string, egressPort uint32) error
	RemoveServiceDependency(serviceName string, dependServiceNames string) (bool, error)
	GetRefferersOf(serviceName string) ([]string, error)
	RegisterHost(serviceName, hostName, ingressAddr, substanceAddr, egressHost string) (model.Host, error)
	UnregisterHost(serviceName string, hostName string) (bool, error)
	GetHostByName(name string) (model.Host, bool, error)
	GetHostNames() ([]string, error)
	GetHostsOfService(serviceName string) ([]model.Host, error)
	UpdateHost(serviceName string, hostName string, ingressAddr, substanceAddr, egressHost *string) (host model.Host, err error)
	IdempotentService(serviceName string, param model.IdempotentServiceParam) (changed bool, err error)
}

// TODO: logging
type inventoryService struct {
	repo       repository.InventoryRepository
	discovery  repository.DiscoveryRepository
	versionGen VersionGenerator
	logger     *logrus.Logger
}

// NewInventoryService creates an InventoryService instance.
func NewInventoryService(
	repo repository.InventoryRepository,
	discoery repository.DiscoveryRepository,
	versionGen VersionGenerator,
	logger *logrus.Logger,
) InventoryService {
	return &inventoryService{
		repo:       repo,
		discovery:  discoery,
		versionGen: versionGen,
		logger:     logger,
	}
}

// RegisterService stores a new Service object.
// TODO: with consistency
func (inv *inventoryService) RegisterService(name string, protocol string) (service model.Service, err error) {
	service = model.NewService(name, protocol)

	err = service.Validate()
	if err != nil {
		return service, err
	}

	// check dup
	names, err := inv.repo.SelectAllServiceNames()
	if err != nil {
		return service, err
	}
	_, ok := utils.ContainsString(names, name)
	if ok {
		return service, fmt.Errorf("service %s already exists", name)
	}

	version := inv.versionGen.New()
	err = inv.repo.PutService(service, version)
	if err != nil {
		return service, err
	}
	service.Version = version

	inv.logger.Infof("Service %s is registered! version=%s", name, service.Version)

	return service, nil
}

// UnregisterService removes the Service object and removes dependencies of all service.
func (inv *inventoryService) UnregisterService(name string) (deleted bool, referrers []string, err error) {
	referrers, err = inv.repo.SelectReferringServiceNamesTo(name)
	if err != nil {
		return false, nil, err
	}

	for _, ref := range referrers {
		_, err := inv.RemoveServiceDependency(ref, name)
		if err != nil {
			return false, referrers, err
		}
	}

	// delete service object
	deleted, err = inv.repo.DeleteService(name)
	if err != nil {
		return false, referrers, err
	}

	if deleted {
		inv.logger.Infof("Service %s is deleted!", name)
	}
	return deleted, referrers, nil
}

// GetServiceRelations returns names of service which refferes the service.
func (inv *inventoryService) GetRefferersOf(serviceName string) ([]string, error) {
	return inv.repo.SelectReferringServiceNamesTo(serviceName)
}

// GetService finds a service by name.
func (inv *inventoryService) GetService(name string) (model.Service, bool, error) {
	return inv.repo.SelectServiceByName(name)
}

// GetServiceNames returns all service names
func (inv *inventoryService) GetServiceNames() ([]string, error) {
	return inv.repo.SelectAllServiceNames()
}

// AddServiceDependency adds a new service dependency to the service.
func (inv *inventoryService) AddServiceDependency(serviceName string, dependServiceName string, egressPort uint32) error {
	hosts, err := inv.GetHostsOfService(serviceName)
	if err != nil {
		return nil
	}

	var i int
	for i = 0; i < len(hosts); i++ {
		if hosts[i].IngressAddr.Port == egressPort {
			return fmt.Errorf("port=%d is already used by the ingress of %s", egressPort, hosts[i].Name)
		}
		if hosts[i].SubstanceAddr.Port == egressPort {
			return fmt.Errorf("port=%d is already used by the substance of %s", egressPort, hosts[i].Name)
		}
	}

	depend := model.DependentService{Name: dependServiceName, EgressPort: egressPort}
	version := inv.versionGen.New()
	err = inv.repo.AddServiceDependency(serviceName, depend, version)
	if err != nil {
		return err
	}

	inv.logger.Infof("Added service dependency! service=%s, dep=%s, port=%d, version=%s", serviceName, dependServiceName, egressPort, version)
	return nil
}

// RemoveServiceDependencies removes a service dependency from the service.
func (inv *inventoryService) RemoveServiceDependency(serviceName string, dependServiceName string) (bool, error) {
	version := inv.versionGen.New()
	ok, err := inv.repo.RemoveServiceDependency(serviceName, dependServiceName, version)
	if ok {
		inv.logger.Infof("Removed service dependency! service=%s, dep=%s, version=%s", serviceName, dependServiceName, version)
	}
	return ok, err
}

// RegisterHost stores a new Host object.
// TODO: with consistency
func (inv *inventoryService) RegisterHost(serviceName, hostName, ingressAddr, substanceAddr, egressHost string) (host model.Host, err error) {
	host, err = model.NewHost(hostName, ingressAddr, substanceAddr, egressHost)
	if err != nil {
		return host, err
	}

	err = host.Validate()
	if err != nil {
		return host, err
	}

	// check dup
	names, err := inv.repo.SelectAllHostNames()
	if err != nil {
		return host, err
	}
	_, ok := utils.ContainsString(names, hostName)
	if ok {
		return host, fmt.Errorf("host %s already exists", hostName)
	}

	// get the service
	service, ok, err := inv.GetService(serviceName)
	if err != nil {
		return host, err
	}
	if !ok {
		return host, fmt.Errorf("No such service: %s", serviceName)
	}

	// save the host
	err = inv.repo.PutHost(host)
	if err != nil {
		return host, err
	}
	inv.logger.Infof("Host registered! host=%s, service=%s", hostName, serviceName)

	// update the service list
	version := inv.versionGen.New()
	service.HostNames = append(service.HostNames, hostName)
	err = inv.repo.PutService(service, version)
	if err != nil {
		return host, errors.Wrapf(err, "failed to append a host(%s) to the service(%s)", hostName, serviceName)
	}
	inv.logger.Infof("Added a host to service's host list! host=%s, service=%s, version=%s", hostName, serviceName, version)

	// register the host to discovery service if discovery is available
	err = inv.registerDiscoverService(&service, &host)
	if err != nil {
		return host, err
	}

	return host, nil
}

// UnregisterHost removes the Host object and hostname in host list of a service
func (inv *inventoryService) UnregisterHost(serviceName string, hostName string) (bool, error) {
	svc, ok, err := inv.GetService(serviceName)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	i, ok := utils.ContainsString(svc.HostNames, hostName)
	if !ok {
		return false, nil
	}
	// remove from service's host list and update service version
	svc.HostNames = append(svc.HostNames[:i], svc.HostNames[i+1:]...)
	version := inv.versionGen.New()
	err = inv.repo.PutService(svc, version)
	if err != nil {
		return false, err
	}
	inv.logger.Infof("Removed a host from service's host list! host=%s, service=%s, version=%s", hostName, serviceName, version)

	// unregister the host from discovery service if discovery is available
	err = inv.unregisterDiscoverService(hostName)
	if err != nil {
		return false, err
	}

	ok, err = inv.repo.DeleteHost(hostName)
	if ok {
		inv.logger.Infof("Host is removed! host=%s, service=%s", hostName, serviceName)
	}
	return ok, err
}

// GetHost finds a host by name.
func (inv *inventoryService) GetHostByName(name string) (model.Host, bool, error) {
	return inv.repo.SelectHostByName(name)
}

// GetHostNames returns the names of all hosts
func (inv *inventoryService) GetHostNames() ([]string, error) {
	return inv.repo.SelectAllHostNames()
}

// GetHostOfService returns the objects of all hosts to which the service belongs
func (inv *inventoryService) GetHostsOfService(serviceName string) ([]model.Host, error) {
	return inv.repo.SelectHostsOfService(serviceName)
}

// UpdateHost updates a host.
func (inv *inventoryService) UpdateHost(serviceName string, hostName string, ingressAddr, substanceAddr, egressHost *string) (host model.Host, err error) {
	svc, ok, err := inv.GetService(serviceName)
	if err != nil {
		return host, err
	}
	_, ok = utils.ContainsString(svc.HostNames, hostName)
	if !ok {
		return host, fmt.Errorf("hostname=%s is not found in service=%s", hostName, serviceName)
	}

	host, ok, err = inv.GetHostByName(hostName)
	if err != nil {
		return host, err
	}
	if !ok {
		return host, fmt.Errorf("hostname=%s is not found in service=%s", hostName, serviceName)
	}

	err = host.Update(ingressAddr, substanceAddr, egressHost)
	if err != nil {
		return host, err
	}

	err = host.Validate()
	if err != nil {
		return host, err
	}

	// save the host
	version := inv.versionGen.New()
	err = inv.repo.PutHost(host)
	if err != nil {
		return host, err
	}
	inv.logger.Infof("Updated a host! host=%s, service=%s, ia=%v, sa=%v, eh=%v", hostName, serviceName, ingressAddr, substanceAddr, egressHost)

	// update the service version
	err = inv.repo.PutService(svc, version)
	if err != nil {
		return host, errors.Wrapf(err, "failed to update service version(%s)", svc.Name)
	}
	inv.logger.Infof("Updated a service! host=%s, service=%s, version=%s", hostName, serviceName, version)

	return host, nil
}

// IdempotentService updates service and its hosts idempotently.
func (inv *inventoryService) IdempotentService(serviceName string, param model.IdempotentServiceParam) (changed bool, err error) {
	var i int
	// validate
	service := param.NewService(serviceName)
	err = service.Validate()
	if err != nil {
		return changed, err
	}
	paramHostsMap := map[string]*model.Host{}
	for i = 0; i < len(param.Hosts); i++ {
		err = param.Hosts[i].Validate()
		if err != nil {
			return changed, err
		}
		paramHostsMap[param.Hosts[i].Name] = &param.Hosts[i]
	}

	// get the current service state
	currentService, ok, err := inv.GetService(serviceName)
	if err != nil {
		return changed, err
	}

	if ok {
		// update
		// get current host states
		hosts, err := inv.GetHostsOfService(serviceName)
		if err != nil {
			return changed, err
		}
		currentHostsMap := map[string]*model.Host{}
		currentHostNames := make([]string, len(hosts))
		for i = 0; i < len(hosts); i++ {
			currentHostNames[i] = hosts[i].Name
			currentHostsMap[hosts[i].Name] = &hosts[i]
		}

		// compare hostnames
		newHosts := utils.FilterNotContainsString(service.HostNames, currentHostNames)
		delHosts := utils.FilterNotContainsString(currentHostNames, service.HostNames)
		modifiedHosts := utils.IntersectStringSlice(currentHostNames, service.HostNames)
		// register appended hosts
		for _, hostname := range newHosts {
			host, ok := paramHostsMap[hostname]
			if !ok {
				return changed, fmt.Errorf("something wrong, consistency may be broken: %+v, %+v", paramHostsMap, hosts)
			}
			_, err = inv.RegisterHost(service.Name, host.Name, host.IngressAddr.String(), host.SubstanceAddr.String(), host.EgressHost)
			if err != nil {
				return changed, err
			}
			changed = true
		}
		// unregister disappeared hosts
		for _, hostname := range delHosts {
			_, err = inv.UnregisterHost(service.Name, hostname)
			if err != nil {
				return changed, err
			}
			changed = true
		}
		// update the hosts that is modified
		for _, hostname := range modifiedHosts {
			cur, ok1 := currentHostsMap[hostname]
			new, ok2 := paramHostsMap[hostname]
			if !ok1 || !ok2 {
				return changed, fmt.Errorf("something wrong, consistency may be broken: %+v : %+v", paramHostsMap, hosts)
			}
			if !reflect.DeepEqual(cur, new) {
				ingress := new.IngressAddr.String()
				substance := new.SubstanceAddr.String()
				_, err = inv.UpdateHost(serviceName, new.Name, &ingress, &substance, &new.EgressHost)
				if err != nil {
					return changed, err
				}
				changed = true
			}
		}

		// compare service dependencies and protocol
		if (service.Protocol != currentService.Protocol) ||
			(!model.EqualsServiceDependencies(currentService.DependentServices, service.DependentServices)) {
			service.Version = inv.versionGen.New()
			err := inv.repo.PutService(service, service.Version)
			if err != nil {
				return changed, err
			}
			changed = true
		}

	} else {
		// all new ones
		_, err = inv.RegisterService(service.Name, service.Protocol)
		if err != nil {
			return changed, err
		}
		changed = true
		for _, depsvc := range param.DependentServices {
			err = inv.AddServiceDependency(service.Name, depsvc.Name, depsvc.EgressPort)
			if err != nil {
				return changed, err
			}
		}
		for i = 0; i < len(param.Hosts); i++ {
			host := &param.Hosts[i]
			_, err = inv.RegisterHost(service.Name, host.Name, host.IngressAddr.String(), host.SubstanceAddr.String(), host.EgressHost)
			if err != nil {
				return changed, err
			}
		}
	}
	if changed {
		inv.logger.Infof("Updated service via idempotent function! service=%s", serviceName)
	}
	return changed, nil
}

func (inv *inventoryService) registerDiscoverService(svc *model.Service, host *model.Host) error {
	if inv.discovery != nil {
		tags := inv.makeDiscoveryTags(svc, host)
		err := inv.discovery.Register(*host, tags)
		if err != nil {
			return err
		}
		inv.logger.Infof("Registered a host to discovery service! host=%s, service=%s", host.Name, svc.Name)
	}
	return nil
}

func (inv *inventoryService) makeDiscoveryTags(service *model.Service, host *model.Host) map[string]string {
	tags := map[string]string{}
	tags["meshem_service"] = service.Name
	return tags
}

func (inv *inventoryService) unregisterDiscoverService(hostname string) error {
	if inv.discovery != nil {
		err := inv.discovery.Unregister(hostname)
		if err != nil {
			return err
		}
		inv.logger.Infof("Remove a host from discovery service! host=%s", hostname)
	}
	return nil
}
