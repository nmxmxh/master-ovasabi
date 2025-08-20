
extends Node
# Helper: Build backend-compatible metadata structure for campaign events
func build_campaign_metadata(campaign_id: String, user_id: String = "godot", campaign_slug: String = "") -> Dictionary:
    var metadata = {
        "service_specific": {
            "campaign": {
                "campaign_id": campaign_id,
                "slug": campaign_slug
            },
            "global": {
                "user_id": "godot",
                "entity_type": "backend",
                "client_type": "godot"
            }
        }
    }
    return metadata

var particle_buffer = PackedFloat32Array()
var nexus_client = null
var campaign_state = {}
var active_campaigns := {} # campaign_id -> state

# Called when the node enters the scene tree for the first time
func _ready():
    print("[nexus_bridge] _ready() called.")

# Orchestration logic moved to init_bridge()
func init_bridge():
    nexus_client = get_node_or_null("/root/NexusClient")
    var campaign_ids = ["0"] # Extend this list for multi-campaign orchestration
    if nexus_client:
        print("[nexus_bridge] NexusClient found, requesting campaign state for:", campaign_ids)
        for campaign_id in campaign_ids:
            var event = {
                "type": "campaign:state:v1:request",
                "payload": {},
                "metadata": build_campaign_metadata(campaign_id)
            }
            if nexus_client.has_method("send_event"):
                nexus_client.send_event(campaign_id, event)
            elif nexus_client.has_method("emit_event"):
                nexus_client.emit_event(event)
        print("[nexus_bridge] Subscribing to event_received signal.")
        nexus_client.connect("event_received", Callable(self, "_on_nexus_event"))
    else:
        print("[nexus_bridge] NexusClient NOT found!")

func _on_nexus_event(campaign_id, event):
    print("[nexus_bridge] Event received for campaign %s: %s" % [campaign_id, str(event)])
    if not is_canonical_event_type(event.type):
        print("[nexus_bridge] Invalid event type: %s" % event.type)
        return
    if event.type.begins_with("campaign:state"):
        active_campaigns[campaign_id] = event.payload
        campaign_state = event.payload # For backward compatibility
        print("[nexus_bridge] Campaign state updated for %s: %s" % [campaign_id, str(event.payload)])
    elif event.type.begins_with("particle:update"):
        _update_particle_buffer(event.payload)
        print("[nexus_bridge] Particle buffer updated from event.")

func _update_particle_buffer(payload):
    print("[nexus_bridge] Updating particle buffer with payload: %s" % str(payload))
    if payload.has("buffer"):
        particle_buffer = payload["buffer"]
        print("[nexus_bridge] Particle buffer updated.")
    else:
        print("[nexus_bridge] No buffer found in payload.")

func get_particle_buffer():
    print("[nexus_bridge] get_particle_buffer() called.")
    return particle_buffer

func forward_particle_buffer(campaign_ids = []):
    print("[nexus_bridge] forward_particle_buffer() called.")
    if campaign_ids.size() == 0:
        campaign_ids = active_campaigns.keys()
    if nexus_client:
        for campaign_id in campaign_ids:
            var event = {
                "type": "particle:update:v1:success",
                "payload": {"buffer": particle_buffer},
                "metadata": build_campaign_metadata(campaign_id)
            }
            if nexus_client.has_method("send_event"):
                nexus_client.send_event(campaign_id, event)
            elif nexus_client.has_method("emit_event"):
                nexus_client.emit_event(event)
            print("[nexus_bridge] Forwarded particle buffer to campaign %s." % campaign_id)
    else:
        print("[nexus_bridge] NexusClient not available for forwarding.")

func _get_buffer_for_js():
    print("[nexus_bridge] _get_buffer_for_js() called.")
    return get_particle_buffer()

func is_canonical_event_type(event_type):
    print("[nexus_bridge] is_canonical_event_type() called for: %s" % event_type)
    var parts = event_type.split(":")
    var result = parts.size() == 4 and parts[2].begins_with("v")
    print("[nexus_bridge] Canonical event type: %s" % str(result))
    return result

func handle_event(event, client_id):
    print("[nexus_bridge] handle_event() called for event: %s, client_id: %s" % [str(event), str(client_id)])
    if not is_canonical_event_type(event.type):
        print("[nexus_bridge] Invalid event type: %s" % event.type)
        return
    # Attach metadata, handle request/response, forward to backend if needed
    pass
