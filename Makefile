.PHONY: compile-el compile-dl clean protoc lint build unit-tests integration-tests-churner integration-tests-indexer integration-tests-inabox integration-tests-inabox-nochurner integration-tests-graph-indexer

PROTOS := ./api/proto
PROTOS_DISPERSER := ./disperser/api/proto
PROTO_GEN := ./api/grpc
PROTO_GEN_DISPERSER_PATH = ./disperser/api/grpc

compile-el:
	cd contracts && ./compile.sh compile-el

compile-dl:
	cd contracts && ./compile.sh compile-dl

clean:
	find $(PROTO_GEN) -name "*.pb.go" -type f | xargs rm -rf
	mkdir -p $(PROTO_GEN)
	find $(PROTO_GEN_DISPERSER_PATH) -name "*.pb.go" -type f | xargs rm -rf
	mkdir -p $(PROTO_GEN_DISPERSER_PATH)

protoc: clean
	protoc -I $(PROTOS) \
	--go_out=$(PROTO_GEN) \
	--go_opt=paths=source_relative \
	--go-grpc_out=$(PROTO_GEN) \
	--go-grpc_opt=paths=source_relative \
	$(PROTOS)/**/*.proto
	# Generate Protobuf for sub directories of ./api/proto/disperser
	protoc -I $(PROTOS_DISPERSER) \
	--go_out=$(PROTO_GEN_DISPERSER_PATH) \
	--go_opt=paths=source_relative \
	--go-grpc_out=$(PROTO_GEN_DISPERSER_PATH) \
	--go-grpc_opt=paths=source_relative \
	$(PROTOS_DISPERSER)/**/*.proto

lint:
	golint -set_exit_status ./...
	go tool fix ./..
	golangci-lint run

build: 
	# cd churner && make build
	cd disperser && make build
	# cd node && make build
	cd retriever && make build
	# cd tools/traffic && make build

unit-tests:
	./test.sh

integration-tests-churner:
	go test -v ./churner/tests

integration-tests-indexer:
	go test -v ./core/indexer

integration-tests-node-plugin:
	go test -v ./node/plugin/tests

integration-tests-inabox:
	make build 
	cd inabox && make run-e2e

integration-tests-inabox-nochurner:
	make build
	cd inabox && make run-e2e-nochurner

integration-tests-graph-indexer:
	make build 
	go test -v ./core/thegraph
