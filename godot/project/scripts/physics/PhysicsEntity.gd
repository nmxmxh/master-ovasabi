class_name PhysicsEntity
extends RefCounted

var metadata: Dictionary
var physics_body: RID
var behavior_instance: Node

func _init(metadata_dict: Dictionary):
    metadata = metadata_dict
    physics_body = PhysicsServer3D.body_create()
    var shape: Shape3D
    match metadata.collision_shape:
        "box":
            shape = BoxShape3D.new()
            shape.size = metadata.dimensions
        "sphere":
            shape = SphereShape3D.new()
            shape.radius = metadata.radius
        "mesh":
            shape = ConcavePolygonShape3D.new()
            shape.data = _generate_collision_mesh(metadata.mesh_data)
    PhysicsServer3D.body_add_shape(physics_body, shape.get_rid())
    # Space must be set from a Node context after instantiation

func set_space(space: RID):
    PhysicsServer3D.body_set_space(physics_body, space)

func _generate_collision_mesh(mesh_data):
    # TODO: Implement mesh generation from mesh_data
    # For now, return an empty PackedVector3Array
    return PackedVector3Array()

func apply_behavior(delta: float):
    if behavior_instance:
        behavior_instance.process(delta, self)
