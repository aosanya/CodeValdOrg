module github.com/aosanya/CodeValdOrg

go 1.25.3

require (
	github.com/aosanya/CodeValdSharedLib v0.0.0
	github.com/arangodb/go-driver v1.6.0
	github.com/soheilhy/cmux v0.1.5
	golang.org/x/crypto v0.27.0
	google.golang.org/grpc v1.79.1
	google.golang.org/protobuf v1.36.11
)

replace github.com/aosanya/CodeValdSharedLib => ../CodeValdSharedLib
