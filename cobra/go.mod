module cobra_practice

go 1.15

replace local.packages/cmd => ./cmd

require (
	github.com/spf13/cobra v1.1.3 // indirect
	github.com/spf13/viper v1.7.1 // indirect
	local.packages/cmd v0.0.0-00010101000000-000000000000
)
