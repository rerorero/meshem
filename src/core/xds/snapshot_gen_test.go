package xds

import (
	"fmt"
	"testing"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	mcore "github.com/rerorero/meshem/src/core"
	"github.com/rerorero/meshem/src/model"
	"github.com/rerorero/meshem/src/repository"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func addr2str(addr *core.Address) string {
	return fmt.Sprintf("%s:%d", addr.GetSocketAddress().GetAddress(), addr.GetSocketAddress().GetPortValue())
}

func TestMakeSnapshot(t *testing.T) {
	repo := repository.NewInventoryHeap()
	gen := mcore.NewCurrentTimeGenerator()
	inventory := mcore.NewInventoryService(repo, nil, gen, logrus.New())
	conf := model.EnvoyConf{
		ClusterTimeoutMS: 2000,
		AccessLogDir:     "/var/log/test",
	}
	sut := NewSnapshotGen(inventory, logrus.New(), gen, conf)

	// register service
	svcA := &model.Service{
		Name:     "serviceA",
		Version:  "1",
		Protocol: model.ProtocolTCP,
	}
	svcB := &model.Service{
		Name:     "serviceB",
		Version:  "1",
		Protocol: model.ProtocolHTTP,
	}
	allsvc := []*model.Service{svcA, svcB}

	var firstVer model.Version
	for _, svc := range allsvc {
		s, err := inventory.RegisterService(svc.Name, svc.Protocol)
		assert.NoError(t, err)
		firstVer = s.Version
	}
	a2bport := uint32(10001)
	assert.NoError(t, inventory.AddServiceDependency(svcA.Name, svcB.Name, a2bport))

	// register host
	svcA1, err := inventory.RegisterHost(svcA.Name, "svcA1", "192.168.0.1:80", "127.0.0.1:8001", "127.0.0.1")
	assert.NoError(t, err)
	_, err = inventory.RegisterHost(svcB.Name, "svcB1", "192.168.1.1:8080", "127.0.0.1:9001", "127.0.0.1")
	assert.NoError(t, err)
	_, err = inventory.RegisterHost(svcB.Name, "svcB2", "192.168.1.2:8080", "127.0.0.1:9002", "127.0.0.1")
	assert.NoError(t, err)

	// make snapshot
	shotA, err := sut.MakeSnapshotsOfService(svcA.Name)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(shotA))
	actualA1, ok := FindSnapshotByName(shotA, svcA1.Name)
	assert.True(t, ok)
	// listeneres
	assert.NotEqual(t, actualA1.Listeners.Version, firstVer)
	assert.Equal(t, 2, len(actualA1.Listeners.Items))
	for _, item := range actualA1.Listeners.Items {
		listnere := item.(*v2.Listener)
		switch listnere.Name {
		case "listener-ingress-19216801-80":
			assert.Equal(t, "192.168.0.1:80", addr2str(&listnere.Address))
		case "listener-egress-serviceB-127001-10001":
			assert.Equal(t, "127.0.0.1:10001", addr2str(&listnere.Address))
		default:
			assert.Failf(t, "unknown listenere name: %s", listnere.Name)
		}
	}
	// clusters
	assert.Equal(t, 2, len(actualA1.Clusters.Items))
	// routes
	assert.Equal(t, 1, len(actualA1.Routes.Items))
	// endpoints
	assert.Equal(t, 2, len(actualA1.Endpoints.Items))
	ingressCluster := []*v2.ClusterLoadAssignment{}
	egressBCluster := []*v2.ClusterLoadAssignment{}
	for _, item := range actualA1.Endpoints.Items {
		i := item.(*v2.ClusterLoadAssignment)
		switch i.ClusterName {
		case "ingress":
			ingressCluster = append(ingressCluster, i)
		case "egress-serviceB":
			egressBCluster = append(egressBCluster, i)
		default:
			assert.Failf(t, "invalid cluster: %s", i.ClusterName)
		}
	}
	// ingress
	assert.Equal(t, 1, len(ingressCluster))
	assert.Equal(t, "127.0.0.1:8001", addr2str(ingressCluster[0].Endpoints[0].LbEndpoints[0].Endpoint.Address))
	// egress
	assert.Equal(t, 1, len(egressBCluster))
	assert.Equal(t, 2, len(egressBCluster[0].Endpoints[0].LbEndpoints))
	egressBAddress := []string{}
	for _, e := range egressBCluster[0].Endpoints[0].LbEndpoints {
		egressBAddress = append(egressBAddress, addr2str(e.Endpoint.Address))
	}
	assert.ElementsMatch(t, []string{"192.168.1.1:8080", "192.168.1.2:8080"}, egressBAddress)
}
