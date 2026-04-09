module github.com/Nishchal45/orderflow/services/order-service

go 1.26.2

replace github.com/Nishchal45/orderflow/pkg => ../../pkg

require (
	github.com/Nishchal45/orderflow/pkg v0.0.0
	github.com/rs/zerolog v1.35.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/lib/pq v1.12.3 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/segmentio/kafka-go v0.4.50 // indirect
	golang.org/x/sys v0.42.0 // indirect
)
