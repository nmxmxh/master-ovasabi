# Physics Authority Script for Godot
# Handles physics simulation and event streaming to WASM layer

extends Node

class_name PhysicsAuthority

# Physics entities
var physics_entities = {}
var physics_events = []
var frame_id = 0
var last_physics_time = 0.0

# WebSocket connection to WASM
var ws_client: WebSocketClient
var ws_connected = false

# Physics settings
var physics_rate = 60.0
var max_entities = 1000
var world_bounds = Rect3(Vector3(-100, -100, -100), Vector3(200, 200, 200))

# Performance tracking
var performance_metrics = {
	"fps": 60.0,
	"frame_time": 16.67,
	"entity_count": 0,
	"collision_count": 0,
	"event_count": 0
}

# Physics rules
var physics_rules = {
	"gravity": Vector3(0, -9.81, 0),
	"damping": 0.99,
	"restitution": 0.8,
	"friction": 0.5
}

# LOD settings
var lod_distances = [50.0, 100.0, 200.0, 500.0]

func _ready():
	print("[PhysicsAuthority] Initializing physics authority...")
	
	# Connect to physics engine
	connect_physics_signals()
	
	# Initialize WebSocket connection
	init_websocket()
	
	# Start physics processing
	start_physics_processing()

func connect_physics_signals():
	# Connect to physics engine signals
	if PhysicsServer2D:
		PhysicsServer2D.connect("body_entered", self, "_on_body_entered")
		PhysicsServer2D.connect("body_exited", self, "_on_body_exited")
	
	if PhysicsServer3D:
		PhysicsServer3D.connect("body_entered", self, "_on_body_entered")
		PhysicsServer3D.connect("body_exited", self, "_on_body_exited")

func init_websocket():
	ws_client = WebSocketClient.new()
	ws_client.connect("connection_established", self, "_on_ws_connected")
	ws_client.connect("connection_closed", self, "_on_ws_disconnected")
	ws_client.connect("data_received", self, "_on_ws_data_received")
	
	# Connect to WASM WebSocket server
	var url = "ws://localhost:8080/ws"
	var error = ws_client.connect_to_url(url)
	if error != OK:
		print("[PhysicsAuthority] Failed to connect to WebSocket: ", error)

func _on_ws_connected(protocol):
	print("[PhysicsAuthority] WebSocket connected with protocol: ", protocol)
	ws_connected = true

func _on_ws_disconnected(was_clean_close):
	print("[PhysicsAuthority] WebSocket disconnected, clean: ", was_clean_close)
	ws_connected = false

func _on_ws_data_received():
	var packet = ws_client.get_peer(1).get_packet()
	var data = packet.get_string_from_utf8()
	
	# Parse and handle incoming data
	handle_ws_message(data)

func handle_ws_message(message: String):
	var json = JSON.parse(message)
	if json.error != OK:
		print("[PhysicsAuthority] Failed to parse WebSocket message: ", json.error)
		return
	
	var data = json.result
	match data.type:
		"physics:entity:spawn":
			handle_spawn_entity(data)
		"physics:entity:update":
			handle_update_entity(data)
		"physics:entity:destroy":
			handle_destroy_entity(data)
		"physics:campaign:rules":
			handle_campaign_rules(data)
		_:
			print("[PhysicsAuthority] Unknown message type: ", data.type)

func start_physics_processing():
	# Start physics processing timer
	var timer = Timer.new()
	timer.wait_time = 1.0 / physics_rate
	timer.connect("timeout", self, "_process_physics")
	timer.autostart = true
	add_child(timer)

func _process_physics(delta):
	frame_id += 1
	last_physics_time = Time.get_ticks_msec() / 1000.0
	
	# Process all physics entities
	process_physics_entities(delta)
	
	# Handle collisions
	handle_collisions()
	
	# Send physics events to WASM
	send_physics_events()
	
	# Update performance metrics
	update_performance_metrics(delta)

func process_physics_entities(delta):
	for entity_id in physics_entities.keys():
		var entity = physics_entities[entity_id]
		if not entity or not entity.has_method("_physics_process"):
			continue
		
		# Update entity physics
		entity._physics_process(delta)
		
		# Create update event
		var update_event = {
			"type": "physics:entity:update",
			"entity_id": entity_id,
			"position": [entity.global_position.x, entity.global_position.y, entity.global_position.z],
			"rotation": [entity.global_rotation.x, entity.global_rotation.y, entity.global_rotation.z, entity.global_rotation.w],
			"velocity": [entity.linear_velocity.x, entity.linear_velocity.y, entity.linear_velocity.z],
			"properties": {
				"mass": entity.mass,
				"restitution": entity.restitution,
				"friction": entity.friction
			},
			"timestamp": last_physics_time,
			"frame_id": frame_id
		}
		
		physics_events.append(update_event)

func handle_collisions():
	# This is a simplified collision handling
	# In a real implementation, you would use Godot's collision detection
	for entity_id in physics_entities.keys():
		var entity = physics_entities[entity_id]
		if not entity:
			continue
		
		# Check for collisions with other entities
		for other_id in physics_entities.keys():
			if entity_id == other_id:
				continue
			
			var other = physics_entities[other_id]
			if not other:
				continue
			
			# Calculate distance
			var distance = entity.global_position.distance_to(other.global_position)
			var collision_radius = entity.collision_radius + other.collision_radius
			
			if distance < collision_radius:
				# Create collision event
				var collision_event = {
					"type": "physics:entity:collision",
					"entity_id": entity_id,
					"data": {
						"entity_a": entity_id,
						"entity_b": other_id,
						"position": [entity.global_position.x, entity.global_position.y, entity.global_position.z],
						"normal": [1, 0, 0], # Simplified
						"force": entity.linear_velocity.length()
					},
					"timestamp": last_physics_time,
					"frame_id": frame_id
				}
				
				physics_events.append(collision_event)

