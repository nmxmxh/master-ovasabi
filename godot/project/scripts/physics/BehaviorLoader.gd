extends Node
class_name BehaviorLoader

var _behavior_cache := {}

func load_behavior(behavior_name: String) -> GDScript:
    if _behavior_cache.has(behavior_name):
        return _behavior_cache[behavior_name]
    var behavior_path = "res://scripts/physics/behaviors/%s.gd" % behavior_name
    if ResourceLoader.exists(behavior_path):
        var script = load(behavior_path)
        _behavior_cache[behavior_name] = script
        return script
    match behavior_name:
        "aerodynamic":
            return load("res://scripts/physics/behaviors/base/aerodynamic.gd")
        _:
            push_error("Behavior not found: " + behavior_name)
            return null

func create_behavior_instance(behavior_name: String, params: Dictionary) -> Node:
    var script = load_behavior(behavior_name)
    if script:
        var instance = script.new()
        instance.set_parameters(params)
        return instance
    return null
