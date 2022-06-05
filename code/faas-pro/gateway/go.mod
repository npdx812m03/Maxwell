module zhy.openfaas

go 1.13

require (
	github.com/docker/distribution v2.7.1+incompatible
	github.com/gorilla/mux v1.7.3
	github.com/nats-io/gnatsd v1.4.1 // indirect
	github.com/nats-io/go-nats v1.7.2 // indirect
	github.com/nats-io/go-nats-streaming v0.4.4 // indirect
	github.com/nats-io/nats-server v1.4.1 // indirect
	github.com/nats-io/nats-streaming-server v0.22.1 // indirect
	github.com/openfaas/faas v0.0.0-20190124174533-bfa869ec8c0c
	github.com/openfaas/faas-provider v0.0.0-20191005090653-478f741b64cb
	github.com/openfaas/nats-queue-worker v0.0.0-20191210110419-dea1c90b8cc6
	github.com/prometheus/client_golang v1.1.0
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/prometheus/common v0.7.0 // indirect
	go.uber.org/goleak v0.10.0
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v0.18.2
)
