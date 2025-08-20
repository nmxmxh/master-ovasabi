extends Node


var particle_count := 500000
var particles := []
var tick := 0
var buffer : PackedFloat32Array = PackedFloat32Array()

func _ready():
    print("[main] Godot headless main node ready.")
    # Ensure NexusClient (WebSocket client) is present in /root for orchestration
    if not has_node("/root/NexusClient"):
        print("[main] NexusClient not found, instancing...")
        var NexusClientScene = preload("res://scripts/ws_client.gd")
        var nexus_client = NexusClientScene.new()
        nexus_client.name = "NexusClient"
        get_tree().get_root().add_child(nexus_client)
        print("[main] NexusClient node added to /root.")
    else:
        print("[main] NexusClient already present in /root.")

    # Add nexus_bridge after NexusClient
    if not has_node("/root/nexus_bridge"):
        print("[main] nexus_bridge not found, instancing...")
        var NexusBridgeScene = preload("res://scripts/nexus_bridge.gd")
        var nexus_bridge = NexusBridgeScene.new()
        nexus_bridge.name = "nexus_bridge"
        get_tree().get_root().add_child(nexus_bridge)
        print("[main] nexus_bridge node added to /root.")
    _init_particles()
    set_process(true)

    # Headless Optimisations for Godot 4.x
    # Disable rendering subsystems completely
    RenderingServer.set_render_loop_enabled(false)
    RenderingServer.set_debug_generate_wireframes(false)

    # Disable audio system
    AudioServer.set_bus_count(0)

    # Configure physics for headless efficiency
    PhysicsServer2D.set_active(false)  # Disable if not using 2D physics
    PhysicsServer3D.set_active(false)  # Disable if not using 3D physics

    # If using physics, optimize parameters:
    if PhysicsServer2D.is_active() or PhysicsServer3D.is_active():
        Engine.set_physics_ticks_per_second(30)  # Reduce physics FPS
        PhysicsServer2D.set_collision_iterations(8)  # Reduce quality
        PhysicsServer3D.set_collision_iterations(8)

    # Reduce main loop processing
    Engine.set_max_fps(10)  # Limit FPS for CPU savings
    get_tree().set_auto_accept_quit(false)

    # Memory management alternatives (since unload_unused_resources is removed)
    var cleaner = Timer.new()
    cleaner.wait_time = 2.0  # Clean every 2 seconds
    cleaner.timeout.connect(func():
        for res in ResourceCache.get_cached_resources():
            if res.get_reference_count() == 0:
                res.take_over_path("")  # Make unique
                ResourceCache.remove_resource(res)
    )
    add_child(cleaner)
    cleaner.start()

    # Additional optimisations
    OS.low_processor_usage_mode = true  # Reduce CPU pressure
    ProjectSettings.set_setting("physics/common/enable_pause_aware_picking", false)

    # Orchestrate bridge after both nodes are present
    var bridge = get_node("/root/nexus_bridge")
    if bridge and bridge.has_method("init_bridge"):
        print("[main] Calling init_bridge() on nexus_bridge...")
        bridge.init_bridge()
        print("[main] Called init_bridge() on nexus_bridge.")
    else:
        print("[main] nexus_bridge missing or no init_bridge method!")

    # Trigger multi-campaign connection for Godot backend
    var nexus_client = get_node("/root/NexusClient")
    var campaign_ids = ["0"] # Extend this list for multi-campaign orchestration
    var user_id = "godot"
    if nexus_client:
        print("[main] NexusClient found, attempting campaign connections...")
        if nexus_client.has_method("connect_to_campaign"):
            for campaign_id in campaign_ids:
                print("[main] Connecting to campaign:", campaign_id, "as user:", user_id)
                nexus_client.connect_to_campaign(campaign_id, user_id)
                print("[main] Triggered connect_to_campaign for", campaign_id, "and user", user_id)
        else:
            print("[main] NexusClient missing connect_to_campaign method!")
    else:
        print("[main] NexusClient not found for campaign connection!")

    # Initialize Physics Engines (autoloads)
    if Engine.has_singleton("PhysicsEngine2D"):
        Engine.get_singleton("PhysicsEngine2D").init()
        print("[main] PhysicsEngine2D initialized.")
    else:
        print("[main] PhysicsEngine2D autoload not found!")
    if Engine.has_singleton("PhysicsEngine3D"):
        Engine.get_singleton("PhysicsEngine3D").init()
        print("[main] PhysicsEngine3D initialized.")
    else:
        print("[main] PhysicsEngine3D autoload not found!")

    if nexus_client:
        nexus_client.connect("campaign_connected", Callable(self, "_on_campaign_connected"))

