# kube-pulse-gateway
It is the entry point of Kube-Pulse Project where customer can connect with application through REST APIs

The Proto Gen Command
``
protoc \
  --proto_path=proto \
  --go_out=gen \
  --go_opt=paths=source_relative \
  --go-grpc_out=gen \
  --go-grpc_opt=paths=source_relative \
  health/v1/health.proto && echo "protoc OK" && ls gen/health/v1/
``