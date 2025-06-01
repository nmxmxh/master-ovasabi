# Package adapters

## Types

### AIAdapter

AIAdapter implements a production-grade AI/ML protocol adapter for the Nexus bridge.

#### Methods

##### Capabilities

Capabilities returns the supported capabilities of the adapter.

##### Close

Close closes the AI/ML connection (stub).

##### Connect

Connect establishes a connection to the AI/ML service (stub).

##### Endpoint

Endpoint returns the AI/ML endpoint (if any).

##### HealthCheck

HealthCheck returns the health status of the adapter.

##### Protocol

Protocol returns the protocol name.

##### Receive

Receive is a stub for AIAdapter (simulate streaming/push if supported).

##### Send

Send sends a message to the AI/ML service for inference (stub).

### AMQPAdapter

#### Methods

##### Capabilities

##### Close

##### Connect

##### Endpoint

##### HealthCheck

##### Protocol

##### Receive

##### Send

### AMQPConfig

### AidAdapter

#### Methods

##### Capabilities

##### Close

##### Connect

##### Endpoint

##### HealthCheck

##### Protocol

##### Receive

##### Send

### AidConfig

### AirlineAdapter

#### Methods

##### Capabilities

##### Close

##### Connect

##### Endpoint

##### HealthCheck

##### Protocol

##### Receive

##### Send

### AirlineConfig

### BLEAdapter

BLEAdapter implements a production-grade Bluetooth Low Energy (BLE) protocol adapter for the Nexus
bridge.

#### Methods

##### Capabilities

Capabilities returns the supported capabilities of the adapter.

##### Close

Close closes the BLE connection (stub).

##### Connect

Connect establishes a connection to the BLE device (stub).

##### Endpoint

Endpoint returns the BLE device endpoint (if any).

##### HealthCheck

HealthCheck returns the health status of the adapter.

##### Protocol

Protocol returns the protocol name.

##### Receive

Receive starts a goroutine to listen for BLE messages and invokes the handler (stub).

##### Send

Send sends a message to the BLE device (stub).

### BLEConfig

BLEConfig holds configuration for the BLE adapter.

### CANAdapter

CANAdapter implements a production-grade CAN bus protocol adapter for the Nexus bridge. TODO:
Integrate with a real CAN library for your platform (e.g., github.com/brutella/can).

#### Methods

##### Capabilities

Capabilities returns the supported capabilities of the adapter.

##### Close

Close closes the CAN interface (stub).

##### Connect

Connect establishes a connection to the CAN bus (stub).

##### Endpoint

Endpoint returns the CAN interface endpoint (if any).

##### HealthCheck

HealthCheck returns the health status of the adapter.

##### Protocol

Protocol returns the protocol name.

##### Receive

Receive starts a goroutine to listen for CAN messages and invokes the handler (stub).

##### Send

Send writes a message to the CAN bus (stub).

### CoAPAdapter

CoAPAdapter implements a production-grade CoAP protocol adapter for the Nexus bridge.

#### Methods

##### Capabilities

Capabilities returns the supported capabilities of the adapter.

##### Close

Close closes the UDP client connection.

##### Connect

Connect establishes a UDP connection to the CoAP server.

##### Endpoint

Endpoint returns the configured endpoint address.

##### HealthCheck

HealthCheck returns the health status of the adapter.

##### Protocol

Protocol returns the protocol name.

##### Receive

Receive starts a goroutine to listen for incoming CoAP messages and invokes the handler.

##### Send

Send sends a message to the CoAP server using POST. Path is taken from msg.Metadata["coap_path"].

### CoAPConfig

CoAPConfig holds configuration for the CoAP adapter.

### GRPCAdapter

#### Methods

##### Capabilities

##### Close

##### Connect

##### Endpoint

##### HealthCheck

##### Protocol

##### Receive

##### Send

### HTTPAdapter

#### Methods

##### Capabilities

##### Close

##### Connect

##### Endpoint

##### HealthCheck

##### Protocol

##### Receive

##### Send

### HackingAdapter

HackingAdapter implements a production-grade C2/hacking protocol adapter for the Nexus bridge.

#### Methods

##### Capabilities

Capabilities returns the supported capabilities of the adapter.

##### Close

Close closes the C2 connection (stub).

##### Connect

Connect establishes a connection to the C2 endpoint (stub).

##### Endpoint

Endpoint returns the C2 endpoint (if any).

##### HealthCheck

HealthCheck returns the health status of the adapter.

##### Protocol

Protocol returns the protocol name.

##### Receive

Receive starts a goroutine to listen for C2/exploit responses and invokes the handler (stub).

##### Send

Send sends a command or exploit to the C2 endpoint (stub).

### HackingConfig

HackingConfig holds configuration for the Hacking adapter.

### HospitalAdapter

#### Methods

##### Capabilities

##### Close

##### Connect

##### Endpoint

##### HealthCheck

##### Protocol

##### Receive

##### Send

### HospitalConfig

### KafkaAdapter

#### Methods

##### Capabilities

##### Close

##### Connect

##### Endpoint

##### HealthCheck

##### Protocol

##### Receive

Receive starts a goroutine to consume messages from Kafka and invokes the handler.

##### Send

Send writes a message to the Kafka topic.

### KafkaConfig

### MQTTAdapter

#### Methods

##### Capabilities

##### Close

##### Connect

##### Endpoint

##### HealthCheck

##### Protocol

##### Receive

##### Send

### MQTTConfig

### MilitaryAdapter

#### Methods

##### Capabilities

##### Close

##### Connect

##### Endpoint

##### HealthCheck

##### Protocol

##### Receive

Receive starts a goroutine to simulate receiving encrypted messages and invokes the handler.

##### Send

Send encrypts and sends a message to the tactical endpoint.

### MilitaryConfig

### SatelliteAdapter

#### Methods

##### Capabilities

##### Close

##### Connect

##### Endpoint

##### HealthCheck

##### Protocol

##### Receive

Receive starts a goroutine to simulate receiving telemetry or downlink and invokes the handler.

##### Send

Send sends a command or telemetry to the satellite system.

### SatelliteConfig

### SerialAdapter

SerialAdapter implements a production-grade serial protocol adapter for the Nexus bridge.

#### Methods

##### Capabilities

Capabilities returns the supported capabilities of the adapter.

##### Close

Close closes the serial port connection.

##### Connect

Connect establishes a connection to the serial port using the provided config.

##### Endpoint

Endpoint returns the serial port endpoint (if any).

##### HealthCheck

HealthCheck returns the health status of the adapter.

##### Protocol

Protocol returns the protocol name.

##### Receive

Receive starts a goroutine to listen for serial messages and invokes the handler.

##### Send

Send writes a message to the serial port.

### SerialConfig

SerialConfig holds configuration for the Serial adapter.

### TVAdapter

#### Methods

##### Capabilities

##### Close

##### Connect

##### Endpoint

##### HealthCheck

##### Protocol

##### Receive

##### Send

### TVConfig

### WebSocketAdapter

WebSocketAdapter implements a production-grade WebSocket protocol adapter for the Nexus bridge.

#### Methods

##### Capabilities

##### Close

##### Connect

##### Endpoint

##### HealthCheck

##### Protocol

##### Receive

Receive sets the handler for incoming messages.

##### Send

Send sends a message to a specific WebSocket client using a buffered channel.

### WebSocketConfig
