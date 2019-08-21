## Texas Hold'em GRPC Server


#### Build protos:
```protoc -I poker/ poker/protobufs/poker.proto --go_out=plugins=grpc:poker```

#### Test:

```go run poker/server/server_test.go```

####  Get test coverage:

From within `server/` package:
```bash
go test -coverprofile=coverage.out 
go tool cover -func=coverage.out
go tool cover -html=coverage.out
rm coverage.out
```

#### Run Commands

run server/client
```bash
go run poker/run_server/main.go
go run poker/run_client/main.go
```