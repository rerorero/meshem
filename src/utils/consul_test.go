package utils

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPutGetKV(t *testing.T) {
	mock := NewConsulMock()

	// delete
	mock.Client.KV().DeleteTree("_testing", nil)

	err := mock.PutKV("_testing/foo/bar", "abc")
	assert.NoError(t, err)

	actual, ok, err := mock.GetKV("_testing/foo/bar")
	assert.NoError(t, err)
	assert.Equal(t, actual, "abc")
	assert.True(t, ok)

	_, err = mock.Client.KV().Delete("_testing/foo/bar", nil)
	assert.NoError(t, err)

	actual, ok, err = mock.GetKV("_testing/foo/bar")
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestGetKeys(t *testing.T) {
	mock := NewConsulMock()

	// delete
	mock.Client.KV().DeleteTree("_testing", nil)

	err := mock.PutKV("_testing/a/b", "a/b")
	assert.NoError(t, err)
	err = mock.PutKV("_testing/a/bb", "a/bb")
	assert.NoError(t, err)
	err = mock.PutKV("_testing/a/b/c", "a/b/c")
	assert.NoError(t, err)
	err = mock.PutKV("_testing/a/bbb/c", "a/b/c")
	assert.NoError(t, err)
	err = mock.PutKV("_testing/aa/b", "aa/b")
	assert.NoError(t, err)

	// recursive
	var keys []string
	keys, err = mock.GetKeys("_testing/a", true)
	assert.NoError(t, err)
	expected := []string{"_testing/a/b", "_testing/a/bb", "_testing/a/b/c", "_testing/a/bbb/c"}
	sort.Strings(keys)
	sort.Strings(expected)
	assert.Equal(t, keys, expected)

	// not recursive
	keys, err = mock.GetKeys("_testing/a", false)
	assert.NoError(t, err)
	expected = []string{"_testing/a/b", "_testing/a/bb", "_testing/a/bbb"}
	sort.Strings(keys)
	sort.Strings(expected)
	assert.Equal(t, keys, expected)

	// Get first children nodes
	keys, err = mock.GetSubKeyNames("_testing/a")
	assert.NoError(t, err)
	expected = []string{"b", "bb", "bbb"}
	sort.Strings(keys)
	sort.Strings(expected)
	assert.Equal(t, keys, expected)

	// not found
	keys, err = mock.GetKeys("_testing/ab", true)
	assert.NoError(t, err)
	assert.Empty(t, keys)
}
