# bamboo

bamboo is a library for distributing work across machines with asynchronous communication.


## Overview

* Workers are applications that execute time-consuming processes.
* App combines worker processing.
* App and Workers communicate asynchronously using Redis, for example.


```mermaid
sequenceDiagram

participant App
participant Worker1
participant Redis_Worker1

Worker1 -->> Redis_Worker1: BRPOP to wait a message for Worker1

App -->> Redis_Request1: SUBSCRIBE result for request1
App -->> Redis_Worker1: LPUSH message
Redis_Worker1 ->> Worker1: Fetch message
Worker1 ->> Worker1: process
Worker1 ->> Redis_Request1: Conn
Worker1 -->> Redis_Request1: PUBLISH result of request1
Worker1 ->> Redis_Request1: Close
Redis_Request1 ->> App: Fetch result
```


## Installation

## Example

## Development

* https://vektra.github.io/mockery/latest/installation/

