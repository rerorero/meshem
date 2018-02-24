FROM golang:1.9 as builder

# build directories
RUN mkdir -p /go/src/github.com/rerorero/meshem
WORKDIR /go/src/github.com/rerorero/meshem
ADD . .

# Build meshem
RUN go get github.com/golang/dep/...
RUN dep ensure
RUN CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -installsuffix netgo --ldflags '-extldflags "-static"' -o /meshem ./src/meshem
RUN CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -installsuffix netgo --ldflags '-extldflags "-static"' -o /meshemctl ./src/meshemctl

# runner container
FROM alpine:latest
COPY --from=builder /meshem /bin/meshem
COPY --from=builder /meshemctl /bin/meshemctl
