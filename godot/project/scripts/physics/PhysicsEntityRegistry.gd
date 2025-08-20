extends Node

var WindAffectorBehavior = load("res://scripts/physics/WindAffectorBehavior.gd")
var ThermalBehavior = load("res://scripts/physics/BehaviorBase.gd")
var ElectromagneticBehavior = load("res://scripts/physics/ElectromagneticBehavior.gd")

var _entities := {}
var _material_library := {}
var _behavior_registry := {}

func get_entities():
    return _entities.values()

func _ready():
    _load_default_materials()
    _load_behavior_templates()

func register_entity(entity_id: String, metadata: Dictionary):
    var entity = PhysicsEntity.new(metadata)
    _entities[entity_id] = entity
    _apply_physics_properties(entity_id)
    _attach_behavior(entity_id)

func update_property(entity_id: String, property_path: String, value):
    var path = property_path.split("/")
    var current = _entities[entity_id].metadata
    for i in range(path.size() - 1):
        current = current[path[i]]
    current[path[-1]] = value
    _apply_physics_properties(entity_id)

func _apply_physics_properties(entity_id: String):
    var entity = _entities[entity_id]
    var body_rid = entity.physics_body
    PhysicsServer3D.body_set_param(body_rid, PhysicsServer3D.BODY_PARAM_MASS, entity.metadata.base_properties.mass)
    PhysicsServer3D.body_set_param(body_rid, PhysicsServer3D.BODY_PARAM_FRICTION, entity.metadata.base_properties.friction)
    var material = _material_library[entity.metadata.material.type]
    PhysicsServer3D.body_set_collision_layer(body_rid, material.collision_layer)
    for behavior in entity.metadata.dynamic_properties:
        match behavior:
            "wind_affector":
                WindAffectorBehavior.new().apply(body_rid, entity.metadata.dynamic_properties[behavior])
            "thermal":
                ThermalBehavior.new().apply(body_rid, entity.metadata.dynamic_properties[behavior])
            "electromagnetic":
                ElectromagneticBehavior.new().apply(body_rid, entity.metadata.dynamic_properties[behavior])

func _attach_behavior(entity_id: String):
    var entity = _entities[entity_id]
    if entity.metadata.has("behavior"):
        var script = load(entity.metadata.behavior.script)
        var instance = script.new()
        instance.set_parameters(entity.metadata.behavior.parameters)
        entity.behavior_instance = instance

func _load_default_materials():
    _material_library = {
        "metal": { "density": 7.8, "restitution": 0.3, "collision_layer": 1 },
        "wood": { "density": 0.7, "restitution": 0.5, "collision_layer": 2 },
        "fluid": { "density": 1.0, "restitution": 0.1, "collision_layer": 3 }
    }

func _load_behavior_templates():
    _behavior_registry = {
        "wind_affector": WindAffectorBehavior,
        "thermal": ThermalBehavior,
        "electromagnetic": ElectromagneticBehavior
    }
