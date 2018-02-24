package model

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Address struct {
	Hostname string `json:"host" yaml:"host"`
	Port     uint32 `json:"port" yaml:"port"`
}

func (addr *Address) String() string {
	return fmt.Sprintf("%s:%d", addr.Hostname, addr.Port)
}

func ParseAddress(s string) (*Address, error) {
	pair := strings.Split(s, ":")
	if len(pair) != 2 {
		return nil, fmt.Errorf("Invalid Address: %s", s)
	}
	port, err := strconv.Atoi(pair[1])
	if err != nil {
		return nil, errors.Wrapf(err, "Invalid port address: %s", s)
	}

	return &Address{pair[0], uint32(port)}, nil
}

// TODO: replaced with hash?
func (addr *Address) ListenerSuffix() string {
	return fmt.Sprintf("%s-%d", strings.Replace(addr.Hostname, ".", "", -1), addr.Port)
}
