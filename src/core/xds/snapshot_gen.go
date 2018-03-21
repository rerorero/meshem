package xds

import (
	"fmt"
	"time"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	accesslog "github.com/envoyproxy/go-control-plane/envoy/config/filter/accesslog/v2"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	tcp "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/tcp_proxy/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/envoyproxy/go-control-plane/pkg/util"
	"github.com/pkg/errors"
	mcore "github.com/rerorero/meshem/src/core"
	"github.com/rerorero/meshem/src/model"
	"github.com/sirupsen/logrus"
)

type SnapshotGen interface {
	MakeSnapshotsOfService(serviceName string) (snapshots map[*model.Host]*cache.Snapshot, err error)
}

type snapGen struct {
	inventory  mcore.InventoryService
	logger     *logrus.Logger
	versionGen mcore.VersionGenerator
	envoyConf  model.EnvoyConf
}

const (
	// XdsCluster is the cluster name for the control server (used by non-ADS set-up)
	XdsCluster = "xds_cluster"
)

// NewSnapshotGen creates snapshot generator instance.
func NewSnapshotGen(is mcore.InventoryService, logger *logrus.Logger, vg mcore.VersionGenerator, envoyConf model.EnvoyConf) SnapshotGen {
	return &snapGen{
		inventory:  is,
		logger:     logger,
		versionGen: vg,
		envoyConf:  envoyConf,
	}
}

// FindSnapshotByName finds a snapshot by hostname from a snapshot map
func FindSnapshotByName(snapshots map[*model.Host]*cache.Snapshot, hostname string) (*cache.Snapshot, bool) {
	for h, s := range snapshots {
		if h.Name == hostname {
			return s, true
		}
	}
	return nil, false
}

func (gen *snapGen) MakeSnapshotsOfService(serviceName string) (map[*model.Host]*cache.Snapshot, error) {
	// get service
	service, ok, err := gen.inventory.GetService(serviceName)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("service=%s not found", serviceName)
	}

	// get service dependencies
	dependencies := map[*model.Service][]model.Host{}
	var i int
	for i = 0; i < len(service.DependentServices); i++ {
		name := service.DependentServices[i].Name
		depHosts, err := gen.inventory.GetHostsOfService(name)
		if err != nil {
			return nil, err
		}
		depService, ok, err := gen.inventory.GetService(name)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("depndencies of %s not found", name)
		}
		dependencies[&depService] = depHosts
	}

	hosts, err := gen.inventory.GetHostsOfService(service.Name)
	if err != nil {
		return nil, err
	}

	snapshots := map[*model.Host]*cache.Snapshot{}
	for i = 0; i < len(hosts); i++ {
		snapshot, err := gen.makeHostSnapshot(&service, &hosts[i], dependencies)
		if err != nil {
			return nil, errors.Wrapf(err, "udpate snapshot failed: service=%+v, host=%+v", service, hosts[i])
		}
		err = snapshot.Consistent()
		if err != nil {
			return nil, errors.Wrapf(err, "snapshot incosistency: %+v", snapshot)
		}
		snapshots[&hosts[i]] = snapshot
	}
	return snapshots, nil
}

