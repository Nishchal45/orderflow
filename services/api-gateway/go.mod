module github.com/Nishchal45/orderflow/services/api-gateway

go 1.26.2

replace github.com/Nishchal45/orderflow/pkg => ../../pkg

require (
	github.com/Nishchal45/orderflow/pkg v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.6.0
	github.com/rs/zerolog v1.35.0
)

require (
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.42.0 // indirect
)
