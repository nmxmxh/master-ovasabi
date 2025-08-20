extends Node

class_name WindAffectorBehavior

func apply(body_rid: RID, properties: Dictionary):
    if properties.has("direction") and properties.has("strength"):
        var force = Vector3(properties.direction) * properties.strength
        PhysicsServer3D.body_apply_central_force(body_rid, force)

func process(delta: float, entity):
    if entity.metadata.dynamic_properties.has("wind_affector"):
        var wind = entity.metadata.dynamic_properties.wind_affector
        var force = Vector3(wind.direction) * wind.strength
        PhysicsServer3D.body_apply_central_force(entity.physics_body, force * delta)