func (gen *snapGen) makeHostSnapshot(service *model.Service, host *model.Host, dependencies map[*model.Service][]model.Host) (*cache.Snapshot, error) {
	clusters := []cache.Resource{}
	endpoints := []cache.Resource{}
	routes := []cache.Resource{}
	listeners := []cache.Resource{}
	defaultTimeout := time.Duration(gen.envoyConf.ClusterTimeoutMS) * time.Millisecond

	// version of the data to be cached
	version := gen.latestNodeVersion(service, dependencies)

	// ingress
	ingressClusterName := "ingress"
	clusters = append(clusters, MakeEDSCluster(ingressClusterName, defaultTimeout))
	endpoints = append(endpoints, MakeEndpoint(ingressClusterName, []model.Address{host.SubstanceAddr}))

	listenerName := fmt.Sprintf("listener-%s-%s", ingressClusterName, host.IngressAddr.ListenerSuffix())
	switch service.Protocol {
	case model.ProtocolHTTP:
		ingressRouteName := "route-" + ingressClusterName
		routes = append(routes, MakeRoute(ingressRouteName, ingressClusterName, service.TraceSpan))
		l, err := MakeHTTPListener(&httpListenerParam{
			listenerName: listenerName,
			address:      &host.IngressAddr,
			route:        ingressRouteName,
			statPrefix:   ingressClusterName,
			logfileDir:   gen.envoyConf.AccessLogDir,
			logfileName:  ingressClusterName + ".log",
			health:       NewDisabledHTTPHealthCheck(),
			isIngress:    true,
			traceEnabled: len(service.TraceSpan) > 0,
		})
		if err != nil {
			return nil, err
		}
		listeners = append(listeners, l)
	case model.ProtocolTCP:
		l, err := MakeTCPListener(listenerName, host.IngressAddr, ingressClusterName, ingressClusterName, gen.envoyConf.AccessLogDir, ingressClusterName+".log")
		if err != nil {
			return nil, err
		}
		listeners = append(listeners, l)
	default:
		return nil, fmt.Errorf("%s provides unsupported protocol=%s", service.Name, service.Protocol)
	}

	// egress(dependent services)
	for depsvc, dephosts := range dependencies {
		// integrity checking
		ok, ref := service.FindDependentServiceName(depsvc.Name)
		if !ok {
			return nil, fmt.Errorf("service and its dependencies state are not matched, %s is not found in dependencies", depsvc.Name)
		}

		egressClusterName := "egress-" + depsvc.Name
		clusters = append(clusters, MakeEDSCluster(egressClusterName, defaultTimeout))

		depAddresses := make([]model.Address, len(dephosts))
		var i int
		for i = 0; i < len(dephosts); i++ {
			depAddresses[i] = dephosts[i].IngressAddr
		}
		endpoints = append(endpoints, MakeEndpoint(egressClusterName, depAddresses))

		addrstr := fmt.Sprintf("%s:%d", host.EgressHost, ref.EgressPort)
		addr, err := model.ParseAddress(addrstr)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid address of the egress endpoint, svc=%s, host=%s, depend=%s, addr=%s", service.Name, host.Name, depsvc.Name, addrstr)
		}

		listenerName := fmt.Sprintf("listener-%s-%s", egressClusterName, addr.ListenerSuffix())
		switch depsvc.Protocol {
		case model.ProtocolHTTP:
			egressRouteName := "route-" + egressClusterName
			routes = append(routes, MakeRoute(egressRouteName, egressClusterName, depsvc.TraceSpan))
			l, err := MakeHTTPListener(&httpListenerParam{
				listenerName: listenerName,
				address:      addr,
				route:        egressRouteName,
				statPrefix:   egressClusterName,
				logfileDir:   gen.envoyConf.AccessLogDir,
				logfileName:  egressClusterName + ".log",
				health:       NewDisabledHTTPHealthCheck(),
				isIngress:    false,
				traceEnabled: len(depsvc.TraceSpan) > 0,
			})
			if err != nil {
				return nil, err
			}
			listeners = append(listeners, l)
		case model.ProtocolTCP:
			l, err := MakeTCPListener(listenerName, *addr, egressClusterName, egressClusterName, gen.envoyConf.AccessLogDir, egressClusterName+".log")
			if err != nil {
				return nil, err
			}
			listeners = append(listeners, l)
		default:
			return nil, fmt.Errorf("%s provides unsupported protocol=%s", service.Name, service.Protocol)
		}
	}

	snapshot := cache.NewSnapshot(string(version), endpoints, clusters, routes, listeners)
	return &snapshot, nil
}

// latestNodeVersion determines the version of the cache data of the node. It selects the latest from the all related service version.
func (gen *snapGen) latestNodeVersion(service *model.Service, depndencies map[*model.Service][]model.Host) model.Version {
	latest := service.Version
	for dep := range depndencies {
		if gen.versionGen.Compare(latest, dep.Version) > 0 {
			latest = dep.Version
		}
	}
	return latest
}

// MakeEDSCluster creates a EDS cluster.
func MakeEDSCluster(clusterName string, timeout time.Duration) *v2.Cluster {
	edsSource := &core.ConfigSource{
		ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
			ApiConfigSource: &core.ApiConfigSource{
				ApiType:      core.ApiConfigSource_GRPC,
				ClusterNames: []string{XdsCluster},
			},
		},
	}

	return &v2.Cluster{
		Name:           clusterName,
		ConnectTimeout: timeout,
		Type:           v2.Cluster_EDS,
		EdsClusterConfig: &v2.Cluster_EdsClusterConfig{
			EdsConfig: edsSource,
		},
	}
}

// MakeEndpoint creates a endpoint on a given address.
func MakeEndpoint(clusterName string, addresses []model.Address) *v2.ClusterLoadAssignment {
	endpoints := make([]endpoint.LbEndpoint, len(addresses))
	var i int
	for i = 0; i < len(addresses); i++ {
		endpoints[i] = endpoint.LbEndpoint{
			Endpoint: &endpoint.Endpoint{
				Address: &core.Address{
					Address: &core.Address_SocketAddress{
						SocketAddress: &core.SocketAddress{
							Protocol: core.TCP,
							Address:  addresses[i].Hostname,
							PortSpecifier: &core.SocketAddress_PortValue{
								PortValue: addresses[i].Port,
							},
						},
					},
				},
			},
		}
	}

	return &v2.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []endpoint.LocalityLbEndpoints{{
			LbEndpoints: endpoints,
		}},
	}
}

