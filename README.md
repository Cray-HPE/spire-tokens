# Go API Server for spire-tokens

OpenAPI for spire-tokens which serves spire node join tokens. This container is deployed in the spire chart.

Re-running the generator:

```
mkdir -p ./schema
cp ./api/openapi.yaml ./schema
docker run --rm -v ${PWD}:/local openapitools/openapi-generator-cli generate \
  -i /local/schema/openapi.yaml \
  --package-name tokens \
  -g go-server \
  -o /local
rm -rf ./schema
```

## Environment Variables used

These environment variables get passed to the service via the spire chart.

TTL - How long spire join tokens should be valid for in seconds
SPIRE_DOMAIN - The spire cluster domain
NCN_ENTRY - NCN's base trust domain
COMPUTE_ENTRY - compute's base trust domain
NCN_CLUSTER_ENTRY - NCN cluster trust domain
COMPUTE_CLUSTER_ENTRY - computer cluster trust domain
