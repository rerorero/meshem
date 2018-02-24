package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHostValidate(t *testing.T) {
	host, err := NewHost("valid-01_32", "192.168.0.1:1234", "127.0.0.1:5678", "127.0.0.1")
	assert.NoError(t, err)
	assert.NoError(t, host.Validate())

	// invalid hostname
	host.Name = "ivalid.aaa"
	assert.Error(t, host.Validate())
	host.Name = "ivalidddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	assert.Error(t, host.Validate())

	// duplicate port
	host2, err := NewHost("valid-01_32", "192.168.0.1:1234", "192.168.0.1:1234", "127.0.0.1")
	assert.NoError(t, err)
	assert.Error(t, host2.Validate())
}

func TestUpdate(t *testing.T) {
	host, err := NewHost("name", "192.168.0.1:1234", "192.168.0.1:5678", "127.0.0.1")
	assert.NoError(t, err)
	assert.NoError(t, host.Validate())

	newIng := "192.168.0.1:9090"
	newSub := "192.168.0.1:8080"
	expect, err := NewHost("name", newIng, newSub, "127.0.0.1")
	assert.NoError(t, err)
	err = host.Update(&newIng, &newSub, nil)
	assert.NoError(t, err)
	assert.Equal(t, expect, host)

	newEgress := "192.168.0.2"
	err = host.Update(nil, nil, &newEgress)
	assert.NoError(t, err)
	expect, err = NewHost("name", newIng, newSub, newEgress)
	assert.NoError(t, err)
	assert.Equal(t, expect, host)
}
