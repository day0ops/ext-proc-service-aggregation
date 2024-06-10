# Ext Proc Service Example for API Aggregation

This gRPC service demonstrates how to expose an interface to ext proc to aggregate multiple APIs.

External APIs are hardcoded to `jsonplaceholder.typicode.com`.

## Build

- Use `make build` to build the processor and the test services.
- To build and push the Docker images use `PUSH_MULTIARCH=true make docker`. By default it only builds `linux/amd64` & `linux/arm64`.
  - The images get pushed to `australia-southeast1-docker.pkg.dev/field-engineering-apac/public-repo` but you can override this with the env var ``
- Run make help for all the build directives.

