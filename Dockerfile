########### Build ##########
FROM artifactory.algol60.net/docker.io/library/golang:alpine AS build

RUN apk add --no-cache git build-base

RUN mkdir -p /build
COPY . /build
RUN cd /build && go build ./...
RUN cd /build && go test -v ./...
RUN cd /build && go build -o /usr/local/bin/spire-tokens

########## Runtime ##########
FROM artifactory.algol60.net/docker.io/library/alpine:latest AS runtime

RUN apk add --no-cache bash curl

COPY --from=build /usr/local/bin/spire-tokens /usr/local/bin/spire-tokens
COPY ./.version /tokens-version
EXPOSE 54440/tcp
ENTRYPOINT ["/usr/local/bin/spire-tokens"]
