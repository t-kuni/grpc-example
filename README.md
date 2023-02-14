# About

Samples with gRPC

# Requirements

* go 1.20

# Usage

```bash
go run server/server.go
```

```bash
go run client/client.go
```

# compile protocol buffer

```bash
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    grpc/chat/chat.proto
```