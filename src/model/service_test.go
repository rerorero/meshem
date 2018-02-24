package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceValidate(t *testing.T) {
	depends := []DependentService{
		DependentService{
			Name:       "valid1",
			EgressPort: 9001,
		},
		DependentService{
			Name:       "valid2",
			EgressPort: 9002,
		},
	}
	s := Service{
		Name:              "service",
		HostNames:         []string{"valid"},
		Protocol:          ProtocolHTTP,
		DependentServices: depends,
	}

	s.Name = "valid-01_32"
	assert.NoError(t, s.Validate())

	// invalid service name
	s.Name = "ivalid.aaa"
	assert.Error(t, s.Validate())

	// invalid host names
	s.Name = "valid"
	s.HostNames = append(s.HostNames, "in.valid")
	assert.Error(t, s.Validate())

	// invalid protocol
	s.HostNames = []string{"valid"}
	s.Protocol = "invalid"
	assert.Error(t, s.Validate())

	// duplicated service names
	s.Protocol = ProtocolHTTP
	err := s.AppendDependent(DependentService{Name: "valid1", EgressPort: 9003})
	assert.Error(t, err)
	s.DependentServices = append(depends, DependentService{Name: "valid1", EgressPort: 9003})
	assert.Error(t, s.Validate())

	// duplicated service port
	s.DependentServices = depends
	err = s.AppendDependent(DependentService{Name: "valid3", EgressPort: 9001})
	assert.Error(t, err)
	s.DependentServices = append(depends, DependentService{Name: "valid3", EgressPort: 9001})
	assert.Error(t, s.Validate())

	// invalid egress port
	s.DependentServices = depends
	err = s.AppendDependent(DependentService{Name: "valid3"})
	assert.Error(t, err)
	s.DependentServices = append(depends, DependentService{Name: "valid3"})
	assert.Error(t, s.Validate())

	// can append
	s.DependentServices = depends
	err = s.AppendDependent(DependentService{Name: "valid3", EgressPort: 9003})
	assert.NoError(t, err)
	assert.Equal(t, len(depends)+1, len(s.DependentServices))

	// can remove
	ok := s.RemoveDependent("valid3")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, len(depends), len(s.DependentServices))
}

func TestEqualsSserviceDependencies(t *testing.T) {
	a := []DependentService{
		DependentService{
			Name:       "abc",
			EgressPort: 9002,
		},
		DependentService{
			Name:       "ab",
			EgressPort: 9001,
		},
		DependentService{
			Name:       "abcd",
			EgressPort: 9003,
		},
	}
	sameA := []DependentService{
		DependentService{
			Name:       "abcd",
			EgressPort: 9003,
		},
		DependentService{
			Name:       "ab",
			EgressPort: 9001,
		},
		DependentService{
			Name:       "abc",
			EgressPort: 9002,
		},
	}
	b := []DependentService{
		DependentService{
			Name:       "abcd",
			EgressPort: 9003,
		},
		DependentService{
			Name:       "abc",
			EgressPort: 9002,
		},
	}
	c := []DependentService{
		DependentService{
			Name:       "abcd",
			EgressPort: 9003,
		},
		DependentService{
			Name:       "abc",
			EgressPort: 9001,
		},
		DependentService{
			Name:       "ab",
			EgressPort: 9002,
		},
	}
	assert.True(t, EqualsServiceDependencies(a, sameA))
	assert.False(t, EqualsServiceDependencies(a, b))
	assert.False(t, EqualsServiceDependencies(a, c))
}
