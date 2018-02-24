package core

import (
	"testing"

	"github.com/rerorero/meshem/src/model"
	"github.com/rerorero/meshem/src/repository"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockedDiscoveryRepository struct {
	mock.Mock
}

func (mdr *MockedDiscoveryRepository) Register(host model.Host, tags map[string]string) error {
	args := mdr.Called(host, tags)
	return args.Error(0)
}
func (mdr *MockedDiscoveryRepository) Unregister(hostname string) error {
	args := mdr.Called(hostname)
	return args.Error(0)
}
func (mdr *MockedDiscoveryRepository) FindByName(hostname string) (*repository.DiscoveryInfo, bool, error) {
	args := mdr.Called(hostname)
	return args.Get(0).(*repository.DiscoveryInfo), args.Bool(1), args.Error(2)
}

func TestRegisterService(t *testing.T) {
	repo := repository.NewInventoryHeap()
	discovery := MockedDiscoveryRepository{}
	gen := &MockedVersionGen{Version: "abc"}
	sut := NewInventoryService(repo, &discovery, gen, logrus.New())

	svc, err := sut.RegisterService("svc1", model.ProtocolHTTP)
	assert.NoError(t, err)

	actualsvc, ok, err := sut.GetService("svc1")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, actualsvc, model.Service{Name: "svc1", Protocol: model.ProtocolHTTP, Version: gen.Version, TraceSpan: "svc1"})
	actual, err := repo.SelectAllServiceNames()
	assert.NoError(t, err)
	assert.ElementsMatch(t, actual, []string{svc.Name})

	// duplicates
	_, err = sut.RegisterService("svc1", model.ProtocolTCP)
	assert.Error(t, err)
	// validation failed
	_, err = sut.RegisterService(" in va lid", model.ProtocolHTTP)
	assert.Error(t, err)

	svc2, err := sut.RegisterService("svc2", model.ProtocolHTTP)
	assert.NoError(t, err)

	svcs, err := repo.SelectAllServices()
	assert.NoError(t, err)
	assert.ElementsMatch(t, svcs, []model.Service{svc, svc2})

	ok, _, err = sut.UnregisterService("svc3")
	assert.NoError(t, err)
	assert.False(t, ok)
	actual, err = repo.SelectAllServiceNames()
	assert.NoError(t, err)
	assert.ElementsMatch(t, actual, []string{svc.Name, svc2.Name})

	ok, _, err = sut.UnregisterService("svc1")
	assert.NoError(t, err)
	assert.True(t, ok)
	actual, err = repo.SelectAllServiceNames()
	assert.NoError(t, err)
	assert.ElementsMatch(t, actual, []string{svc2.Name})
}

func TestRegisterHost(t *testing.T) {
	repo := repository.NewInventoryHeap()
	discovery := MockedDiscoveryRepository{}
	gen := &MockedVersionGen{Version: "abc"}
	sut := NewInventoryService(repo, &discovery, gen, logrus.New())

	svc, err := sut.RegisterService("svc1", model.ProtocolHTTP)
	assert.NoError(t, err)
	gen.Version = "def"

	expect, err := model.NewHost("host1", "192.168.0.1:8081", "127.0.0.1:8080", "127.0.0.1")
	expectTags := map[string]string{}
	expectTags["meshem_service"] = "svc1"
	assert.NoError(t, err)
	discovery.On("Register", expect, expectTags).Return(nil)
	host1, err := sut.RegisterHost(svc.Name, "host1", "192.168.0.1:8081", "127.0.0.1:8080", "127.0.0.1")
	assert.NoError(t, err)
	assert.Equal(t, expect, host1)

	actualHost, ok, err := sut.GetHostByName("host1")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, actualHost, host1)
	// registered host in service
	actualSvc, ok, err := sut.GetService(svc.Name)
	assert.NoError(t, err)
	assert.True(t, ok)
	// svc = expected
	svc.Version = "def"
	svc.HostNames = []string{"host1"}
	assert.Equal(t, svc, actualSvc)

	// unregister
	discovery.On("Unregister", "host1").Return(nil)
	deleted, err := sut.UnregisterHost("svc1", "host1")
	assert.NoError(t, err)
	assert.True(t, deleted)
}

