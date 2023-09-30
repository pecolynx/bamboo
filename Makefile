SHELL=/bin/bash

.PHONY: gen-proto
gen-proto:
	@protoc --go_out=./proto --go_opt=paths=source_relative \
        --go-grpc_out=./proto --go-grpc_opt=paths=source_relative \
        ./bamboo.proto

.PHONY: gen-src
gen-src:
	mockery

.PHONY: update-mod
update-mod:
	@pushd . && \
		go get -u ./... && \
	popd

.PHONY: lint
lint:
	@pushd ./ && \
		golangci-lint run --config .github/.golangci.yml && \
	popd

.PHONY: work-init
work-init:
	@go work init

.PHONY: work-use
work-use:
	@go work use ./ \
	example/worker-redis-redis \
	example/calc-app \
	example/goroutine-app \


.PHONY: test-docker-up
test-docker-up:
	@docker-compose -f docker-compose-test.yml up -d

.PHONY: test-docker-down
test-docker-down:
	@docker-compose -f docker-compose-test.yml down
