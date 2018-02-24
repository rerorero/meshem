FROM golang:alpine AS build-env

ADD . /src
RUN cd /src && go build -o app1

FROM alpine
WORKDIR /app
COPY --from=build-env /src/app1 /app/

EXPOSE 9001
CMD ./app1
