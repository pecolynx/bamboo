# bamboo

[![test workflow](https://github.com/pecolynx/bamboo/actions/workflows/test.yml/badge.svg)](https://github.com/pecolynx/bamboo/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/pecolynx/bamboo/graph/badge.svg?token=PDYjoKPa2z)](https://codecov.io/gh/pecolynx/bamboo)


bamboo is a library for distributing work across machines with asynchronous communication.

## Overview

* Workers are applications that execute time-consuming processes.
* ClientApp combines worker processing.
* ClientApp and Workers communicate asynchronously using Redis, for example.


```mermaid
sequenceDiagram
autonumber

participant ClientApp
participant Worker1
box Redis
participant RedisList_for_Worker1
participant RedisPubSub_for_Request1
end

Worker1 -->> RedisList_for_Worker1: BRPOP to wait a message for Worker1

ClientApp -->> RedisPubSub_for_Request1: SUBSCRIBE result for request1
ClientApp -->> RedisList_for_Worker1: LPUSH message
RedisList_for_Worker1 ->> Worker1: Fetch message
Worker1 ->> Worker1: process
Worker1 ->> RedisPubSub_for_Request1: Conn
Worker1 -->> RedisPubSub_for_Request1: PUBLISH result of request1
Worker1 ->> RedisPubSub_for_Request1: Close
RedisPubSub_for_Request1 ->> ClientApp: Fetch result
```

## Components

### Worker

1. Worker consumes a request.
2. Worker creates a Worker Job from the request.
3. Worker dispatches the Worker Job to the goroutine worker.
4. Worker Job gets the result.
5. Worker Job Publishes the result.

### Worker Client

1. Worker Client starts to subscribes to a result first.
2. Worker Client produces a request.
3. Worker Client receives the result after Worker Job publishes the result.



### Request Producer / Consumer

|Middleware|Component|
|---|---|
|Redis|Redis List can be used as a Request Producer / Consumer.|
|Kafka|Kafka can be used as a Request Producer / Consumer.|

#### Result Publisher / Subscriber

|Middleware|Component|
|---|---|
|Redis|Redis Pub/Sub can be used as a Result Publisher / Subscriber.|

## Installation

## Example

## Development

* https://vektra.github.io/mockery/latest/installation/