func _init_particles():
    particles.clear()
    for i in range(particle_count):
        var angle = randf() * TAU
        var radius = 2.0 + randf() * 3.0
        var phase = randf() * TAU
        var color = Color.from_hsv(randf(), 0.8, 1.0)
        var intensity = 0.7 + randf() * 0.3
        particles.append({
            "x": cos(angle) * radius,
            "y": sin(angle) * radius,
            "z": sin(phase) * radius * 0.5,
            "vx": 0.0,
            "vy": 0.0,
            "vz": 0.0,
            "phase": phase,
            "intensity": intensity,
            "color": color,
            "id": i
        })

func _process(delta):
    tick += 1
    buffer.resize(particle_count * 10)
    for i in range(particle_count):
        var p = particles[i]
        # Magical, whimsical movement: spiral, wave, sparkle
        var t = tick * 0.03 + p["phase"]
        p["x"] = cos(t + i * 0.05) * (2.5 + sin(t * 0.7 + i * 0.1))
        p["y"] = sin(t + i * 0.07) * (2.5 + cos(t * 0.5 + i * 0.13))
        p["z"] = sin(t * 1.2 + i * 0.09) * (1.5 + cos(t * 0.3 + i * 0.11))
        p["vx"] = cos(t + i * 0.02) * 0.1 * sin(t * 0.5)
        p["vy"] = sin(t + i * 0.03) * 0.1 * cos(t * 0.7)
        p["vz"] = sin(t + i * 0.04) * 0.1 * sin(t * 0.9)
        p["intensity"] = 0.7 + abs(sin(t + i * 0.1)) * 0.3
        # Pack buffer: [x, y, z, vx, vy, vz, phase, intensity, color.to_rgba(), id]
        buffer[i * 10 + 0] = p["x"]
        buffer[i * 10 + 1] = p["y"]
        buffer[i * 10 + 2] = p["z"]
        buffer[i * 10 + 3] = p["vx"]
        buffer[i * 10 + 4] = p["vy"]
        buffer[i * 10 + 5] = p["vz"]
        buffer[i * 10 + 6] = p["phase"]
        buffer[i * 10 + 7] = p["intensity"]
        buffer[i * 10 + 8] = p["color"].to_rgba32()
        buffer[i * 10 + 9] = float(p["id"])
    # Emit buffer to WASM/Go or ws_server for frontend

func _on_campaign_connected(campaign_id):
    stream_particle_buffer(campaign_id)

func stream_particle_buffer(campaign_id):
    var nexus_client = get_node_or_null("/root/NexusClient")
    if nexus_client and nexus_client.has_method("send_event"):
        var event_dict = {
            "type": "particle:update:v1:success",
            "payload": {"buffer": buffer},
            "metadata": {"campaign_id": campaign_id, "user_id": "godot", "entity_type": "backend", "client_type": "godot"}
        }
        print("[main] Streaming particle buffer to campaign:", campaign_id)
        nexus_client.send_event(campaign_id, event_dict)
        print("[main] Streamed particle buffer to campaign.")
    else:
        print("[main] NexusClient missing send_event method or not found!")
