extends Node

# Signals
signal campaign_connected(campaign_id)
signal campaign_disconnected(campaign_id)
signal event_received(campaign_id, event)

# Multi-campaign WebSocket client for Godot 4.x - Headless Optimized
var connections := {} # campaign_id -> {peer, connected, ...}
var max_reconnect_attempts : int = 3  # Reduced for headless
var reconnect_delay : int = 2000  # Increased delay for headless
var log_enabled : bool = true
var headless_mode : bool = false

# Called when the node enters the scene tree
func _ready():
    # Detect headless mode
    headless_mode = OS.has_feature("headless") or OS.get_environment("GODOT_HEADLESS_MODE") == "true"
    
    if log_enabled and not headless_mode:
        print("[ws_client] Ready. Initializing connection pool.")
    connections = {}
    
    # Multi-environment optimizations
    if headless_mode:
        # Optimized for multiple concurrent environments
        reconnect_delay = 3000  # 3 seconds for multiple environments
        max_reconnect_attempts = 5  # More attempts for reliability

# Connect to a campaign
func connect_to_campaign(campaign_id: String, user_id: String = "", ws_url: String = ""):
    if log_enabled:
        print("[ws_client] Attempting to connect to campaign:", campaign_id, "as user:", user_id)
    
    if connections.has(campaign_id):
        var state = connections[campaign_id]["peer"].get_ready_state()
        if state == WebSocketPeer.STATE_OPEN || state == WebSocketPeer.STATE_CONNECTING:
            if log_enabled:
                print("[ws_client] Already connected/connecting to campaign:", campaign_id)
            return
        else:
            # Clean up old connection if not active
            if log_enabled:
                print("[ws_client] Cleaning up old connection for campaign:", campaign_id)
            connections.erase(campaign_id)
    
    var url = ws_url if !ws_url.is_empty() else get_ws_url(campaign_id, user_id)
    if log_enabled:
        print("[ws_client] WebSocket URL:", url)
    
    var ws_peer = WebSocketPeer.new()
    
    # Configure WebSocket settings for multiple concurrent environments
    ws_peer.set_max_queued_packets(1024)  # Higher queue for multiple environments
    ws_peer.set_handshake_headers([
        "User-Agent: GodotMultiEnv/4.x",
        "X-Client-Type: headless",
        "X-Resource-Mode: concurrent",
        "X-Max-Environments: 10"
    ])
    # Note: WebSocket buffer size is controlled by the underlying wslay library
    # We'll handle large messages by chunking them if needed
    
    var err = ws_peer.connect_to_url(url)
    if log_enabled:
        print("[ws_client] Connecting to ", url, " for campaign ", campaign_id, " - Error: ", error_string(err))
    
    if err != OK:
        if log_enabled:
            push_error("[ws_client] Connection failed: " + error_string(err))
        return
    
    connections[campaign_id] = {
        "peer": ws_peer,
        "connected": false,
        "connecting": true,
        "reconnect_attempts": 0,
        "ws_url": url,
        "user_id": user_id,
        "last_activity": Time.get_unix_time_from_system()
    }
    
    if log_enabled:
        print("[ws_client] Connection state stored for campaign:", campaign_id)
    
    # Start connection monitoring
    _monitor_connection(campaign_id)

# Disconnect from a campaign
func disconnect_from_campaign(campaign_id: String):
    if !connections.has(campaign_id):
        return
    
    var ws_peer = connections[campaign_id]["peer"]
    if ws_peer.get_ready_state() == WebSocketPeer.STATE_OPEN:
        ws_peer.close()
    
    connections.erase(campaign_id)
    if log_enabled:
        print("[ws_client] Disconnected from campaign:", campaign_id)

# Send large data in chunks to avoid WebSocket buffer overflow
func send_large_event(campaign_id: String, event: Dictionary, chunk_size: int = 1024 * 1024):  # 1MB chunks for better compatibility
    var json_string = JSON.stringify(event)
    var data_size = json_string.length()
    
    if data_size <= chunk_size:
        # Small enough to send directly
        send_event(campaign_id, event)
        return
    
    if log_enabled:
        print("[ws_client] Large event detected (", data_size, " bytes), implementing chunking")
    
    # Implement proper chunking for large events
    send_chunked_event(campaign_id, event, chunk_size)

