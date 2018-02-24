package utils

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
)

type Consul struct {
	Client     *api.Client
	Datacenter string
}

func NewConsul(url *url.URL, token string, datacenter string) (*Consul, error) {
	config := api.DefaultConfig()
	config.Address = url.String()
	config.Token = token
	config.Datacenter = datacenter
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &Consul{
		Client:     client,
		Datacenter: datacenter,
	}, nil
}

func (c *Consul) GetKV(key string) (value string, ok bool, err error) {
	pair, _, err := c.Client.KV().Get(key, nil)
	if err != nil {
		return "", false, err
	}
	if pair == nil {
		return "", false, nil
	}
	return string(pair.Value), true, nil
}

// returns an error if key doesn't exist
func (c *Consul) GetKVExactly(key string) (value string, err error) {
	value, ok, err := c.GetKV(key)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("key=%s does not exist", key)
	}
	return value, nil
}

func (c *Consul) PutKV(key string, value string) error {
	pair := &api.KVPair{Key: key, Value: []byte(value)}
	_, err := c.Client.KV().Put(pair, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Consul) GetKeys(prefix string, recurse bool) ([]string, error) {
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}
	keys, _, err := c.Client.KV().Keys(prefix, "", nil)
	if err != nil {
		return nil, err
	}
	if recurse {
		return keys, nil
	}

	// remove sub trees if not recusive
	keymap := make(map[string]struct{})
	for _, key := range keys {
		sub := strings.Split(key[len(prefix):], "/")[0]
		if sub != "" {
			keymap[sub] = struct{}{}
		}
	}

	replacedKeys := []string{}
	for key := range keymap {
		replacedKeys = append(replacedKeys, prefix+key)
	}

	return replacedKeys, nil
}

// get keys which is first children of the prefix and removes the prefix
func (c *Consul) GetSubKeyNames(prefix string) ([]string, error) {
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}
	fullkeys, err := c.GetKeys(prefix, false)
	if err != nil {
		return nil, err
	}

	keys := []string{}
	prelen := len(prefix)
	for _, key := range fullkeys {
		keys = append(keys, string(key[prelen:]))
	}
	return keys, nil
}

// returns an error if key doesn't exist
func (c *Consul) GetKVBoolExactly(key string) (value bool, err error) {
	str, err := c.GetKVExactly(key)
	if err != nil {
		return false, err
	}
	if str == "true" {
		return true, nil
	} else if str == "false" {
		return false, nil
	}
	return false, fmt.Errorf("%s=%s is not boolean", key, str)
}

func (c *Consul) PutKVBool(key string, value bool) error {
	str := ""
	if value {
		str = "true"
	} else {
		str = "false"
	}
	return c.PutKV(key, str)
}

// returns an error if key doesn't exist
func (c *Consul) GetKVIntExactly(key string) (value int, err error) {
	str, err := c.GetKVExactly(key)
	if err != nil {
		return 0, err
	}

	num, err := strconv.Atoi(str)
	if err != nil {
		return 0, errors.Wrapf(err, "%s=%s is not int", key, str)
	}
	return num, nil
}

func (c *Consul) PutKVInt(key string, num int) error {
	return c.PutKV(key, strconv.Itoa(num))
}

// returns whether or not to delete
func (c *Consul) DeleteTreeIfExists(key string) (bool, error) {
	_, ok, err := c.GetKV(key)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	_, err = c.Client.KV().DeleteTree(key, nil)
	if err != nil {
		return false, err
	}

	return true, nil
}

// only for test
func NewConsulMock() *Consul {
	// It must be set the same as start-mock-consul.sh
	addr, err := url.Parse("http://127.0.0.1:18500")
	if err != nil {
		panic(err)
	}
	consul, err := NewConsul(addr, "master", "dc1")
	if err != nil {
		panic(err)
	}

	_, err = consul.Client.Agent().NodeName()
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			panic("It seems that the mock consul is not running. You need to execute start-mock-consul.sh.")
		}
		panic(err.Error())
	}
	return consul
}
