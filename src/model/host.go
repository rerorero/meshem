package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Host contains information of an user managed host.
type Host struct {
	Name          string  `json:"name" yaml:"name"`
	IngressAddr   Address `json:"ingressAddr" yaml:"ingressAddr"`
	SubstanceAddr Address `json:"substanceAddr" yaml:"substanceAddr"`
	EgressHost    string  `json:"egressHost" yaml:"egressHost"`
	// TODO: add an admin port (IngressAddr + admin port(8001))
	// AdminAddr     Address `json:"adminAddr" yaml:"adminAddr"`
}

const (
	// DefaultAdminPort is default value of envoy admin port
	DefaultAdminPort = 8001
)

var (
	rHostName = regexp.MustCompile(`^[A-Za-z0-9_\-]{1,64}$`)
)

// NewHost creates a new Host instance.
func NewHost(name string, ingresAddress string, substanceAddress string, egressHost string) (host Host, err error) {
	ingress, err := ParseAddress(ingresAddress)
	if err != nil {
		return host, err
	}

	substance, err := ParseAddress(substanceAddress)
	if err != nil {
		return host, err
	}

	return Host{
		Name:          name,
		IngressAddr:   *ingress,
		SubstanceAddr: *substance,
		EgressHost:    egressHost,
	}, nil
}

// Validate checks that the host is valid.
func (h *Host) Validate() error {
	err := validateHostname(h.Name)
	if err != nil {
		return err
	}

	if h.IngressAddr == h.SubstanceAddr {
		return fmt.Errorf("duplicate ingress port and egress port (host=%s, addr=%s)", h.Name, h.IngressAddr.String())
	}

	if strings.Contains(h.EgressHost, ":") {
		return fmt.Errorf("egrsshost can not contain port number: host=%s, egress=%s", h.Name, h.EgressHost)
	}

	return nil
}

// Update updates the host attributes.
func (h *Host) Update(ingresAddress *string, substanceAddress *string, egressHost *string) (err error) {
	ingress := h.IngressAddr.String()
	if ingresAddress != nil {
		ingress = *ingresAddress
	}
	substance := h.SubstanceAddr.String()
	if substanceAddress != nil {
		substance = *substanceAddress
	}
	egress := h.EgressHost
	if egressHost != nil {
		egress = *egressHost
	}

	*h, err = NewHost(h.Name, ingress, substance, egress)
	return err
}

// GetAdminAddr returns envoy's admin endpoint.
func (h *Host) GetAdminAddr() *Address {
	// TODO: get from property
	return &Address{
		Hostname: h.IngressAddr.Hostname,
		Port:     DefaultAdminPort,
	}
}

func validateHostname(s string) error {
	if !rHostName.MatchString(s) {
		return errors.New("hostname must consist of alphanumeric characters, underscores and dashes, and less than 64 characters")
	}
	return nil
}

// FilterHosts filters a slice of Host.
func FilterHosts(hosts []*Host, pred func(*Host) bool) (filtered []*Host) {
	for _, host := range hosts {
		if pred(host) {
			filtered = append(filtered, host)
		}
	}
	return filtered
}

// MapHostsToString transforms a slice of Host to a slice of string.
func MapHostsToString(hosts []*Host, f func(*Host) string) (mapped []string) {
	for _, host := range hosts {
		mapped = append(mapped, f(host))
	}
	return mapped
}
