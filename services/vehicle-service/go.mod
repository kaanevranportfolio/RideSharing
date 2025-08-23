module github.com/rideshare-platform/services/vehicle-service

go 1.21

replace github.com/rideshare-platform/shared => ../../shared

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/rideshare-platform/shared v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.31.0
)
