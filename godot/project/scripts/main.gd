extends Node

# Helper function to create properly formatted CanonicalEventEnvelope
func create_canonical_event(event_type: String, campaign_id: String, user_id: String = "godot", payload_data: Dictionary = {}) -> Dictionary:
    var correlation_id = "godot_" + event_type.replace(":", "_") + "_" + str(int(Time.get_unix_time_from_system()))
    return {
        "type": event_type,
        "correlation_id": correlation_id,
        "timestamp": Time.get_datetime_string_from_system(true, true).replace(" ", "T") + "Z",
        "version": "1.0.0",
        "environment": "development",
        "source": "backend",
        "payload": payload_data,
        "metadata": {
            "global_context": {
                "user_id": user_id,
                "campaign_id": campaign_id,
                "correlation_id": correlation_id,
                "session_id": "godot_session_" + str(int(Time.get_unix_time_from_system())),
                "device_id": "godot_device_" + str(int(Time.get_unix_time_from_system())),
                "source": "backend"
            },
            "envelope_version": "1.0.0",
            "environment": "development"
        }
    }

var particle_count := 100000  # 100k particles for better WebSocket compatibility
var particles := []
var tick := 0
var buffer : PackedFloat32Array = PackedFloat32Array()
var log_enabled : bool = true

# LOD (Level of Detail) settings
var lod_levels := [100000, 50000, 25000, 10000, 5000, 1000]  # Different LOD levels
var current_lod := 0  # Current LOD level (0 = highest detail)
var lod_distance_threshold := 10.0  # Distance threshold for LOD changes
var performance_target_fps := 60.0  # Target FPS for performance-based LOD

func _ready():
    print("[main] Godot headless main node ready.")
    
    # Initialize particles first
    _init_particles()
    set_process(true)
    
    # Setup nodes after initialization
    call_deferred("_setup_nodes")
    call_deferred("_setup_audio")
    call_deferred("_setup_optimizations")
    call_deferred("_setup_connections")

func _setup_nodes():
    print("[main] Setting up nodes...")
    
    # Ensure NexusClient (WebSocket client) is present in /root for orchestration
    if not has_node("/root/NexusClient"):
        print("[main] NexusClient not found, instancing...")
        var nexus_client = Node.new()
        var script = load("res://scripts/ws_client.gd")
        if script:
            nexus_client.set_script(script)
        else:
            print("[main] Failed to load ws_client.gd script")
            return
        nexus_client.name = "NexusClient"
        get_tree().get_root().add_child(nexus_client)
        print("[main] NexusClient node added to /root.")
    else:
        print("[main] NexusClient already present in /root.")

    # Add nexus_bridge after NexusClient
    if not has_node("/root/nexus_bridge"):
        print("[main] nexus_bridge not found, instancing...")
        var nexus_bridge = Node.new()
        var bridge_script = load("res://scripts/nexus_bridge.gd")
        if bridge_script:
            nexus_bridge.set_script(bridge_script)
        else:
            print("[main] Failed to load nexus_bridge.gd script")
            return
        nexus_bridge.name = "nexus_bridge"
        get_tree().get_root().add_child(nexus_bridge)
        print("[main] nexus_bridge node added to /root.")
    
    print("[main] Node setup complete.")

func _setup_audio():
    print("[main] Setting up audio...")
    # Disable audio system (set to minimum instead of 0 to avoid errors)
    AudioServer.set_bus_count(1)
    AudioServer.set_bus_mute(0, true)  # Mute the default bus
    print("[main] Audio setup complete.")

