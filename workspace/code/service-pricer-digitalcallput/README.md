# digitalcallput

The owner of the repo was too lazy to replace the instructions below with something more specific to this service:

## Quick start with the template

1. Define your API in `proto/digitalcallput/v1/digitalcallput.proto`
2. Call `make lint-proto` to verify that all is good with your .proto file
3. Call `make grpc-generate` to generate `github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput` packages from proto definitions
4. Implement `github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput.(DigitalcallputServiceServer)` interface in `internal/grpcsvc/grpcsvc.go`
5. Update `grpcsvc.New` call in `internal/api` to correctly initialise your service
6. Run `go run cmd/digitalcallput/main.go`. gRPC service listens on :8090 by default, and HTTP service on :8080
7. Replace the content of this file with something that describes your service and what is it good for
