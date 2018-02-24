FROM envoyproxy/envoy-alpine:latest

RUN mkdir -p /var/log/envoy
CMD /usr/local/bin/envoy  -c /etc/envoy.yaml -l trace
