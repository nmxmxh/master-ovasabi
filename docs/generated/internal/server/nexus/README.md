# Package nexus

## Types

### RedisEventBus

RedisEventBus is a multi-instance event bus using Redis Pub/Sub.

#### Methods

##### Close

##### Publish

##### Subscribe

##### Unsubscribe

### Server

Server implements the Nexus gRPC service with Redis-backed event streaming.

#### Methods

##### EmitEvent

EmitEvent receives an event from a client and broadcasts it to all subscribers.

##### PublishEvent

PublishEvent allows other parts of the system to publish events to all subscribers.

##### RegisterPattern

Stub implementation for RegisterPattern.

##### SubscribeEvents

SubscribeEvents streams real-time events to the client.
