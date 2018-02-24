#!/bin/sh
container=${CONSUL_MOCK_NAME:-consul_mock}

docker ps | grep "$container" && docker stop "$container"

