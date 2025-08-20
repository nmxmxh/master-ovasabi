# OVASABI Service Registry Loader for Godot
# This script provides an architecture-aware interface to the OVASABI service registry.
# Place in godot/project/scripts/services_loader.gd

extends Node

# Dictionary to hold all service definitions
var services = {}

# Loads and parses the services.godot config file
func _ready():
    print("[services_loader] _ready() called.")
    load_services_config("res://services.godot")
    print("[services_loader] Loaded OVASABI service architecture:")
    for name in services.keys():
        print_service_info(name)

# Load and parse the config file
func load_services_config(path):
    print("[services_loader] Loading services config from: %s" % path)
    var config = ConfigFile.new()
    var err = config.load(path)
    if err != OK:
        push_error("[services_loader] Failed to load services config: %s" % path)
        return
    var section = "services"
    for key in config.get_section_keys(section):
        var svc = config.get_value(section, key)
        services[key] = svc
        print("[services_loader] Loaded service: %s" % key)

# Print detailed info for a service
func print_service_info(name):
    print("[services_loader] print_service_info() called for: %s" % name)
    var svc = services.get(name, null)
    if svc == null:
        print("[services_loader] Service not found: %s" % name)
        return
    print("[services_loader] Service: %s" % name)
    print("  Version: %s" % svc.version)
    print("  Health Check: %s" % svc.health_check)
    print("  Metrics: %s" % svc.metrics)
    print("  Capabilities: %s" % str(svc.capabilities))
    print("  Dependencies: %s" % str(svc.dependencies))

# Query services by capability
func get_services_with_capability(capability):
    var result = []
    for name in services.keys():
        if capability in services[name].capabilities:
            result.append(name)
    return result

# Get dependencies for a service
func get_service_dependencies(name):
    var svc = services.get(name, null)
    if svc == null:
        return []
    return svc.dependencies

# Get health check path for a service
func get_health_check_path(name):
    var svc = services.get(name, null)
    if svc == null:
        return ""
    return svc.health_check

# Get metrics path for a service
func get_metrics_path(name):
    var svc = services.get(name, null)
    if svc == null:
        return ""
    return svc.metrics

# Get all capabilities in the architecture
func get_all_capabilities():
    var caps = {}
    for name in services.keys():
        for cap in services[name].capabilities:
            caps[cap] = true
    return caps.keys()

# Get all services and their dependencies as a graph
func get_service_graph():
    var graph = {}
    for name in services.keys():
        graph[name] = services[name].dependencies
    return graph

# Example: Simulate health check for all services
func simulate_health_checks():
    for name in services.keys():
        var health_path = get_health_check_path(name)
        print("Simulating health check for %s at %s" % [name, health_path])

# Example: Print all services with a given capability
func print_services_with_capability(cap):
    var svc_list = get_services_with_capability(cap)
    print("Services with capability '%s': %s" % [cap, str(svc_list)])