// MakeRoute creates an HTTP route that routes to a given cluster.
func MakeRoute(routeName, clusterName string, traceSpan string) *v2.RouteConfiguration {
	var decorater *route.Decorator
	if len(traceSpan) > 0 {
		decorater = &route.Decorator{Operation: traceSpan}
	}

	return &v2.RouteConfiguration{
		Name: routeName,
		VirtualHosts: []route.VirtualHost{{
			Name:    routeName,
			Domains: []string{"*"},
			Routes: []route.Route{{
				Match: route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: clusterName,
						},
					},
				},
				Decorator: decorater,
			}},
		}},
	}
}

type httpListenerParam struct {
	listenerName string
	address      *model.Address
	route        string
	statPrefix   string
	logfileDir   string
	logfileName  string
	health       *HTTPHealthCheck
	isIngress    bool
	traceEnabled bool
	// TODO: more trace settings
}

// MakeHTTPListener creates a listener using either ADS or RDS for the route.
func MakeHTTPListener(p *httpListenerParam) (*v2.Listener, error) {
	// access log service configuration
	alsConfig := &accesslog.FileAccessLog{
		Path: p.logfileDir + "/" + p.logfileName,
	}
	alsConfigPbst, err := util.MessageToStruct(alsConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "listnere FileAccessLog generation failed(%+v)", *p)
	}

	// HTTP filter configuration
	httpFilters := []*hcm.HttpFilter{{
		Name: cache.Router,
	}}
	if p.health.Enabled {
		filter, err := p.health.createEnvoyHTTPFilter()
		if err != nil {
			return nil, err
		}
		httpFilters = append(httpFilters, filter)
	}

	// tracing
	var tracing *hcm.HttpConnectionManager_Tracing
	if p.traceEnabled {
		tracing = &hcm.HttpConnectionManager_Tracing{}
		if p.isIngress {
			tracing.OperationName = hcm.INGRESS
		} else {
			tracing.OperationName = hcm.EGRESS
		}
	}

	// HTTP connection manager configuration
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.AUTO,
		StatPrefix: p.statPrefix,
		Tracing:    tracing,
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource: core.ConfigSource{
					ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
						ApiConfigSource: &core.ApiConfigSource{
							ApiType:      core.ApiConfigSource_GRPC,
							ClusterNames: []string{XdsCluster},
						},
					},
				},
				RouteConfigName: p.route,
			},
		},
		HttpFilters: httpFilters,
		AccessLog: []*accesslog.AccessLog{{
			Name:   "envoy.file_access_log",
			Config: alsConfigPbst,
		}},
	}

	pbst, err := util.MessageToStruct(manager)
	if err != nil {
		return nil, errors.Wrapf(err, "listnere Manager generation failed(%+v)", *p)
	}

	return &v2.Listener{
		Name: p.listenerName,
		Address: core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.TCP,
					Address:  p.address.Hostname,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: p.address.Port,
					},
				},
			},
		},
		FilterChains: []listener.FilterChain{{
			Filters: []listener.Filter{{
				Name:   cache.HTTPConnectionManager,
				Config: pbst,
			}},
		}},
	}, nil
}

// MakeTCPListener creates a TCP listener for a cluster.
func MakeTCPListener(listenerName string, address model.Address, clusterName string, statPrefix string, logfileDir string, logfileName string) (*v2.Listener, error) {
	// access log service configuration
	alsConfig := &accesslog.FileAccessLog{
		Path: logfileDir + "/" + logfileName,
	}
	alsConfigPbst, err := util.MessageToStruct(alsConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "listnere FileAccessLog generation failed(name=%s, cluste=%s, addr=%+v, log=%s:%s)", listenerName, address, clusterName, logfileDir, logfileName)
	}
	// TCP filter configuration
	config := &tcp.TcpProxy{
		StatPrefix: statPrefix,
		Cluster:    clusterName,
		AccessLog: []*accesslog.AccessLog{{
			Name:   "envoy.file_access_log",
			Config: alsConfigPbst,
		}},
	}
	pbst, err := util.MessageToStruct(config)
	if err != nil {
		return nil, errors.Wrapf(err, "tcp proxy generation failed(name=%s, addr=%+v, cluster=%s)", listenerName, address, clusterName)
	}
	return &v2.Listener{
		Name: listenerName,
		Address: core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.TCP,
					Address:  address.Hostname,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: address.Port,
					},
				},
			},
		},
		FilterChains: []listener.FilterChain{{
			Filters: []listener.Filter{{
				Name:   cache.TCPProxy,
				Config: pbst,
			}},
		}},
	}, nil
}
