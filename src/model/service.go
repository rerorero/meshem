package model

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/rerorero/meshem/src/utils"
)

// DependentService contains service and port
type DependentService struct {
	Name       string `json:"name" yaml:"name"`
	EgressPort uint32 `json:"egressPort" yaml:"egressPort"`
}

// Service contains information of user service.
// TODO: be able to configure more flexible routes.
type Service struct {
	Name              string             `json:"name" yaml:"name"`
	HostNames         []string           `json:"hostNames" yaml:"hostNames"`
	DependentServices []DependentService `json:"dependentServices" yaml:"dependentServices"`
	Protocol          string             `json:"protocol" yaml:"protocol"`
	TraceSpan         string             `json:"trace_sapn" yaml:"trace_span"`
	Version           Version            `json:"version" yaml:"version"`
}

// IdempotentServiceParam is used as a parameter by updating idempotently
type IdempotentServiceParam struct {
	Protocol          string             `json:"protocol" yaml:"protocol"`
	Hosts             []Host             `json:"hosts" yaml:"hosts"`
	DependentServices []DependentService `json:"dependentServices" yaml:"dependentServices"`
}

const (
	// ProtocolHTTP is for HTTP service
	ProtocolHTTP = "HTTP"
	// ProtocolTCP is for TCP service
	ProtocolTCP = "TCP"
)

var (
	rServiceName = regexp.MustCompile(`^[A-Za-z0-9_\-]{1,64}$`)
	allProtocol  = []string{ProtocolHTTP, ProtocolTCP}
)

// NewService creates a new service instance.
func NewService(name string, protocol string) Service {
	// currently trace span is same as service name.
	return Service{Name: name, Protocol: protocol, TraceSpan: name}
}

// Validate checks Service object format.
func (s *Service) Validate() error {
	err := validateServiceName(s.Name)
	if err != nil {
		return err
	}

	for _, h := range s.HostNames {
		err := validateHostname(h)
		if err != nil {
			return err
		}
	}

	// check the service names and port duplicates
	ports := map[uint32]string{}
	svcNames := map[string]bool{}
	var i int
	for i = 0; i < len(s.DependentServices); i++ {
		err := validateServiceName(s.DependentServices[i].Name)
		if err != nil {
			return err
		}
		if s.DependentServices[i].EgressPort == 0 {
			return fmt.Errorf("invalid egress port number(%d) of service=%s", s.DependentServices[i].EgressPort, s.DependentServices[i].Name)
		}
		if s2, ok := ports[s.DependentServices[i].EgressPort]; ok {
			return fmt.Errorf("duplicate service port=%d used for %s and %s", s.DependentServices[i].EgressPort, s.DependentServices[i].Name, s2)
		}
		if _, ok := svcNames[s.DependentServices[i].Name]; ok {
			return fmt.Errorf("duplicate dependent service names: %s", s.DependentServices[i].Name)
		}
		svcNames[s.DependentServices[i].Name] = true
		ports[s.DependentServices[i].EgressPort] = s.DependentServices[i].Name
	}

	_, ok := utils.ContainsString(allProtocol, s.Protocol)
	if !ok {
		return fmt.Errorf("%s is invalid protocol", s.Protocol)
	}

	return nil
}

// AppendDependent appends a new dpendent service.
func (s *Service) AppendDependent(dependent DependentService) error {
	// check dupclicates
	if ok, dup := s.FindDependentServicePort(dependent.EgressPort); ok {
		return fmt.Errorf("the port=%d is already used by %s", dependent.EgressPort, dup.Name)
	}
	if ok, _ := s.FindDependentServiceName(dependent.Name); ok {
		return fmt.Errorf("the service=%s is already referenced by %s", dependent.Name, s.Name)
	}
	if dependent.EgressPort == 0 {
		return fmt.Errorf("invalid egress port number(%d) of service=%s", dependent.EgressPort, dependent.Name)
	}
	s.DependentServices = append(s.DependentServices, dependent)
	return nil
}

// RemoveDependent removes a dependent service.
func (s *Service) RemoveDependent(name string) bool {
	updated := false
	services := []DependentService{}
	var i int
	for i = 0; i < len(s.DependentServices); i++ {
		if s.DependentServices[i].Name == name {
			updated = true
		} else {
			services = append(services, s.DependentServices[i])
		}
	}
	s.DependentServices = services
	return updated
}

// FindDependentServicePort finds a dependent service which uses the port.
func (s *Service) FindDependentServicePort(port uint32) (bool, *DependentService) {
	var i int
	for i = 0; i < len(s.DependentServices); i++ {
		if s.DependentServices[i].EgressPort == port {
			return true, &s.DependentServices[i]
		}
	}
	return false, nil
}

// FindDependentServiceName finds a dependent service.
func (s *Service) FindDependentServiceName(name string) (bool, *DependentService) {
	var i int
	for i = 0; i < len(s.DependentServices); i++ {
		if s.DependentServices[i].Name == name {
			return true, &s.DependentServices[i]
		}
	}
	return false, nil
}

// DependentServiceNames returns dependent service names.
func (s *Service) DependentServiceNames() []string {
	names := make([]string, len(s.DependentServices))
	var i int
	for i = 0; i < len(s.DependentServices); i++ {
		names[i] = s.DependentServices[i].Name
	}
	return names
}

func compareServiceDependent(l *DependentService, r *DependentService) int {
	if l.Name != r.Name {
		return strings.Compare(l.Name, r.Name)
	}
	return int(r.EgressPort) - int(l.EgressPort)
}

// EqualsServiceDependencies compares two DependentService slices.
func EqualsServiceDependencies(l []DependentService, r []DependentService) bool {
	if len(l) != len(r) {
		return false
	}
	sort.Slice(l, func(i, j int) bool {
		return compareServiceDependent(&l[i], &l[j]) > 0
	})
	sort.Slice(r, func(i, j int) bool {
		return compareServiceDependent(&r[i], &r[j]) > 0
	})
	var i int
	for i = 0; i < len(l); i++ {
		if !reflect.DeepEqual(l[i], r[i]) {
			return false
		}
	}
	return true
}

func validateServiceName(s string) error {
	if !rServiceName.MatchString(s) {
		return errors.New("service name must consist of alphanumeric characters, underscores and dashes, and less than 64 characters")
	}
	return nil
}

// FilterServices filteres a slice of service via prediction function.
func FilterServices(services []*Service, pred func(*Service) bool) (filtered []*Service) {
	for _, svc := range services {
		if pred(svc) {
			filtered = append(filtered, svc)
		}
	}
	return filtered
}

// MapServicesToString transforms a slice of service to a string slice.
func MapServicesToString(services []*Service, f func(*Service) string) (mapped []string) {
	for _, host := range services {
		mapped = append(mapped, f(host))
	}
	return mapped
}

// NewService creates a Service object from an IdempontentServiceParam.
func (param *IdempotentServiceParam) NewService(name string) Service {
	hostnames := make([]string, len(param.Hosts))
	var i int
	for i = 0; i < len(param.Hosts); i++ {
		hostnames[i] = param.Hosts[i].Name
	}
	// currently trace span is same as service name.
	return Service{
		Name:              name,
		HostNames:         hostnames,
		DependentServices: param.DependentServices,
		Protocol:          param.Protocol,
		TraceSpan:         name,
	}
}

// NewIdempotentService creates an IdempotentServiceParam object from a service and its hosts.
func NewIdempotentService(svc *Service, hosts []Host) IdempotentServiceParam {
	return IdempotentServiceParam{
		Protocol:          svc.Protocol,
		Hosts:             hosts,
		DependentServices: svc.DependentServices,
	}
}
