## Texas Hold'em GRPC Server


#### Build protos:
```protoc -I poker/ poker/protobufs/poker.proto --go_out=plugins=grpc:poker```

#### Test:

```go run poker/server/server_test.go```

####  Get coverage:
```aidl
go test -coverprofile=coverage.out 
go tool cover -func=coverage.out
go tool cover -html=coverage.out
```

#### Run Commands

run server/client
```aidl
go run poker/run_server/main.go
go run poker/run_client/main.go

```