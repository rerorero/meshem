package xds

import "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"

// Hasher returns node ID as an ID
type Hasher struct {
}

// ID function
func (h Hasher) ID(node *core.Node) string {
	if node == nil {
		return "unknown"
	}
	return node.Id
}