func send_physics_events():
	if not ws_connected or physics_events.size() == 0:
		return
	
	# Send events in batches
	var batch_size = 10
	var events_to_send = physics_events.slice(0, batch_size)
	physics_events = physics_events.slice(batch_size)
	
	var batch = {
		"type": "physics:batch",
		"events": events_to_send,
		"frame_id": frame_id,
		"timestamp": last_physics_time
	}
	
	var json_string = JSON.print(batch)
	ws_client.get_peer(1).put_packet(json_string.to_utf8())

func handle_spawn_entity(data):
	var entity_id = data.entity_id
	var position = Vector3(data.position[0], data.position[1], data.position[2])
	var scale = Vector3(data.scale[0], data.scale[1], data.scale[2])
	var entity_type = data.properties.get("type", "default")
	
	# Create physics entity
	var entity = create_physics_entity(entity_id, position, scale, entity_type)
	if entity:
		physics_entities[entity_id] = entity
		print("[PhysicsAuthority] Spawned entity: ", entity_id)

func create_physics_entity(entity_id: String, position: Vector3, scale: Vector3, entity_type: String) -> Node:
	# Create a basic physics entity
	var entity = RigidBody.new()
	entity.name = entity_id
	entity.global_position = position
	entity.scale = scale
	
	# Add collision shape
	var collision_shape = CollisionShape.new()
	var box_shape = BoxShape.new()
	box_shape.extents = Vector3(1, 1, 1)
	collision_shape.shape = box_shape
	entity.add_child(collision_shape)
	
	# Add visual representation
	var mesh_instance = MeshInstance.new()
	var box_mesh = CubeMesh.new()
	mesh_instance.mesh = box_mesh
	entity.add_child(mesh_instance)
	
	# Set physics properties
	entity.mass = 1.0
	entity.restitution = physics_rules.restitution
	entity.friction = physics_rules.friction
	
	# Add to scene
	get_tree().current_scene.add_child(entity)
	
	return entity

func handle_update_entity(data):
	var entity_id = data.entity_id
	var entity = physics_entities.get(entity_id)
	if not entity:
		print("[PhysicsAuthority] Entity not found for update: ", entity_id)
		return
	
	# Update position
	if data.has("position"):
		var pos = data.position
		entity.global_position = Vector3(pos[0], pos[1], pos[2])
	
	# Update rotation
	if data.has("rotation"):
		var rot = data.rotation
		entity.global_rotation = Quat(rot[0], rot[1], rot[2], rot[3])
	
	# Update velocity
	if data.has("velocity"):
		var vel = data.velocity
		entity.linear_velocity = Vector3(vel[0], vel[1], vel[2])

func handle_destroy_entity(data):
	var entity_id = data.entity_id
	var entity = physics_entities.get(entity_id)
	if not entity:
		print("[PhysicsAuthority] Entity not found for destruction: ", entity_id)
		return
	
	# Remove from scene
	entity.queue_free()
	physics_entities.erase(entity_id)
	print("[PhysicsAuthority] Destroyed entity: ", entity_id)

func handle_campaign_rules(data):
	# Update physics rules based on campaign
	if data.has("gravity"):
		var gravity = data.gravity
		physics_rules.gravity = Vector3(gravity[0], gravity[1], gravity[2])
	
	if data.has("restitution"):
		physics_rules.restitution = data.restitution
	
	if data.has("friction"):
		physics_rules.friction = data.friction
	
	print("[PhysicsAuthority] Updated campaign rules")

func update_performance_metrics(delta):
	performance_metrics.fps = 1.0 / delta
	performance_metrics.frame_time = delta * 1000.0
	performance_metrics.entity_count = physics_entities.size()
	performance_metrics.event_count = physics_events.size()

func get_performance_metrics():
	return performance_metrics

func get_entity_count():
	return physics_entities.size()

func get_event_count():
	return physics_events.size()

func set_physics_rate(rate: float):
	physics_rate = rate
	print("[PhysicsAuthority] Physics rate set to: ", rate)

func set_world_bounds(bounds: Rect3):
	world_bounds = bounds
	print("[PhysicsAuthority] World bounds set to: ", bounds)

func add_physics_entity(entity_id: String, entity: Node):
	physics_entities[entity_id] = entity
	print("[PhysicsAuthority] Added physics entity: ", entity_id)

func remove_physics_entity(entity_id: String):
	physics_entities.erase(entity_id)
	print("[PhysicsAuthority] Removed physics entity: ", entity_id)

func get_physics_entity(entity_id: String) -> Node:
	return physics_entities.get(entity_id)

func get_all_entities():
	return physics_entities

func clear_all_entities():
	for entity in physics_entities.values():
		if entity and is_instance_valid(entity):
			entity.queue_free()
	physics_entities.clear()
	print("[PhysicsAuthority] Cleared all entities")

func _on_body_entered(body):
	# Handle body entered collision
	print("[PhysicsAuthority] Body entered: ", body.name)

func _on_body_exited(body):
	# Handle body exited collision
	print("[PhysicsAuthority] Body exited: ", body.name)

func _exit_tree():
	# Cleanup
	if ws_client:
		ws_client.disconnect_from_host()
	clear_all_entities()


