module github.com/usip/adapters/custom-engine

go 1.21.4

require (
	github.com/usip/backend v0.0.0
	google.golang.org/grpc v1.64.0
	google.golang.org/protobuf v1.34.2
)

require (
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
)

// Resolve the shared schema/contract from the local backend module.
// The go.work file makes this seamless in-tree; the replace keeps
// `go mod tidy` and out-of-workspace builds working too.
replace github.com/usip/backend => ../../backend
