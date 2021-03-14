module routeguide

go 1.15

replace local.packages/pb => ./routeguide

require (
	github.com/golang/protobuf v1.4.3
	google.golang.org/grpc v1.36.0
	google.golang.org/grpc/examples v0.0.0-20210312231957-21976fa3e38a
	local.packages/pb v0.0.0-00010101000000-000000000000
)