# Send chunked event for very large data
func send_chunked_event(campaign_id: String, event: Dictionary, chunk_size: int):
    if !connections.has(campaign_id):
        if log_enabled:
            push_error("[ws_client] No connection for campaign: " + campaign_id)
        return
    
    var conn = connections[campaign_id]
    var ws_peer = conn["peer"]
    var state = ws_peer.get_ready_state()
    
    if state != WebSocketPeer.STATE_OPEN:
        if log_enabled:
            push_error("[ws_client] Connection not open for campaign " + campaign_id + " (state: " + str(state) + ")")
        return
    
    # Create chunked event structure
    var correlation_id = event.get("correlation_id", "chunked_" + str(int(Time.get_unix_time_from_system())))
    var total_chunks = 0
    var chunk_data = []
    
    # Handle particle buffer data specially
    if event.has("payload") and event.payload.has("data") and event.payload.data.has("buffer"):
        var buffer_data = event.payload.data.buffer
        var buffer_size = buffer_data.size() * 4  # 4 bytes per float
        
        if log_enabled:
            print("[ws_client] Chunking particle buffer: ", buffer_size, " bytes")
        
        # Compress particle data for efficiency
        var compressed_data = compress_particle_buffer(buffer_data)
        var compressed_size = compressed_data.size()
        
        if log_enabled:
            print("[ws_client] Compressed particle data: ", buffer_size, " -> ", compressed_size, " bytes (", int((1.0 - float(compressed_size) / float(buffer_size)) * 100), "% reduction)")
        
        # Create binary chunks with compressed data
        var chunk_count = int(ceil(float(compressed_size) / float(chunk_size)))
        total_chunks = chunk_count
        
        for i in range(chunk_count):
            var start_idx = i * (chunk_size / 4)  # Convert to float indices
            var end_idx = min((i + 1) * (chunk_size / 4), compressed_data.size())
            
            var chunk = {
                "type": "physics:particle:chunk",
                "correlation_id": correlation_id,
                "timestamp": Time.get_datetime_string_from_system(true, true).replace(" ", "T") + "Z",
                "version": "1.0.0",
                "environment": "development",
                "source": "backend",
                "payload": {
                    "data": {
                        "chunk_index": i,
                        "total_chunks": chunk_count,
                        "buffer": compressed_data.slice(start_idx, end_idx),
                        "original_type": event.type,
                        "compressed": true,
                        "original_size": buffer_size,
                        "format": "10_values_per_particle"
                    }
                },
                "metadata": event.metadata
            }
            chunk_data.append(chunk)
    else:
        # Handle regular large events
        var json_string = JSON.stringify(event)
        var chunk_count = int(ceil(float(json_string.length()) / float(chunk_size)))
        total_chunks = chunk_count
        
        for i in range(chunk_count):
            var start_idx = i * chunk_size
            var end_idx = min((i + 1) * chunk_size, json_string.length())
            
            var chunk = {
                "type": "event:chunk:v1:data",
                "correlation_id": correlation_id,
                "timestamp": Time.get_datetime_string_from_system(true, true).replace(" ", "T") + "Z",
                "version": "1.0.0",
                "environment": "development",
                "source": "backend",
                "payload": {
                    "data": {
                        "chunk_index": i,
                        "total_chunks": chunk_count,
                        "chunk_data": json_string.substr(start_idx, end_idx - start_idx),
                        "original_type": event.type
                    }
                },
                "metadata": event.metadata
            }
            chunk_data.append(chunk)
    
    # Send all chunks
    for chunk in chunk_data:
        var msg = JSON.stringify(chunk)
        var err = ws_peer.send_text(msg)
        
        if err != OK:
            if log_enabled:
                push_error("[ws_client] Chunk send failed for campaign " + campaign_id + ": " + error_string(err))
            return
        elif log_enabled:
            print("[ws_client] Sent chunk ", chunk.payload.data.chunk_index + 1, "/", total_chunks, " to campaign ", campaign_id)
    
    if log_enabled:
        print("[ws_client] Successfully sent ", total_chunks, " chunks to campaign ", campaign_id)

