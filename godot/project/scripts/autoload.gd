extends Node

func _ready():
    var root = get_tree().get_root()
    if not root.has_node("main"):
        var main_node = Node.new()
        main_node.name = "main"
        main_node.set_script(load("res://scripts/main.gd"))
        root.add_child.call_deferred(main_node)
        print("[autoload] Auto-attached main.gd to /root/main node (deferred).")
    else:
        print("[autoload] main.gd already attached to /root/main.")
