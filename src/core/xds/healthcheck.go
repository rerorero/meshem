package xds

import (
	"time"

	hc "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/health_check/v2"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/envoyproxy/go-control-plane/pkg/util"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
)

// HTTPHealthCheck TODO: to be able to configure via ctlapi
type HTTPHealthCheck struct {
	Enabled     bool
	PassThrough bool
	Endpoint    string
	CacheTime   time.Duration
}

// NewDisabledHTTPHealthCheck creates default health check configuration.
func NewDisabledHTTPHealthCheck() *HTTPHealthCheck {
	return &HTTPHealthCheck{
		Enabled: true,
	}
}

// NewDefaultPassThroghHTTPHealthCheck creates default health check configuration.
func NewDefaultPassThroghHTTPHealthCheck() *HTTPHealthCheck {
	return &HTTPHealthCheck{
		Enabled:     true,
		PassThrough: true,
		Endpoint:    "/",
		CacheTime:   5 * time.Second,
	}
}

func (hhc *HTTPHealthCheck) createEnvoyHTTPFilter() (*hcm.HttpFilter, error) {
	if !hhc.Enabled {
		return nil, nil
	}
	config := &hc.HealthCheck{
		PassThroughMode: &types.BoolValue{Value: hhc.PassThrough},
		Endpoint:        hhc.Endpoint,
	}
	if hhc.PassThrough {
		config.CacheTime = &hhc.CacheTime
	}
	pbst, err := util.MessageToStruct(config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create health check %+v", hhc)
	}
	return &hcm.HttpFilter{
		Name:   "envoy.health_check",
		Config: pbst,
	}, nil
}