# Send event to a specific campaign
# Monitor connection status
func _monitor_connection(campaign_id: String):
    if !connections.has(campaign_id):
        return
    
    var ws_peer = connections[campaign_id]["peer"]
    var state = ws_peer.get_ready_state()
    
    if state == WebSocketPeer.STATE_OPEN:
        if !connections[campaign_id]["connected"]:
            connections[campaign_id]["connected"] = true
            connections[campaign_id]["connecting"] = false
            if log_enabled:
                print("[ws_client] Connected to campaign:", campaign_id)
            campaign_connected.emit(campaign_id)
    elif state == WebSocketPeer.STATE_CLOSED:
        if connections[campaign_id]["connected"] or connections[campaign_id]["connecting"]:
            connections[campaign_id]["connected"] = false
            connections[campaign_id]["connecting"] = false
            if log_enabled:
                print("[ws_client] Disconnected from campaign:", campaign_id)
            campaign_disconnected.emit(campaign_id)
    
    # Schedule next check
    if connections.has(campaign_id):
        get_tree().create_timer(0.1).timeout.connect(func(): _monitor_connection(campaign_id))

func send_event(campaign_id: String, event: Dictionary):
    if !connections.has(campaign_id):
        if log_enabled:
            print("[ws_client] No connection for campaign:", campaign_id, "- attempting to reconnect...")
        # Try to reconnect
        var user_id = "godot"  # Default user ID
        connect_to_campaign(campaign_id, user_id)
        # Wait a bit for connection
        await get_tree().create_timer(1.0).timeout
        if !connections.has(campaign_id) || !connections[campaign_id]["connected"]:
            push_error("[ws_client] No connection for campaign: " + campaign_id)
            return
    
    var conn = connections[campaign_id]
    var ws_peer = conn["peer"]
    var state = ws_peer.get_ready_state()
    
    if state != WebSocketPeer.STATE_OPEN:
        if log_enabled:
            push_error("[ws_client] Connection not open for campaign " + campaign_id + " (state: " + str(state) + ")")
        return
    
    var msg = JSON.stringify(event)
    var msg_size = msg.length()
    
    # Check message size before sending - use large event for big messages
    if msg_size > 8 * 1024 * 1024:  # 8MB limit, use chunking for larger
        if log_enabled:
            print("[ws_client] Message large (", msg_size, " bytes), using chunked sending")
        send_large_event(campaign_id, event)
        return
    
    var err = ws_peer.send_text(msg)
    
    if err != OK:
        if log_enabled:
            push_error("[ws_client] Send failed for campaign " + campaign_id + ": " + error_string(err) + " (msg_size: " + str(msg_size) + " bytes)")
    elif log_enabled:
        print("[ws_client] Sent event to campaign ", campaign_id, ": ", msg)
        conn["last_activity"] = Time.get_unix_time_from_system()

# Process all connections
func _process(_delta):
    var to_remove = []
    
    for campaign_id in connections:
        var conn = connections[campaign_id]
        var ws_peer = conn["peer"]
        
        # Process connection
        ws_peer.poll()
        var state = ws_peer.get_ready_state()
        
        # State handling
        match state:
            WebSocketPeer.STATE_OPEN:
                if !conn["connected"]:
                    conn["connected"] = true
                    conn["connecting"] = false
                    conn["reconnect_attempts"] = 0
                    if log_enabled:
                        print("[ws_client] Connected to campaign:", campaign_id)
                    emit_signal("campaign_connected", campaign_id)
                    
                    # Send handshake with proper CanonicalEventEnvelope format
                    var handshake = create_canonical_event(
                        "campaign:state:v1:request",
                        campaign_id,
                        conn["user_id"],
                        {"data": {"client_type": "godot", "entity_type": "backend"}}
                    )
                    if log_enabled:
                        print("[ws_client] DEBUG: Sending handshake with version field:", handshake.has("version"))
                        print("[ws_client] DEBUG: Handshake event:", JSON.stringify(handshake))
                    send_event(campaign_id, handshake)
                
                # Process incoming packets
                while ws_peer.get_available_packet_count() > 0:
                    var packet = ws_peer.get_packet()
                    if packet:
                        var msg = packet.get_string_from_utf8()
                        if msg:
                            handle_event(campaign_id, msg)
                            conn["last_activity"] = Time.get_unix_time_from_system()
            
            WebSocketPeer.STATE_CONNECTING:
                # Connection in progress - just wait
                pass
            
            WebSocketPeer.STATE_CLOSED:
                handle_closed_connection(campaign_id, conn)
                to_remove.append(campaign_id)
            
            WebSocketPeer.STATE_CLOSING:
                # Wait for proper closure
                pass
    
    # Clean up closed connections
    for campaign_id in to_remove:
        connections.erase(campaign_id)

