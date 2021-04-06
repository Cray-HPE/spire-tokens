########### Build ##########
FROM arti.dev.cray.com/baseos-docker-master-local/golang:alpine3.12 AS build

RUN apk add --no-cache git build-base

RUN mkdir -p /build
COPY . /build
RUN cd /build && go build ./...
RUN cd /build && go test -v ./...
RUN cd /build && go build -o /usr/local/bin/spire-tokens

########## Runtime ##########
FROM arti.dev.cray.com/baseos-docker-master-local/alpine:3.13.2 AS runtime

RUN apk add --no-cache bash curl

COPY --from=build /usr/local/bin/spire-tokens /usr/local/bin/spire-tokens
COPY ./.version /tokens-version
EXPOSE 54440/tcp
ENTRYPOINT ["/usr/local/bin/spire-tokens"]
