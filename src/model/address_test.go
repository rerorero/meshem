package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAddress(t *testing.T) {
	addr, err := ParseAddress("1.2.3.4:5678")
	assert.NoError(t, err)
	assert.Equal(t, addr, &Address{"1.2.3.4", 5678})

	_, err = ParseAddress("1.2.3.4")
	assert.Error(t, err)

	_, err = ParseAddress("1.2:domain:80")
	assert.Error(t, err)

	_, err = ParseAddress("1.2.3.4:f")
	assert.Error(t, err)
}