# Handle closed connections
func handle_closed_connection(campaign_id: String, conn: Dictionary):
    if conn["connected"]:
        conn["connected"] = false
        if log_enabled:
            print("[ws_client] Connection closed for campaign:", campaign_id)
    
    if conn["reconnect_attempts"] < max_reconnect_attempts:
        conn["reconnect_attempts"] += 1
        var delay = reconnect_delay * pow(2, conn["reconnect_attempts"] - 1)
        
        if log_enabled:
            print("[ws_client] Reconnecting to ", campaign_id, " in ", delay, "ms (attempt ", conn["reconnect_attempts"], ")")
        
        # Schedule reconnect
        await get_tree().create_timer(delay / 1000.0).timeout
        connect_to_campaign(campaign_id, conn["user_id"], conn["ws_url"])
    else:
        if log_enabled:
            print("[ws_client] Max reconnect attempts reached for campaign:", campaign_id)
        # Notify about permanent disconnection
        emit_signal("connection_failed", campaign_id)

# Handle incoming events
func handle_event(campaign_id: String, msg: String):
    var json = JSON.new()
    var error = json.parse(msg)
    
    if error != OK:
        if log_enabled:
            push_error("[ws_client] JSON parse error: " + json.get_error_message())
        return
    
    var event = json.get_data()
    if log_enabled:
        print("[ws_client] Received from ", campaign_id, ": ", event)
    
    emit_signal("event_received", campaign_id, event)

# Helper function to create properly formatted CanonicalEventEnvelope
func create_canonical_event(event_type: String, campaign_id: String, user_id: String = "godot", payload_data: Dictionary = {}) -> Dictionary:
    var correlation_id = "godot_" + event_type.replace(":", "_") + "_" + str(int(Time.get_unix_time_from_system()))
    var event = {
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
    if log_enabled:
        print("[ws_client] DEBUG: Created canonical event with version:", event.has("version"))
        print("[ws_client] DEBUG: Event structure:", JSON.stringify(event))
    return event

# Get WebSocket URL
func get_ws_url(campaign_id: String, user_id: String = "") -> String:
    var host = OS.get_environment("WS_GATEWAY_HOST")
    var port = OS.get_environment("WS_GATEWAY_PORT")
    var protocol = "wss" if OS.get_environment("WS_USE_SSL") == "true" else "ws"
    
    if host.is_empty(): host = "localhost"
    if port.is_empty(): port = "8090"
    
    var uid = user_id if !user_id.is_empty() else OS.get_environment("USER_ID")
    if uid.is_empty(): uid = "guest-%d" % randi()
    
    return "%s://%s:%s/ws/%s/%s" % [protocol, host, port, campaign_id, uid]

# Send campaign action
func send_campaign_action(campaign_id: String, action: String, data: Dictionary):
    if campaign_id.is_empty():
        push_error("[ws_client] Invalid campaign_id")
        return
    
    var event_dict = create_canonical_event(
        "campaign:action:%s" % action,
        campaign_id,
        "godot",
        data
    )
    send_event(campaign_id, event_dict)

# Compress particle buffer using simple delta compression
func compress_particle_buffer(buffer: PackedFloat32Array) -> PackedFloat32Array:
    if buffer.size() < 4:
        return buffer
    
    var compressed = PackedFloat32Array()
    compressed.append(buffer[0])  # First value as reference
    
    # Delta compression: store differences between consecutive values
    for i in range(1, buffer.size()):
        var delta = buffer[i] - buffer[i-1]
        compressed.append(delta)
    
    return compressed

# Decompress particle buffer (for receiving end)
func decompress_particle_buffer(compressed: PackedFloat32Array) -> PackedFloat32Array:
    if compressed.size() < 2:
        return compressed
    
    var decompressed = PackedFloat32Array()
    decompressed.append(compressed[0])  # First value as reference
    
    # Reconstruct original values from deltas
    for i in range(1, compressed.size()):
        var value = decompressed[i-1] + compressed[i]
        decompressed.append(value)
    
    return decompressed
# Signals
signal event_received(campaign_id, event)
signal connection_failed(campaign_id)
signal campaign_connected(campaign_id)