func _setup_optimizations():
    print("[main] Setting up headless optimizations...")
    
    # === RENDERING OPTIMIZATIONS ===
    # Completely disable rendering for headless operation
    RenderingServer.set_render_loop_enabled(false)
    RenderingServer.set_debug_generate_wireframes(false)
    # VSync is not available in headless mode
    
    # Disable unnecessary rendering features
    # Note: set_use_occlusion_culling methods are not available in Godot 4.4.1
    
    # === PHYSICS OPTIMIZATIONS ===
    # Balanced physics for multiple concurrent environments
    Engine.set_physics_ticks_per_second(60)  # Keep high for multiple environments
    Engine.set_max_fps(30)  # Balanced FPS for streaming multiple environments
    
    # Disable physics features not needed for headless
    ProjectSettings.set_setting("physics/common/enable_pause_aware_picking", false)
    ProjectSettings.set_setting("physics/2d/run_on_separate_thread", false)
    ProjectSettings.set_setting("physics/3d/run_on_separate_thread", false)
    
    # === MEMORY OPTIMIZATIONS ===
    # Aggressive memory management for headless
    var cleaner = Timer.new()
    cleaner.wait_time = 1.0  # More frequent cleanup (every 1 second)
    cleaner.timeout.connect(func():
        # Force garbage collection more aggressively
        OS.request_permission("notifications")  # Triggers cleanup
        RenderingServer.call_deferred("free_rid", RID())
        # Force garbage collection
        OS.delay_msec(1)  # Small delay to allow GC
    )
    add_child(cleaner)
    cleaner.start()
    
    # === SMART CPU MANAGEMENT FOR MULTIPLE ENVIRONMENTS ===
    # Intelligent CPU usage that adapts to concurrent environments
    OS.low_processor_usage_mode = false  # Allow full CPU usage for multiple environments
    OS.set_thread_name("GodotMultiEnv")  # Set thread name for debugging
    
    # Enable features needed for multiple environments
    get_tree().set_auto_accept_quit(false)
    get_tree().set_pause(false)  # Ensure not paused
    
    # CPU usage monitoring and adaptation
    var cpu_monitor = Timer.new()
    cpu_monitor.wait_time = 5.0  # Check CPU every 5 seconds
    cpu_monitor.timeout.connect(func():
        var current_fps = Engine.get_frames_per_second()
        var target_fps = performance_target_fps
        
        # Adaptive FPS based on performance
        if current_fps < target_fps * 0.8:  # If FPS is below 80% of target
            # Reduce quality to maintain performance
            if current_lod < lod_levels.size() - 1:
                current_lod += 1
                print("[main] Increased LOD to level ", current_lod, " due to low FPS: ", current_fps)
        elif current_fps > target_fps * 1.2 and current_lod > 0:  # If FPS is above 120% of target
            # Increase quality if performance allows
            current_lod -= 1
            print("[main] Decreased LOD to level ", current_lod, " due to good FPS: ", current_fps)
    )
    add_child(cpu_monitor)
    cpu_monitor.start()
    
    # === NETWORK OPTIMIZATIONS ===
    # Optimize for WebSocket connections
    ProjectSettings.set_setting("network/limits/tcp/connect_timeout_seconds", 5)
    ProjectSettings.set_setting("network/limits/tcp/read_timeout_seconds", 10)
    
    # === PARTICLE SYSTEM OPTIMIZATIONS ===
    # Optimized for multiple concurrent environments
    particle_count = min(particle_count, 100000)  # Keep high particle count for quality
    current_lod = 0  # Start at highest LOD for quality
    performance_target_fps = 60.0  # High target FPS for smooth streaming
    
    # === DOCKER-SPECIFIC OPTIMIZATIONS ===
    # Optimize for containerized headless operation
    OS.set_thread_name("GodotHeadless")
    
    # Disable features not needed in containers
    ProjectSettings.set_setting("application/run/main_scene", "")
    ProjectSettings.set_setting("display/window/size/resizable", false)
    ProjectSettings.set_setting("display/window/size/borderless", true)
    
    # === LOGGING OPTIMIZATIONS ===
    # Reduce log verbosity for containerized operation
    if OS.has_feature("release") or OS.get_environment("GODOT_HEADLESS_MODE") == "true":
        log_enabled = false
    
    # === SMART MEMORY MANAGEMENT FOR MULTIPLE ENVIRONMENTS ===
    # Intelligent memory management that adapts to concurrent environments
    var memory_monitor = Timer.new()
    memory_monitor.wait_time = 2.0  # Check memory every 2 seconds
    memory_monitor.timeout.connect(func():
        var memory_usage = OS.get_static_memory_usage()
        var memory_mb = memory_usage / (1024 * 1024)
        
        # Adaptive memory management based on usage
        if memory_mb > 500:  # If over 500MB (high threshold for multiple environments)
            # Force garbage collection
            OS.delay_msec(5)
            # Gradually reduce particle count if memory is very high
            if particle_count > 50000:
                particle_count = max(50000, particle_count - 10000)
                print("[main] Reduced particle count to ", particle_count, " due to high memory usage: ", memory_mb, "MB")
        elif memory_mb < 200 and particle_count < 100000:  # If memory is low, increase quality
            particle_count = min(100000, particle_count + 5000)
            print("[main] Increased particle count to ", particle_count, " due to available memory: ", memory_mb, "MB")
    )
    add_child(memory_monitor)
    memory_monitor.start()
    
    print("[main] Multi-environment headless optimizations complete - FPS: 30, Physics: 60Hz, Particles: ", particle_count)

