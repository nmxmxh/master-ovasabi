# OVASABI Campaign Dashboard Controller
# Handles UI stimulation for campaign state and health orchestration
# Place in godot/project/scripts/CampaignDashboard.gd

extends Control

@onready var state_label = $CampaignStateLabel
@onready var health_list = $HealthStatusList
var ws_client
var WSClient = preload("res://scripts/ws_client.gd")

func _ready():
    ws_client = WSClient.new()
    add_child(ws_client)
    ws_client.connect("campaign_state_updated", self, "_on_campaign_state_updated")
    ws_client.connect("health_status_updated", self, "_on_health_status_updated")
    ws_client.connect("connection_error", self, "_on_connection_error")
    ws_client.connect("connected", self, "_on_connected")
    ws_client.connect("disconnected", self, "_on_disconnected")
    ws_client.connect("connected", self, "_run_startup_test")

func _run_startup_test():
    state_label.text = "[TEST] Running startup test..."
    # Send a campaign action
    ws_client.send_campaign_action("test_action", {"test_key": "test_value"})
    # Request health status for 'nexus' service
    ws_client.request_health_status("nexus")
    print("[TEST] Sent campaign action and health request on startup.")

func _on_campaign_state_updated(state):
    state_label.text = "Campaign State: %s" % str(state)

func _on_health_status_updated(service, status):
    var idx = health_list.get_item_index(service)
    if idx == -1:
        health_list.add_item("%s: %s" % [service, status])
    else:
        health_list.set_item_text(idx, "%s: %s" % [service, status])

func _on_connection_error(error):
    state_label.text = "Connection Error: %s" % error

func _on_connected():
    state_label.text = "Connected to Gateway"

func _on_disconnected():
    state_label.text = "Disconnected from Gateway"

func _request_all_health():
    var services = ["admin", "ai", "analytics", "campaign", "commerce", "content", "contentmoderation", "crawler", "localization", "media", "messaging", "nexus", "centralized_health", "notification", "product", "referral", "scheduler", "search", "security", "talent", "user", "waitlist"]
    for svc in services:
        ws_client.request_health_status(svc)

# Example: Send campaign action from UI
func send_campaign_action(action, data):
    ws_client.send_campaign_action(action, data)
