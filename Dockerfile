########### Build ##########
FROM dtr.dev.cray.com/baseos/golang:1.14-alpine AS build

RUN apk add --no-cache git build-base

RUN mkdir -p /build
COPY . /build
RUN cd /build && go build ./...
RUN cd /build && go test -v ./...
RUN cd /build && go build -o /usr/local/bin/spire-tokens

########## Runtime ##########
FROM dtr.dev.cray.com/baseos/alpine:3.12 AS runtime

RUN apk add --no-cache bash curl

COPY --from=build /usr/local/bin/spire-tokens /usr/local/bin/spire-tokens
COPY ./.version /tokens-version
EXPOSE 54440/tcp
ENTRYPOINT ["/usr/local/bin/spire-tokens"]
