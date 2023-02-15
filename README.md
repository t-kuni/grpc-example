# About

Example using gRPC

# Requirements

* go 1.20

# Usage

Compile protocol buffer.

```bash
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    grpc/chat/chat.proto
```

Start server.

```bash
go run server/server.go
```

Start client.

```bash
go run client/client.go
```
