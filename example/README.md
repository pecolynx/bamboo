# bamboo example

[![test workflow](https://github.com/pecolynx/bamboo/actions/workflows/test.yml/badge.svg)](https://github.com/pecolynx/bamboo/actions/workflows/test.yml)

## Prerequisite

* docker compose
* protoc
* golang

## How to run

```bash
make docker-up
```

```bash
make run-worker-redis-redis
```

```bash
make run-calc-app
```

## Result
Open [JAEGER](http://localhost:16686/search)
![jaeger](jaeger.png "jaeger")


http://localhost:9090/graph?g0.expr=bamboo_requests_received&g0.tab=0&g0.stacked=0&g0.show_exemplars=0&g0.range_input=1h