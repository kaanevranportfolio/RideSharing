module github.com/rideshare-platform/services/api-gateway

go 1.23.0

replace github.com/rideshare-platform/shared => ../../shared

require (
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.3
	github.com/graph-gophers/graphql-go v1.7.0
)

require (
	github.com/rideshare-platform/shared v0.0.0
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250707201910-8d1bb00bc6a7 // indirect
	google.golang.org/grpc v1.75.0
	google.golang.org/protobuf v1.36.8 // indirect
)
