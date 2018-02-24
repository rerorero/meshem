package model

import (
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// EnvoyConf relates to envoy and xds.
type EnvoyConf struct {
	ClusterTimeoutMS int    `yaml:"cluster_timeout_ms,omitempty"`
	AccessLogDir     string `yaml:"access_log_dir"`
}

// XDSConf relates to xds function.
type XDSConf struct {
	Port                      uint32 `yaml:"port,omitempty"`
	CacheCollectionIntervalMS int    `yaml:"cache_collection_interval_ms,omitempty"`
	IsADSMode                 bool   `yaml:"ads_mode,omitempty"`
}

// ConsulConf relates to consul.
type ConsulConf struct {
	URL        string `yaml:"url"`
	Token      string `yaml:"token"`
	Datacenter string `yaml:"datacenter,omitempty"`
}

// CtlAPIConf relates to meshem control api.
type CtlAPIConf struct {
	Port uint32 `yaml:"port"`
}

// DiscoveryConf relates to discovery service. This is optional
type DiscoveryConf struct {
	Type   string      `yaml:"type"`
	Consul *ConsulConf `yaml:"consul,omitempty"`
}

// MeshemConf is configurations for conductor server.
type MeshemConf struct {
	Envoy     EnvoyConf      `yaml:"envoy"`
	XDS       XDSConf        `yaml:"xds"`
	Consul    ConsulConf     `yaml:"consul"`
	CtlAPI    CtlAPIConf     `yaml:"ctlapi"`
	Discovery *DiscoveryConf `yaml:"discovery"`
}

const (
	// DefaultXDSPort is default port for XDS.
	DefaultXDSPort = 8090
	// DefaultCtrlAPIPort is default port for the control API.
	DefaultCtrlAPIPort = 8091
	// DiscoveryTypeConsul is set to use consul discovery service
	DiscoveryTypeConsul = "consul"
)

// NewMeshemConfFile parses configuration file.
func NewMeshemConfFile(confPath string) (*MeshemConf, error) {
	buf, err := ioutil.ReadFile(confPath)
	if err != nil {
		return nil, fmt.Errorf("could not read config - %s", err)
	}

	return NewMeshemConfYaml(buf)
}

// NewMeshemConfYaml parses configuration YAML string.
func NewMeshemConfYaml(yamlbytes []byte) (*MeshemConf, error) {
	conf := &MeshemConf{}
	err := yaml.Unmarshal(yamlbytes, conf)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Fatal: Could not parse config(%s)", string(yamlbytes)))
	}

	// default values
	if conf.Envoy.ClusterTimeoutMS == 0 {
		conf.Envoy.ClusterTimeoutMS = 5000
	}
	if len(conf.Envoy.AccessLogDir) == 0 {
		conf.Envoy.AccessLogDir = "/var/log/envoy"
	}
	if conf.XDS.Port == 0 {
		conf.XDS.Port = DefaultXDSPort
	}
	if conf.XDS.CacheCollectionIntervalMS == 0 {
		conf.XDS.CacheCollectionIntervalMS = 10000
	}
	if len(conf.Consul.Datacenter) == 0 {
		conf.Consul.Datacenter = "dc1"
	}
	if conf.CtlAPI.Port == 0 {
		conf.CtlAPI.Port = DefaultCtrlAPIPort
	}

	// validation
	if conf.Discovery != nil {
		switch conf.Discovery.Type {
		case DiscoveryTypeConsul:
			if conf.Discovery.Consul == nil {
				return nil, fmt.Errorf("discovery.consul should be set when consul discovery is enabled")
			}
			if len(conf.Discovery.Consul.Datacenter) == 0 {
				conf.Discovery.Consul.Datacenter = "dc1"
			}
		default:
			return nil, fmt.Errorf("invalid discovery type: %s", conf.Discovery.Type)
		}
	}

	return conf, nil
}
