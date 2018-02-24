package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hashicorp/consul/api"
	"github.com/rerorero/meshem/src/model"
	"github.com/rerorero/meshem/src/utils"
)

func unregisterAll(t *testing.T, consul *utils.Consul) {
	nodes, _, err := consul.Client.Catalog().Nodes(nil)
	assert.NoError(t, err)
	for _, n := range nodes {
		dereg := &api.CatalogDeregistration{
			Node: n.Node,
		}
		_, err = consul.Client.Catalog().Deregister(dereg, nil)
		assert.NoError(t, err)
	}
}

func TestDiscoveryRegister(t *testing.T) {
	consul := utils.NewConsulMock()
	unregisterAll(t, consul)
	sut := NewDiscoveryConsul(consul, "")

	host, err := model.NewHost("reg1", "192.168.10.10:80", "127.0.0.1:8080", "127.0.0.1")
	assert.NoError(t, err)
	tags := map[string]string{}
	tags["aaa"] = "a3"
	tags["bbbb"] = "b4"

	err = sut.Register(host, tags)
	assert.NoError(t, err)

	info, ok, err := sut.FindByName("reg1")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, &DiscoveryInfo{
		Name:    host.Name,
		Address: *host.GetAdminAddr(),
		Tags:    tags,
	}, info)

	// overwrite
	host2, err := model.NewHost("reg1", "192.168.20.30:9000", "127.0.0.1:9090", "127.0.0.1")
	tags["c"] = "c1"
	err = sut.Register(host2, tags)
	assert.NoError(t, err)

	info, ok, err = sut.FindByName("reg1")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, &DiscoveryInfo{
		Name:    host2.Name,
		Address: *host2.GetAdminAddr(),
		Tags:    tags,
	}, info)

	// unregister
	err = sut.Unregister("reg1")
	assert.NoError(t, err)

	// not found
	info, ok, err = sut.FindByName("reg1")
	assert.NoError(t, err)
	assert.False(t, ok)

	// unregister unregistered
	err = sut.Unregister("reg1")
	assert.NoError(t, err)
}
