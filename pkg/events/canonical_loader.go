package events

import (
	"encoding/json"
	"os"
)

// LoadCanonicalEvents loads canonical event types for the given service from service_registration.json.
func LoadCanonicalEvents(service string) []string {
	file, err := os.Open("config/service_registration.json")
	if err != nil {
		return nil
	}
	defer file.Close()

	var services []map[string]interface{}
	if err := json.NewDecoder(file).Decode(&services); err != nil {
		return nil
	}

	eventTypes := make([]string, 0)
	for _, svc := range services {
		if svc["name"] == service {
			version, _ := svc["version"].(string)
			endpoints, ok := svc["endpoints"].([]interface{})
			if !ok {
				continue
			}
			for _, ep := range endpoints {
				epMap, ok := ep.(map[string]interface{})
				if !ok {
					continue
				}
				actions, ok := epMap["actions"].([]interface{})
				if !ok {
					continue
				}
				for _, act := range actions {
					if actStr, ok := act.(string); ok {
						for _, state := range []string{"requested", "started", "success", "failed", "completed"} {
							eventTypes = append(eventTypes, service+":"+actStr+":"+version+":"+state)
						}
					}
				}
			}
		}
	}
	return eventTypes
}
