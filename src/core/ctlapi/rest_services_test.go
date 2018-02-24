package ctlapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rerorero/meshem/src/model"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockedInventory struct {
	mock.Mock
}

func (i *MockedInventory) RegisterService(name string, protocol string) (model.Service, error) {
	args := i.Called(name, protocol)
	return args.Get(0).(model.Service), args.Error(1)
}
func (i *MockedInventory) UnregisterService(name string) (deleted bool, referer []string, err error) {
	args := i.Called(name)
	return args.Bool(0), args.Get(1).([]string), args.Error(2)
}
func (i *MockedInventory) GetService(name string) (model.Service, bool, error) {
	args := i.Called(name)
	return args.Get(0).(model.Service), args.Bool(1), args.Error(2)
}
func (i *MockedInventory) GetServiceNames() ([]string, error) {
	args := i.Called()
	return args.Get(0).([]string), args.Error(1)
}
func (i *MockedInventory) AddServiceDependency(serviceName string, dependServiceNames string, egressPort uint32) error {
	args := i.Called(serviceName, dependServiceNames, egressPort)
	return args.Error(0)
}
func (i *MockedInventory) RemoveServiceDependency(serviceName string, dependServiceNames string) (bool, error) {
	args := i.Called(serviceName, dependServiceNames)
	return args.Bool(0), args.Error(1)
}
func (i *MockedInventory) GetRefferersOf(serviceName string) ([]string, error) {
	args := i.Called(serviceName)
	return args.Get(0).([]string), args.Error(1)
}
func (i *MockedInventory) RegisterHost(serviceName, hostName, ingressAddr, substanceAddr, egressHost string) (model.Host, error) {
	args := i.Called(serviceName, hostName, ingressAddr, substanceAddr, egressHost)
	return args.Get(0).(model.Host), args.Error(1)
}
func (i *MockedInventory) UnregisterHost(serviceName string, hostName string) (bool, error) {
	args := i.Called(serviceName, hostName)
	return args.Bool(0), args.Error(1)
}
func (i *MockedInventory) GetHostByName(name string) (model.Host, bool, error) {
	args := i.Called(name)
	return args.Get(0).(model.Host), args.Bool(1), args.Error(2)
}
func (i *MockedInventory) GetHostNames() ([]string, error) {
	args := i.Called()
	return args.Get(0).([]string), args.Error(1)
}
func (i *MockedInventory) GetHostsOfService(serviceName string) ([]model.Host, error) {
	args := i.Called(serviceName)
	return args.Get(0).([]model.Host), args.Error(1)
}
func (i *MockedInventory) UpdateHost(serviceName string, hostName string, ingressAddr, substanceAddr, egressHost *string) (host model.Host, err error) {
	args := i.Called(serviceName, hostName, ingressAddr, substanceAddr, egressHost)
	return args.Get(0).(model.Host), args.Error(1)
}
func (i *MockedInventory) IdempotentService(serviceName string, param model.IdempotentServiceParam) (changed bool, err error) {
	args := i.Called(serviceName, param)
	return args.Bool(0), args.Error(1)
}

func TestPostService(t *testing.T) {
	inventory := MockedInventory{}
	server := NewServer(&inventory, model.CtlAPIConf{}, logrus.New())
	sut := httptest.NewServer(server)
	defer sut.Close()
	client, _ := NewClient(sut.URL, 60*time.Second)

	expect := model.Service{
		Name:     "ban",
		Protocol: "HTTP",
	}
	inventory.On("RegisterService", "test1", "HTTP").Return(expect, nil)

	actual, status, err := client.PostService("test1", PostServiceReq{Protocol: "HTTP"})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, status)
	assert.Equal(t, expect, actual)

	// error
	inventory.On("RegisterService", "test2", "unknown").Return(expect, errors.New("something happens"))
	actual, status, err = client.PostService("test2", PostServiceReq{Protocol: "unknown"})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, status)
}

func TestGetService(t *testing.T) {
	inventory := MockedInventory{}
	server := NewServer(&inventory, model.CtlAPIConf{}, logrus.New())
	sut := httptest.NewServer(server)
	defer sut.Close()
	client, _ := NewClient(sut.URL, 60*time.Second)

	expect := model.IdempotentServiceParam{
		Protocol: "HTTP",
		Hosts: []model.Host{{
			Name:          "host1",
			IngressAddr:   model.Address{Hostname: "host1", Port: 1234},
			SubstanceAddr: model.Address{Hostname: "127.0.0.1", Port: 5678},
			EgressHost:    "hostname",
		}},
	}
	expectsvc := expect.NewService("test1")
	inventory.On("GetService", "test1").Return(expectsvc, true, nil)
	inventory.On("GetHostsOfService", "test1").Return(expect.Hosts, nil)

	actual, status, err := client.GetService("test1")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, expect, actual)

	// error
	inventory.On("GetService", "test2").Return(expectsvc, false, nil)
	actual, status, err = client.GetService("test2")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestIdempotentService(t *testing.T) {
	inventory := MockedInventory{}
	server := NewServer(&inventory, model.CtlAPIConf{}, logrus.New())
	sut := httptest.NewServer(server)
	defer sut.Close()
	client, _ := NewClient(sut.URL, 60*time.Second)

	param := model.IdempotentServiceParam{
		Protocol: "TCP",
		DependentServices: []model.DependentService{
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
	inventory.On("IdempotentService", "test1", param).Return(true, nil)

	actual, status, err := client.PutService("test1", param)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, PutServiceResp{Changed: true}, actual)

	// error
	inventory.On("IdempotentService", "test2", param).Return(false, errors.New("error"))
	actual, status, err = client.PutService("test2", param)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, status)
}
