module hello

go 1.15

require (
	google.golang.org/grpc v1.36.0 // indirect
	local.packages/chat v0.0.0-00010101000000-000000000000 // indirect
)

replace local.packages/chat => ./chat
