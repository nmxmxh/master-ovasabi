extends Node

class_name ElectromagneticBehavior

func apply(body_rid: RID, properties: Dictionary):
    # Example: Apply electromagnetic force
    if properties.has("field_strength") and properties.has("direction"):
        var force = Vector3(properties.direction) * properties.field_strength
        PhysicsServer3D.body_apply_central_force(body_rid, force)

func process(delta: float, entity):
    # Example: Simulate time-varying electromagnetic effects
    if entity.metadata.dynamic_properties.has("electromagnetic"):
        var em = entity.metadata.dynamic_properties.electromagnetic
        var time_factor = sin(Time.get_ticks_msec() / 1000.0)
        var force = Vector3(em.direction) * em.field_strength * time_factor
        PhysicsServer3D.body_apply_central_force(entity.physics_body, force)