func TestServiceDependencies(t *testing.T) {
	repo := repository.NewInventoryHeap()
	discovery := MockedDiscoveryRepository{}
	gen := &MockedVersionGen{Version: "abc"}
	sut := NewInventoryService(repo, &discovery, gen, logrus.New())

	svcC := &model.Service{
		Name:     "serviceC",
		Version:  "abc",
		Protocol: model.ProtocolHTTP,
	}

	svcB := &model.Service{
		Name:     "serviceB",
		Version:  "def",
		Protocol: model.ProtocolTCP,
	}

	svcA := &model.Service{
		Name:     "serviceA",
		Version:  "ghi",
		Protocol: model.ProtocolTCP,
	}
	allsvc := []*model.Service{svcA, svcB, svcC}

	depB := []model.DependentService{
		model.DependentService{
			Name:       svcC.Name,
			EgressPort: 9001,
		},
	}
	depA := []model.DependentService{
		model.DependentService{
			Name:       svcC.Name,
			EgressPort: 9001,
		},
		model.DependentService{
			Name:       svcB.Name,
			EgressPort: 9002,
		},
	}

	var err error
	// register services without dependencies
	for _, svc := range allsvc {
		_, err := sut.RegisterService(svc.Name, svc.Protocol)
		assert.NoError(t, err)
	}
	names, err := sut.GetServiceNames()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{svcA.Name, svcB.Name, svcC.Name}, names)

	// add dependencies
	gen.Version = "newnew"
	for _, dep := range depA {
		err = sut.AddServiceDependency(svcA.Name, dep.Name, dep.EgressPort)
		assert.NoError(t, err)
	}
	for _, dep := range depB {
		err = sut.AddServiceDependency(svcB.Name, dep.Name, dep.EgressPort)
		assert.NoError(t, err)
	}
	actualsvc, ok, err := sut.GetService(svcA.Name)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, actualsvc.Version, model.Version("newnew"))
	assert.ElementsMatch(t, actualsvc.DependentServices, depA)

	// remove service
	refs, err := sut.GetRefferersOf(svcC.Name)
	assert.NoError(t, err)
	assert.ElementsMatch(t, refs, []string{svcA.Name, svcB.Name})

	ok, refs, err = sut.UnregisterService(svcB.Name)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.ElementsMatch(t, refs, []string{svcA.Name})

	refs, err = sut.GetRefferersOf(svcC.Name)
	assert.NoError(t, err)
	assert.ElementsMatch(t, refs, []string{svcA.Name})
}

