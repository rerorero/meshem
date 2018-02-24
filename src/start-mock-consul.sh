#!/bin/sh

docker run -d --rm --name=${CONSUL_MOCK_NAME:-consul_mock} \
  -p ${CONSUL_PORT:-18500}:8500 \
  -e CONSUL_BIND_INTERFACE=eth0 \
  -e 'CONSUL_LOCAL_CONFIG={"acl_datacenter": "dc1", "acl_default_policy": "deny", "acl_master_token": "master"}' \
  consul:${CONSUL_VERSION:-1.0.3}
