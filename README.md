# fitest
Take home tech test for Face IT

## Outstanding items

You have been waiting for this for quite a while and I really need to send it. Given more time I would have added a few things
* A Jaeger exporter for the telemetry tracing. As it stands, the service creates traces but they don't go anywhere
* A Demo Client. I have included example calls which can be made using `grpcurl` but a demo client would have been an improvement
* RPC Middleware. There should be GRPC middleware for the telemetry tracing, to either extract or create a request ID and to set a request timeout
* More descriptive errors. The RPC is only the GRPC error codes with a simple message. GRPC provides a mechanism for a richer error description

## Running tests

Use the provided docker-compose to run an instance of mongodb and then run the tests using make
```shell
docker compose up -d
make test
```

The included docker-compose file will also build and run an instance of the service

### Test Coverage

A Test coverage report is available by replacing `make test` with `make test_cover`
Excluding the generated code the test coverage is currently showing at 87%. Ideally I would improve this somewhat by declaring interfaces for the mongo client so that I could test some edge cases there, because that is the area bringing the coverage down most.

The stub implementation of a message bus could also be improved, but it's just a stub.

## Overview

### RPC API

I'm fairly sure Vesim mentioned GRPC in the interview so I have implemented the RPC (aside from the health check) using GRPC. 
The protobuf and generated code can be found in the userspb folder off the root. The pkg/rpc package implements the RPC Server for the Users service.
The only function of the pkg/rpc package is to convey requests to a provided implementation of the rpc.UsersService. The RPC methods do not publish the hash of the users password

### pkg/user

The user package provides an implementation of the rpc.UsersService interface. It is responsible for validating incoming requests and forwarding them on to the userstore.
It also listens to a stream of events from the userstore and forwarding them on to an event bus (See the section on transactional outbox below)

When creating this package I made the following assumptions.
* The email address is read only once created
* The nickname is read only once created
* Even though we are not providing log in, plain text passwords are bad, so it still hashes the password before storing it.
* When updating a user, it is not necessary to validate their old password before updating it to a new one.

The service uses optimistic locking to prevent updates overwriting with stale data.

### pkg/userstore
The userstore package is a repository for the data stored by the service, implemented on top of mongodb.
It provides CRUD functions for user records, and also provides a stream of mutation events (see section on transactional outbox below)

### pkg/log
pkg/log provides a very basic structured logger, implemented on top of the uber zap logger. 

## Transactional Outbox

The principle of the transactional outbox pattern is to make the decision to mutate a record and the decision to send an event regarding that mutation a single atomic event.
In this implementation it is achieved by storing both the user object and an array of events in each document.
The database is able to read off events which have not yet been processsed or whose processing is timed out, and update them in a single atomic transaction.
These are then provided to a consumer. Once the consumer has verified that the event has been passed on to a message bus, the event can be marked as processed, which removes it from the document.
This provides an "at least once" guarantee for domain events, even in the face of the underlying message bus being unavailable for some time. 
It also decouples the process of sending domain events from the proceess of making mutations, so the RPC API should remain responsive.
The implementation here would need further work for a high traffic service since it only sends one event at a time.

## Healthcheck

The service provides a simple http healthcheck, implmented in the pkg/health package. The userstore and user packages provide implementations of the health.Monitor interface so their state can be included in the healthcheck

The healthcheck of the service run by the included docker compose can be called with
```shell
curl -v http://localhost:9090/healthy
```

## Running and interacting with the service

The included docker-compose file will build and run an instance of the service. The service uses GRPC. Some examples of making calls to the service using the `grpcurl` tool are provided below

### Creating a user
```shell
grpcurl -d '{"firstName": "Max", "lastName":"Mustermann", "nickname": "maxmust", "email": "maxmust@example.com", "password": "password123", "confirmPassword": "password123", "country": "DE"}' -plaintext localhost:8080 Users.CreateUser
```

### Updating a user
```shell
grpcurl -d '{"id": "REPLACE WITH A USER ID", "firstName": "New fist name", "lastName":"New last name", "country": "NL", "version": 1}' -plaintext localhost:8080 Users.UpdateUser
```

### Deleting a user
```shell
grpcurl -d '{"id": "REPLACE WITH A USER ID"}' -plaintext localhost:8080 Users.DeleteUser
```

### Listing users living in DE
```shell
grpcurl -d '{"country":"DE"}' -plaintext localhost:8080 Users.FindUsers
```

The FindUsers RPC also supports a page number, a maximum length for the result, and the ability to request user records created after a certain date