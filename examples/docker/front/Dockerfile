FROM golang:alpine AS build-env

ADD . /src
RUN cd /src && go build -o front

FROM alpine
WORKDIR /front
COPY --from=build-env /src/front /front/

EXPOSE 8080
CMD ./front
