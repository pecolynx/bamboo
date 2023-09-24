SHELL=/bin/bash

.PHONY: gen-proto
gen-proto:
	@protoc --go_out=./ --go_opt=paths=source_relative \
        --go-grpc_out=./ --go-grpc_opt=paths=source_relative \
        ./bamboo.proto

.PHONY: lint
lint:
	@pushd ./ && \
		golangci-lint run --config .github/.golangci.yml && \
	popd