func _setup_connections():
    print("[main] Setting up connections...")
    # Wait for nodes to be ready before setting up connections
    await get_tree().process_frame
    setup_connections()

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

func setup_connections():
    print("[main] Setting up connections...")
    
    # Wait for nodes to be ready
    await get_tree().process_frame
    
    # Trigger multi-campaign connection for Godot backend first
    var nexus_client = get_node_or_null("/root/NexusClient")
    
    # Support for multiple concurrent environments
    var campaign_ids = []
    var max_environments = 5
    if OS.get_environment("MAX_CONCURRENT_ENVIRONMENTS") != "":
        max_environments = int(OS.get_environment("MAX_CONCURRENT_ENVIRONMENTS"))
    
    # Generate campaign IDs for multiple environments
    for i in range(max_environments):
        campaign_ids.append(str(i))
    
    var user_id = "godot"
    if nexus_client:
        print("[main] NexusClient found, attempting connections to ", max_environments, " environments...")
        if nexus_client.has_method("connect_to_campaign"):
            for campaign_id in campaign_ids:
                print("[main] Connecting to environment:", campaign_id, "as user:", user_id)
                nexus_client.connect_to_campaign(campaign_id, user_id)
                # Add small delay between connections to avoid overwhelming
                await get_tree().create_timer(0.1).timeout
        else:
            print("[main] NexusClient missing connect_to_campaign method!")
        
        # Connect to campaign_connected signal
        if not nexus_client.is_connected("campaign_connected", Callable(self, "_on_campaign_connected")):
            nexus_client.connect("campaign_connected", Callable(self, "_on_campaign_connected"))
    else:
        print("[main] NexusClient not found for campaign connection!")

    # Wait a bit for connections to establish
    await get_tree().create_timer(1.0).timeout
    
    # Now orchestrate bridge after connections are established
    var bridge = get_node_or_null("/root/nexus_bridge")
    if bridge and bridge.has_method("init_bridge"):
        print("[main] Calling init_bridge() on nexus_bridge...")
        bridge.init_bridge()
        print("[main] Called init_bridge() on nexus_bridge.")
    else:
        print("[main] nexus_bridge missing or no init_bridge method!")

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

func _process(delta):
    # Update LOD based on performance
    update_lod_based_on_performance()
    
    tick += 1
    var effective_particle_count = lod_levels[current_lod]
    buffer.resize(effective_particle_count * 10)
    
    # Smart processing for multiple concurrent environments
    var headless_mode = OS.get_environment("GODOT_HEADLESS_MODE") == "true"
    var time_factor = 0.03  # Consistent timing for quality
    var batch_size = 2000  # Large batches for efficiency
    var skip_frames = 0  # No frame skipping for quality
    
    # Process particles in efficient batches
    for batch_start in range(0, effective_particle_count, batch_size):
        var batch_end = min(batch_start + batch_size, effective_particle_count)
        
        # Pre-calculate common values for the batch
        var batch_time = tick * time_factor
        var batch_sin = sin(batch_time)
        var batch_cos = cos(batch_time)
        
        for i in range(batch_start, batch_end):
            var p = particles[i]
            var t = batch_time + p["phase"]
            
            # Optimized movement calculations
            var i_factor = i * 0.05
            var sin_t = sin(t + i_factor)
            var cos_t = cos(t + i_factor)
            
            p["x"] = cos_t * (2.5 + sin(t * 0.7 + i * 0.1))
            p["y"] = sin_t * (2.5 + cos(t * 0.5 + i * 0.13))
            p["z"] = sin(t * 1.2 + i * 0.09) * (1.5 + cos(t * 0.3 + i * 0.11))
            
            # Calculate velocity only when needed (every 3rd frame for efficiency)
            if tick % 3 == 0:
                p["vx"] = cos(t + i * 0.02) * 0.1 * sin(t * 0.5)
                p["vy"] = sin(t + i * 0.03) * 0.1 * cos(t * 0.7)
                p["vz"] = sin(t + i * 0.04) * 0.1 * sin(t * 0.9)
            
            p["intensity"] = 0.7 + abs(sin(t + i * 0.1)) * 0.3
            
            # Pack buffer efficiently: [x, y, z, vx, vy, vz, phase, intensity, color.to_rgba(), id]
            var base_idx = i * 10
            buffer[base_idx + 0] = p["x"]
            buffer[base_idx + 1] = p["y"]
            buffer[base_idx + 2] = p["z"]
            buffer[base_idx + 3] = p["vx"]
            buffer[base_idx + 4] = p["vy"]
            buffer[base_idx + 5] = p["vz"]
            buffer[base_idx + 6] = p["phase"]
            buffer[base_idx + 7] = p["intensity"]
            buffer[base_idx + 8] = p["color"].to_rgba32()
            buffer[base_idx + 9] = float(p["id"])
    
    # Emit buffer to WASM/Go or ws_server for frontend

