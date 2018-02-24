package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsString(t *testing.T) {
	slice := []string{"a", "bb", "c", "dd", "e"}
	var i int
	for _, s := range slice {
		idx, ok := ContainsString(slice, s)
		assert.True(t, ok)
		assert.Equal(t, idx, i)
		i++
	}
	_, ok := ContainsString(slice, "b")
	assert.False(t, ok)
}

func TestRemoveFromStringSlice(t *testing.T) {
	slice := []string{"a", "bb", "c", "dd", "e"}
	actual := RemoveFromStringSlice(slice, "bb")
	assert.ElementsMatch(t, []string{"a", "c", "dd", "e"}, actual)
	actual = RemoveFromStringSlice(slice, "b")
	assert.ElementsMatch(t, slice, actual)
}

func TestFilterNotContainsString(t *testing.T) {
	sa := []string{"a", "bb", "c", "dd", "e"}
	sb := []string{"aa", "bb", "cc", "dd", "ee"}
	assert.ElementsMatch(t, []string{"a", "c", "e"}, FilterNotContainsString(sa, sb))
	assert.ElementsMatch(t, []string{"aa", "cc", "ee"}, FilterNotContainsString(sb, sa))
}

func TestIntersectStringSlice(t *testing.T) {
	sa := []string{"a", "bb", "c", "dd", "e"}
	sb := []string{"aa", "bb", "cc", "dd", "ee"}
	assert.ElementsMatch(t, []string{"bb", "dd"}, IntersectStringSlice(sa, sb))
	assert.ElementsMatch(t, []string{"bb", "dd"}, IntersectStringSlice(sb, sa))
}
