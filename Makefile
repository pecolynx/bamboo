SHELL=/bin/bash

.PHONY: gen-proto
gen-proto:
	@protoc --go_out=./ --go_opt=paths=source_relative \
        --go-grpc_out=./ --go-grpc_opt=paths=source_relative \
        ./bamboo.proto

.PHONY: gen-src
gen-src:
	@pushd ./ && \
		go generate ./... && \
	popd

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
	@go work use ./ example/worker-redis-redis example/calc-app


.PHONY: test-docker-up
test-docker-up:
	@docker-compose -f docker-compose-test.yml up -d

.PHONY: test-docker-down
test-docker-down:
	@docker-compose -f docker-compose-test.yml down
