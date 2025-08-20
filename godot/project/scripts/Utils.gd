extends Node

# General-purpose helpers for nested Dictionary property access
# Usage: Utils.has_nested_property(dict, "a/b/c")
#        Utils.get_nested_property(dict, "a/b/c")

static func has_nested_property(dict: Dictionary, property_path: String) -> bool:
    var keys = property_path.split("/")
    var current = dict
    for key in keys:
        if typeof(current) != TYPE_DICTIONARY or not current.has(key):
            return false
        current = current[key]
    return true

static func get_nested_property(dict: Dictionary, property_path: String):
    var keys = property_path.split("/")
    var current = dict
    for key in keys:
        if typeof(current) != TYPE_DICTIONARY or not current.has(key):
            return null
        current = current[key]
    return current
