extends Node

class_name EnvironmentalEffects

var gravity: Vector3 = Vector3(0, -9.8, 0)
var air_density: float = 1.2
var temperature: float = 20.0
var material_property_system = load("res://scripts/physics/MaterialPropertySystem.gd").new()

func apply_environment(entity):
    var volume = entity.calculate_volume()
    var buoyancy = -gravity * air_density * volume
    PhysicsServer3D.body_apply_central_force(entity.physics_body, buoyancy)
    var material = entity.metadata.material.type
    var expansion = material_property_system.get_property(material, "thermal_expansion")
    if entity.metadata.dynamic_properties.has("thermal"):
        var temp_diff = entity.metadata.dynamic_properties.thermal.temperature - temperature
        var scale_factor = 1.0 + (expansion * temp_diff)
        entity.scale_object(Vector3.ONE * scale_factor)
