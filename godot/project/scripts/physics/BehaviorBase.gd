extends Node

class_name PhysicsBehavior

func apply(body_rid: RID, properties: Dictionary):
    pass

func process(delta: float, entity):
    pass

class ThermalBehavior:
    extends PhysicsBehavior

    func apply(body_rid: RID, properties: Dictionary):
        # No built-in thermal conductivity in Godot physics
        # You can store or use this value in your own logic if needed
        pass

    func process(delta: float, entity):
        if entity.metadata.dynamic_properties.has("thermal"):
            var temp = entity.metadata.dynamic_properties.thermal.temperature
            var ambient = 20.0
            entity.metadata.dynamic_properties.thermal.temperature = lerp(temp, ambient, delta * 0.1)
            if temp > 100:
                entity.metadata.base_properties.restitution = min(0.9, temp / 200.0)
