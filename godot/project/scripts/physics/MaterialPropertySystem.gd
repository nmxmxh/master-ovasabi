class MaterialPropertySystem:
    var _property_map = {
        "thermal_expansion": {
            "metal": 0.000012,
            "wood": 0.000005,
            "plastic": 0.00007
        },
        "conductivity": {
            "metal": 401.0,
            "wood": 0.12,
            "plastic": 0.25
        }
    }
    func get_property(material_type: String, property: String) -> float:
        return _property_map.get(property, {}).get(material_type, 0.0)
