SHELL=/bin/bash

.PHONY: gen-proto
gen-proto:
	@pushd ./worker-redis-redis && \
	protoc --go_out=./ --go_opt=paths=source_relative \
        --go-grpc_out=./ --go-grpc_opt=paths=source_relative \
        redis_redis.proto && \
	popd
	cp ./worker-redis-redis/redis_redis.pb.go ./calc-app/
	
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

docker-up:
	@docker compose -f ./docker-compose.yml up -d

docker-down:
	@docker compose -f ./docker-compose.yml down