# Update LOD based on performance
func update_lod_based_on_performance():
    var current_fps = Engine.get_frames_per_second()
    
    # If FPS is below target, reduce LOD
    if current_fps < performance_target_fps and current_lod < lod_levels.size() - 1:
        current_lod += 1
        print("[main] Reducing LOD to level ", current_lod, " (", lod_levels[current_lod], " particles) due to low FPS: ", current_fps)
    # If FPS is good and we're not at highest LOD, increase LOD
    elif current_fps > performance_target_fps + 10 and current_lod > 0:
        current_lod -= 1
        print("[main] Increasing LOD to level ", current_lod, " (", lod_levels[current_lod], " particles) due to good FPS: ", current_fps)

# Set LOD level manually
func set_lod_level(level: int):
    if level >= 0 and level < lod_levels.size():
        current_lod = level
        print("[main] LOD set to level ", level, " (", lod_levels[level], " particles)")

# Get current effective particle count
func get_effective_particle_count() -> int:
    return lod_levels[current_lod]

func _on_campaign_connected(campaign_id):
    print("[main] Environment connected:", campaign_id)
    # Start streaming to this environment
    stream_particle_buffer(campaign_id)
    
    # Set up periodic streaming for this environment
    var stream_timer = Timer.new()
    stream_timer.wait_time = 0.1  # Stream every 100ms for smooth updates
    stream_timer.timeout.connect(func(): stream_particle_buffer(campaign_id))
    stream_timer.name = "stream_timer_" + campaign_id
    add_child(stream_timer)
    stream_timer.start()

func stream_particle_buffer(campaign_id):
    var nexus_client = get_node_or_null("/root/NexusClient")
    if nexus_client and nexus_client.has_method("send_event"):
        var effective_count = get_effective_particle_count()
        var buffer_size = buffer.size() * 4  # 4 bytes per float
        
        # Only log occasionally to avoid spam
        if int(Time.get_unix_time_from_system()) % 10 == 0:
            print("[main] Streaming to env:", campaign_id, " (", buffer_size, " bytes, ", effective_count, " particles, LOD:", current_lod, ")")
        
        var event_dict = create_canonical_event(
            "physics:particle:batch",
            campaign_id,
            "godot",
            {
                "buffer": buffer,
                "particle_count": effective_count,
                "lod_level": current_lod,
                "max_particles": particle_count,
                "compression": "delta",
                "format": "10_values_per_particle",  # x,y,z,vx,vy,vz,phase,intensity,type,id
                "source": "godot_physics",
                "timestamp": Time.get_unix_time_from_system(),
                "environment_id": campaign_id
            }
        )
        
        # Use large event for particle buffers to enable chunking
        if nexus_client.has_method("send_large_event"):
            nexus_client.send_large_event(campaign_id, event_dict)
        else:
            nexus_client.send_event(campaign_id, event_dict)
    else:
        if int(Time.get_unix_time_from_system()) % 30 == 0:  # Log error every 30 seconds
            print("[main] NexusClient missing send_event method or not found!")
