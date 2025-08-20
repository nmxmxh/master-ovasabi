extends Node

var settings = {}

func _ready():
    var file = File.new()
    if file.file_exists("res://config/settings.json"):
        file.open("res://config/settings.json", File.READ)
        settings = parse_json(file.get_as_text())
        file.close()
    print("Loaded settings: ", settings)