func TestIdemopotentService(t *testing.T) {
	repo := repository.NewInventoryHeap()
	gen := &MockedVersionGen{Version: "abc"}
	sut := NewInventoryService(repo, nil, gen, logrus.New())

	svcA := model.IdempotentServiceParam{
		Protocol: "HTTP",
		Hosts: []model.Host{
			{
				Name:          "a-1",
				IngressAddr:   model.Address{Hostname: "192.168.0.1", Port: 8000},
				SubstanceAddr: model.Address{Hostname: "127.0.0.1", Port: 8001},
				EgressHost:    "127.0.0.1",
			},
			{
				Name:          "a-2",
				IngressAddr:   model.Address{Hostname: "192.168.0.2", Port: 8000},
				SubstanceAddr: model.Address{Hostname: "127.0.0.1", Port: 8001},
				EgressHost:    "127.0.0.1",
			},
		},
	}
	svcB := model.IdempotentServiceParam{
		Protocol: "TCP",
		DependentServices: []model.DependentService{
			{
				Name:       "svcA",
				EgressPort: 9002,
			},
		},
		Hosts: []model.Host{
			{
				Name:          "b-1",
				IngressAddr:   model.Address{Hostname: "192.168.0.2", Port: 9000},
				SubstanceAddr: model.Address{Hostname: "127.0.0.1", Port: 9001},
				EgressHost:    "127.0.0.1",
			},
		},
	}

	// put A
	changed, err := sut.IdempotentService("svcA", svcA)
	assert.NoError(t, err)
	assert.True(t, changed)
	changed, err = sut.IdempotentService("svcA", svcA)
	assert.NoError(t, err)
	assert.False(t, changed) // not changed when applied twice

	actualsvc, ok, err := sut.GetService("svcA")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, model.Service{
		Name:              "svcA",
		HostNames:         []string{"a-1", "a-2"},
		DependentServices: nil,
		Protocol:          svcA.Protocol,
		Version:           "abc",
		TraceSpan:         "svcA",
	}, actualsvc)
	actualHosts, err := sut.GetHostsOfService("svcA")
	assert.ElementsMatch(t, svcA.Hosts, actualHosts)

	// put B
	gen.Version = "bbb"
	changed, err = sut.IdempotentService("svcB", svcB)
	assert.NoError(t, err)
	assert.True(t, changed)

	actualsvc, ok, err = sut.GetService("svcB")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, model.Service{
		Name:              "svcB",
		HostNames:         []string{"b-1"},
		DependentServices: svcB.DependentServices,
		Protocol:          svcB.Protocol,
		Version:           "bbb",
		TraceSpan:         "svcB",
	}, actualsvc)
	actualHosts, err = sut.GetHostsOfService("svcB")
	assert.ElementsMatch(t, svcB.Hosts, actualHosts)
	// svcA's version should not be updated
	actualsvc, ok, err = sut.GetService("svcA")
	assert.NoError(t, err)
	assert.Equal(t, model.Version("abc"), actualsvc.Version)

	// update svcA
	gen.Version = "ccc"
	svcAMod := model.IdempotentServiceParam{
		Protocol: "TCP",
		Hosts: []model.Host{
			{
				Name:          "a-2",
				IngressAddr:   model.Address{Hostname: "192.168.99.2", Port: 9000},
				SubstanceAddr: model.Address{Hostname: "localhost", Port: 9001},
				EgressHost:    "localhost",
			},
			{
				Name:          "a-3",
				IngressAddr:   model.Address{Hostname: "192.168.0.3", Port: 9000},
				SubstanceAddr: model.Address{Hostname: "127.0.0.1", Port: 9001},
				EgressHost:    "127.0.0.1",
			},
		},
	}
	changed, err = sut.IdempotentService("svcA", svcAMod)
	assert.NoError(t, err)
	assert.True(t, changed)

	actualsvc, ok, err = sut.GetService("svcA")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, model.Service{
		Name:              "svcA",
		HostNames:         []string{"a-2", "a-3"},
		DependentServices: nil,
		Protocol:          svcAMod.Protocol,
		Version:           "ccc",
		TraceSpan:         "svcA",
	}, actualsvc)
	actualHosts, err = sut.GetHostsOfService("svcA")
	assert.ElementsMatch(t, svcAMod.Hosts, actualHosts)

	// update svcB
	gen.Version = "ddd"
	svcC := model.IdempotentServiceParam{Protocol: "HTTP"}
	changed, err = sut.IdempotentService("svcC", svcC)
	assert.NoError(t, err)
	assert.True(t, changed)
	svcBMod := model.IdempotentServiceParam{
		Protocol: "TCP",
		DependentServices: []model.DependentService{
			{
				Name:       "svcA",
				EgressPort: 9003,
			},
			{
				Name:       "svcC",
				EgressPort: 9004,
			},
		},
		Hosts: []model.Host{
			{
				Name:          "b-1",
				IngressAddr:   model.Address{Hostname: "192.168.0.2", Port: 9000},
				SubstanceAddr: model.Address{Hostname: "127.0.0.1", Port: 9001},
				EgressHost:    "127.0.0.1",
			},
		},
	}
	gen.Version = "eee"
	changed, err = sut.IdempotentService("svcB", svcBMod)
	assert.NoError(t, err)
	assert.True(t, changed)

	actualsvc, ok, err = sut.GetService("svcB")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, model.Service{
		Name:              "svcB",
		HostNames:         []string{"b-1"},
		DependentServices: svcBMod.DependentServices,
		Protocol:          svcBMod.Protocol,
		Version:           "eee",
		TraceSpan:         "svcB",
	}, actualsvc)
	actualHosts, err = sut.GetHostsOfService("svcB")
	assert.ElementsMatch(t, svcBMod.Hosts, actualHosts)
}
