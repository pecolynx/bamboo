SHELL=/bin/bash

.PHONY: gen-proto
gen-proto:
	@pushd ./worker-redis-redis && \
	protoc --go_out=./ --go_opt=paths=source_relative \
        --go-grpc_out=./ --go-grpc_opt=paths=source_relative \
        redis_redis.proto && \
	popd
	cp ./worker-redis-redis/redis_redis.pb.go ./calc-app/

	@pushd ./goroutine-app && \
	protoc --go_out=./ --go_opt=paths=source_relative \
        --go-grpc_out=./ --go-grpc_opt=paths=source_relative \
        goroutine_app.proto && \
	popd
	
.PHONY: run-worker-redis-redis
run-worker-redis-redis:
	@pushd ./worker-redis-redis && \
	go run . && \
	popd

.PHONY: run-calc-app
run-calc-app:
	@pushd ./calc-app && \
	go run . && \
	popd

.PHONY: run-goroutine-app
run-goroutine-app:
	@pushd ./goroutine-app && \
	go run . && \
	popd

.PHONY: lint
docker-up:
	@docker compose -f ./docker-compose.yml up -d

.PHONY: lint
docker-down:
	@docker compose -f ./docker-compose.yml down

docker-build:
	@pushd ./worker-redis-redis && \
	docker build -t worker-redis-redis ./ && \
	popd