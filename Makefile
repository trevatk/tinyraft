
lint:
	golangci-lint run

test:
	go test ./...

test_coverage:
	go test ./... -coverprofile=coverage.out

bootstrap:
	export NODE_ID="00000000-0000-0000-0000-000000000000"
	export NODE_ADVERTISE_ADDR=":50051"
	export NODE_LISTEN_ADDR=":50051"
	export RAFT_BOOTSTRAP="true"
	export RAFT_DATA_DIR="data/00000000-0000-0000-0000-000000000000"
	go run examples/raft.go

node1:
	export NODE_ID="00000000-0000-0000-0000-000000000001"
	export NODE_ADVERTISE_ADDR=":50052"
	export NODE_LISTEN_ADDR=":50052"
	export RAFT_JOIN="localhost:50051"
	export RAFT_BOOTSTRAP="false"
	export RAFT_DATA_DIR="data/00000000-0000-0000-0000-000000000001"
	export RAFT_VOTER="true"
	go run examples/raft.go

node2:
	export NODE_ID="00000000-0000-0000-0000-000000000002"
	export NODE_ADVERTISE_ADDR=":50053"
	export NODE_LISTEN_ADDR=":50053"
	export RAFT_JOIN="localhost:50051"
	export RAFT_BOOTSTRAP="false"
	export RAFT_DATA_DIR="data/00000000-0000-0000-0000-000000000002"
	export RAFT_VOTER="true"
	go run examples/raft.go

nonvoter:
	go run examples/raft.go

protos:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		protos/raft/v1/raft_service.proto