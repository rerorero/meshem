package core

import (
	"strconv"
	"time"

	"github.com/rerorero/meshem/src/model"
)

type VersionGenerator interface {
	New() model.Version
	Compare(l, r model.Version) int
}

type currentTimeGen struct{}

func NewCurrentTimeGenerator() VersionGenerator {
	return &currentTimeGen{}
}

// TODO: generate a distributed sequential unique number (in datastore?)
func (gen *currentTimeGen) New() model.Version {
	now := time.Now().UnixNano() / int64(time.Millisecond)
	return model.Version(strconv.FormatInt(now, 10))
}

// Compare returns 0 if l==r, -1 if l>r, +1 if l<r
func (gen *currentTimeGen) Compare(l, r model.Version) int {
	if l == r {
		return 0
	}

	ln, err := strconv.ParseInt(string(l), 10, 64)
	if err != nil {
		return 0
	}
	rn, err := strconv.ParseInt(string(r), 10, 64)
	if err != nil {
		return 0
	}

	if ln > rn {
		return -1
	} else if ln < rn {
		return 1
	}
	return 0
}

// MockedVersionGen is mock generator for testing
type MockedVersionGen struct {
	Version       model.Version
	CompareResult int
}

func (gen *MockedVersionGen) New() model.Version {
	return gen.Version
}

func (gen *MockedVersionGen) Compare(l, r model.Version) int {
	return gen.CompareResult
}
