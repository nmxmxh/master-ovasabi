extends Node

# Multi-campaign WebSocket client for Godot 4.x
var connections := {} # campaign_id -> {peer, connected, ...}
var max_reconnect_attempts : int = 5
var reconnect_delay : int = 1000  # milliseconds
var log_enabled : bool = true

# Called when the node enters the scene tree
func _ready():
    if log_enabled:
        print("[ws_client] Ready. Initializing connection pool.")
    connections = {}

# Connect to a campaign
func connect_to_campaign(campaign_id: String, user_id: String = "", ws_url: String = ""):
    if connections.has(campaign_id):
        var state = connections[campaign_id]["peer"].get_ready_state()
        if state == WebSocketPeer.STATE_OPEN || state == WebSocketPeer.STATE_CONNECTING:
            if log_enabled:
                print("[ws_client] Already connected/connecting to campaign:", campaign_id)
            return
        else:
            # Clean up old connection if not active
            connections.erase(campaign_id)
    
    var url = ws_url if !ws_url.is_empty() else get_ws_url(campaign_id, user_id)
    var ws_peer = WebSocketPeer.new()
    
    # Configure WebSocket settings
    ws_peer.set_max_queued_packets(1024)  # Prevent memory bloat
    ws_peer.set_handshake_headers(["User-Agent: GodotEngine/4.x"])
    
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

# Send event to a specific campaign
func send_event(campaign_id: String, event: Dictionary):
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
    
    var msg = JSON.stringify(event)
    var err = ws_peer.send_text(msg)
    
    if err != OK:
        if log_enabled:
            push_error("[ws_client] Send failed for campaign " + campaign_id + ": " + error_string(err))
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
                    
                    # Send handshake with minimal data field for backend compatibility
                    var handshake = {
                        "type": "campaign:state:v1:request",
                        "payload": {"data": {"client_type": "godot", "entity_type": "backend"}},
                        "metadata": {"campaign_id": campaign_id, "user_id": conn["user_id"]}
                    }
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
    
    var event_dict = {
        "type": "campaign:action:%s" % action,
        "payload": data,
        "metadata": {"campaign_id": campaign_id}
    }
    send_event(campaign_id, event_dict)

# Signals
signal event_received(campaign_id, event)
signal connection_failed(campaign_id)
signal campaign_connected(campaign_id)
