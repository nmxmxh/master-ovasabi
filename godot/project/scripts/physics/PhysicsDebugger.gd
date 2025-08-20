extends Node

func _ready():
    var viewport = get_viewport()
    if viewport.has_method("set_debug_draw"):
        viewport.set_debug_draw(Viewport.DEBUG_DRAW_UNSHADED)

func _process(delta):
    if Input.is_action_just_pressed("physics_debug"):
        toggle_debug_view()

func toggle_debug_view():
    var viewport = get_viewport()
    if viewport.has_method("set_debug_draw"):
        var current_mode = viewport.debug_draw
        var next_mode = (current_mode + 1) % 3
        viewport.set_debug_draw(next_mode)
        if next_mode == Viewport.DEBUG_DRAW_UNSHADED:
            # Visualize physics properties
            for entity in PhysicsEntityRegistry.get_entities():
                var pos = entity.position if entity.has_method("position") else Vector3.ZERO
                var props = entity.metadata.dynamic_properties
                if props.has("wind_affector"):
                    var wind_dir = Vector3(props.wind_affector.direction)
                    var strength = props.wind_affector.strength
                    print("Wind arrow at ", pos, " direction: ", wind_dir, " strength: ", strength)
