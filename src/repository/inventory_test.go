package repository

import (
	"testing"

	"github.com/rerorero/meshem/src/model"
	"github.com/rerorero/meshem/src/utils"
	"github.com/stretchr/testify/assert"
)

func TestGetServiceConsul(t *testing.T) {
	consul := utils.NewConsulMock()
	consul.Client.KV().DeleteTree(servicePrefix, nil)
	testGetService(t, NewInventoryConsul(consul))
}

func TestGetServiceHeap(t *testing.T) {
	testGetService(t, NewInventoryHeap())
}

func testGetService(t *testing.T, sut InventoryRepository) {
	dep1 := []model.DependentService{
		model.DependentService{
			Name:       "svc1",
			EgressPort: 9001,
		},
		model.DependentService{
			Name:       "svc2",
			EgressPort: 9002,
		},
	}
	s1 := &model.Service{
		Name:              "service1",
		HostNames:         []string{"host1", "host2"},
		DependentServices: dep1,
		Version:           "abc",
	}
	dep2 := []model.DependentService{
		model.DependentService{
			Name:       "svc3",
			EgressPort: 9003,
		},
		model.DependentService{
			Name:       "svc4",
			EgressPort: 9004,
		},
	}
	s2 := &model.Service{
		Name:              "service2",
		HostNames:         []string{"host3", "host4"},
		DependentServices: dep2,
		Version:           "def",
	}
	all := []model.Service{*s1, *s2}

	// put
	var err error
	for _, svc := range all {
		err = sut.PutService(svc, svc.Version)
		assert.NoError(t, err)
	}

	// by name
	for _, svc := range all {
		actual, ok, err := sut.SelectServiceByName(svc.Name)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, svc, actual)
	}
	// by name (not found)
	_, ok, err := sut.SelectServiceByName("unknown")
	assert.NoError(t, err)
	assert.False(t, ok)

	// select all names
	names, err := sut.SelectAllServiceNames()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{s1.Name, s2.Name}, names)

	// select all
	services, err := sut.SelectAllServices()
	assert.NoError(t, err)
	assert.ElementsMatch(t, all, services)

	// delete one
	ok, err = sut.DeleteService(s1.Name)
	assert.NoError(t, err)
	assert.True(t, ok)
	ok, err = sut.DeleteService(s1.Name)
	assert.NoError(t, err)
	assert.False(t, ok) // already deleted
	// not found
	_, ok, err = sut.SelectServiceByName(s1.Name)
	assert.NoError(t, err)
	assert.False(t, ok)
	services, err = sut.SelectAllServices()
	assert.NoError(t, err)
	assert.Equal(t, []model.Service{*s2}, services)
}

func TestHostServiceConsul(t *testing.T) {
	consul := utils.NewConsulMock()
	consul.Client.KV().DeleteTree(hostPrefix, nil)
	testHostService(t, NewInventoryConsul(consul))
}

func TestHostServiceHeap(t *testing.T) {
	testHostService(t, NewInventoryHeap())
}

func testHostService(t *testing.T, sut InventoryRepository) {
	ip1, _ := model.ParseAddress("1.2.3.4:80")
	ip2, _ := model.ParseAddress("5.6.7.8:8080")
	ip3, _ := model.ParseAddress("9.0.1.2:80")
	ip4, _ := model.ParseAddress("3.4.5.6:8080")

	h1 := &model.Host{
		Name:          "host01",
		IngressAddr:   *ip1,
		EgressHost:    "127.0.0.1",
		SubstanceAddr: *ip2,
	}
	h2 := &model.Host{
		Name:          "host02",
		IngressAddr:   *ip3,
		EgressHost:    "127.0.0.1",
		SubstanceAddr: *ip4,
	}
	all := []model.Host{*h1, *h2}

	// put
	var err error
	for _, host := range all {
		err = sut.PutHost(host)
		assert.NoError(t, err)
	}

	// by name
	for _, host := range all {
		actual, ok, err := sut.SelectHostByName(host.Name)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, host, actual)
	}
	// by name (not found)
	_, ok, err := sut.SelectHostByName("unknown")
	assert.NoError(t, err)
	assert.False(t, ok)

	// select all names
	names, err := sut.SelectAllHostNames()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{h1.Name, h2.Name}, names)

	// select all
	hosts, err := sut.SelectAllHosts()
	assert.NoError(t, err)
	assert.ElementsMatch(t, all, hosts)

	// delete one
	ok, err = sut.DeleteHost(h1.Name)
	assert.NoError(t, err)
	assert.True(t, ok)
	ok, err = sut.DeleteHost(h1.Name)
	assert.NoError(t, err)
	assert.False(t, ok) // already deleted
	// not found
	_, ok, err = sut.SelectHostByName(h1.Name)
	assert.NoError(t, err)
	assert.False(t, ok)
	hosts, err = sut.SelectAllHosts()
	assert.NoError(t, err)
	assert.Equal(t, []model.Host{*h2}, hosts)
}

func TestServiceDependenciesConsul(t *testing.T) {
	consul := utils.NewConsulMock()
	consul.Client.KV().DeleteTree(servicePrefix, nil)
	testServiceDependencies(t, NewInventoryConsul(consul))
}

func TestServiceDependenciesHeap(t *testing.T) {
	testServiceDependencies(t, NewInventoryHeap())
}

func testServiceDependencies(t *testing.T, sut InventoryRepository) {
	svcC := model.Service{
		Name:    "serviceC",
		Version: "abc",
	}

	svcB := model.Service{
		Name:    "serviceB",
		Version: "def",
	}

	svcA := model.Service{
		Name:    "serviceA",
		Version: "ghi",
	}
	allsvc := []*model.Service{&svcA, &svcB, &svcC}

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

	// register without dependencies
	var err error
	for _, svc := range allsvc {
		err = sut.PutService(*svc, svc.Version)
		assert.NoError(t, err)
	}

	// add dependencies
	newVersion := model.Version("xxxx")
	for _, dep := range depA {
		err = sut.AddServiceDependency(svcA.Name, dep, newVersion)
		assert.NoError(t, err)
	}
	for _, dep := range depB {
		err = sut.AddServiceDependency(svcB.Name, dep, newVersion)
		assert.NoError(t, err)
	}
	// version and dependencies are updated.
	svcA.Version = newVersion
	svcA.DependentServices = depA
	svcB.Version = newVersion
	svcB.DependentServices = depB

	// verify dependencies
	for _, svc := range allsvc {
		actual, ok, err := sut.SelectServiceByName(svc.Name)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, *svc, actual)
	}
	dependentA, err := sut.SelectReferringServiceNamesTo(svcA.Name)
	assert.NoError(t, err)
	assert.Empty(t, dependentA)
	dependentB, err := sut.SelectReferringServiceNamesTo(svcB.Name)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{svcA.Name}, dependentB)
	dependentC, err := sut.SelectReferringServiceNamesTo(svcC.Name)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{svcA.Name, svcB.Name}, dependentC)

	// remove dependencies
	newVersion = model.Version("yyyy")
	ok, err := sut.RemoveServiceDependency(svcA.Name, svcB.Name, newVersion)
	assert.NoError(t, err)
	assert.True(t, ok)
	actual, ok, err := sut.SelectServiceByName(svcA.Name)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, []model.DependentService{depA[0]}, actual.DependentServices)

	// error cases
	ok, err = sut.RemoveServiceDependency("unknown", svcA.Name, "zzzz")
	assert.NoError(t, err)
	assert.False(t, ok)
}
