package registry

import (
	"encoding/json"
	"os"
	"reflect"
	"sync"
	"time"

	// Metadata proto and handler
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
)

type ServiceMethod struct {
	Name           string   `json:"name"`
	Parameters     []string `json:"parameters"`
	Permissions    []string `json:"permissions,omitempty"`
	AllFields      []string `json:"all_fields,omitempty"`
	RequiredFields []string `json:"required_fields,omitempty"`
	Description    string   `json:"description,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	Version        string   `json:"version,omitempty"`
}

type ServiceRegistration struct {
	ServiceName  string          `json:"service_name"`
	Methods      []ServiceMethod `json:"methods"`
	RegisteredAt time.Time       `json:"registered_at"`
	Description  string          `json:"description,omitempty"`
	Tags         []string        `json:"tags,omitempty"`
	Version      string          `json:"version,omitempty"`
	Owner        string          `json:"owner,omitempty"`
	External     bool            `json:"external,omitempty"` // True if registered by an external server
}

type EventRegistration struct {
	EventName      string    `json:"event_name"`
	Parameters     []string  `json:"parameters"`
	RequiredFields []string  `json:"required_fields"`
	Permissions    []string  `json:"permissions,omitempty"`
	AllFields      []string  `json:"all_fields,omitempty"`
	RegisteredAt   time.Time `json:"registered_at"`
	Description    string    `json:"description,omitempty"`
	Tags           []string  `json:"tags,omitempty"`
	Version        string    `json:"version,omitempty"`
	Owner          string    `json:"owner,omitempty"`
	External       bool      `json:"external,omitempty"` // True if registered by an external server
}

// In-memory registry with mutex for thread safety
var (
	serviceRegistry = make(map[string]ServiceRegistration)
	eventRegistry   = make(map[string]EventRegistration)
	regMutex        sync.RWMutex
)

// RegisterService can be called with or without method metadata. If Methods is empty and instance is provided, it will use reflection to enumerate methods.
// If protoDesc or struct tags are available, will attempt to extract permissions, fields, etc.
// Optionally pass a *commonpb.Metadata proto as the last argument to auto-populate registry fields.
func RegisterService(svc ServiceRegistration, instance ...interface{}) {
	regMutex.Lock()
	defer regMutex.Unlock()
	var meta *commonpb.Metadata
	if len(instance) > 1 {
		// If the last argument is a *commonpb.Metadata, use it
		if m, ok := instance[len(instance)-1].(*commonpb.Metadata); ok {
			meta = m
			instance = instance[:len(instance)-1]
		}
	}
	if meta != nil {
		// Use metadata handler to extract registry fields
		metaMap := metadata.ProtoToMap(meta)
		if desc, ok := metaMap["description"].(string); ok {
			svc.Description = desc
		}
		if tags, ok := metaMap["tags"].([]interface{}); ok {
			var tagStrs []string
			for _, t := range tags {
				if s, ok := t.(string); ok {
					tagStrs = append(tagStrs, s)
				}
			}
			svc.Tags = tagStrs
		}
		if v, ok := metaMap["version"].(string); ok {
			svc.Version = v
		}
		if owner, ok := metaMap["owner"].(string); ok {
			svc.Owner = owner
		}
		// Optionally, set External if metadata says so
		if ext, ok := metaMap["external"].(bool); ok {
			svc.External = ext
		}
		// Optionally, extract method-level metadata if present in metadata["methods"]
		if methods, ok := metaMap["methods"].(map[string]interface{}); ok {
			for i, m := range svc.Methods {
				if mMeta, ok := methods[m.Name].(map[string]interface{}); ok {
					if d, ok := mMeta["description"].(string); ok {
						svc.Methods[i].Description = d
					}
					if v, ok := mMeta["version"].(string); ok {
						svc.Methods[i].Version = v
					}
					if tgs, ok := mMeta["tags"].([]interface{}); ok {
						var tagStrs []string
						for _, t := range tgs {
							if s, ok := t.(string); ok {
								tagStrs = append(tagStrs, s)
							}
						}
						svc.Methods[i].Tags = tagStrs
					}
					if p, ok := mMeta["permissions"].([]interface{}); ok {
						var perms []string
						for _, t := range p {
							if s, ok := t.(string); ok {
								perms = append(perms, s)
							}
						}
						svc.Methods[i].Permissions = perms
					}
					if af, ok := mMeta["all_fields"].([]interface{}); ok {
						var afs []string
						for _, t := range af {
							if s, ok := t.(string); ok {
								afs = append(afs, s)
							}
						}
						svc.Methods[i].AllFields = afs
					}
					if rf, ok := mMeta["required_fields"].([]interface{}); ok {
						var rfs []string
						for _, t := range rf {
							if s, ok := t.(string); ok {
								rfs = append(rfs, s)
							}
						}
						svc.Methods[i].RequiredFields = rfs
					}
				}
			}
		}
	}
	if len(svc.Methods) == 0 && len(instance) > 0 && instance[0] != nil {
		t := reflect.TypeOf(instance[0])
		for i := 0; i < t.NumMethod(); i++ {
			m := t.Method(i)
			var params []string
			for j := 1; j < m.Type.NumIn(); j++ { // skip receiver
				params = append(params, m.Type.In(j).Name())
			}
			// Attempt to extract metadata from struct tags (if present)
			var desc, version string
			var tags, perms, allFields, reqFields []string // Initialize empty slices
			// Try to extract method-level metadata from struct tags (if method is a struct method)
			// If the struct implements a Metadata() map[string]any method, we can use that for method metadata.
			if metaMethod, found := t.MethodByName("Metadata"); found && metaMethod.Type.NumOut() == 1 && metaMethod.Type.Out(0).Kind() == reflect.Map {
				// Call Metadata() and try to extract info for this method
				results := metaMethod.Func.Call([]reflect.Value{reflect.ValueOf(instance[0])})
				if len(results) == 1 {
					if metaMap, ok := results[0].Interface().(map[string]any); ok {
						if mMeta, ok := metaMap[m.Name]; ok {
							if mMetaMap, ok := mMeta.(map[string]any); ok {
								if d, ok := mMetaMap["description"].(string); ok {
									desc = d
								}
								if v, ok := mMetaMap["version"].(string); ok {
									version = v
								}
								if tgs, ok := mMetaMap["tags"].([]string); ok {
									tags = tgs
								}
								if p, ok := mMetaMap["permissions"].([]string); ok {
									perms = p
								}
								if af, ok := mMetaMap["all_fields"].([]string); ok {
									allFields = af
								}
								if rf, ok := mMetaMap["required_fields"].([]string); ok {
									reqFields = rf
								}
							}
						}
					}
				}
			}
			svc.Methods = append(svc.Methods, ServiceMethod{
				Name:           m.Name,
				Parameters:     params,
				Permissions:    perms,
				AllFields:      allFields,
				RequiredFields: reqFields,
				Description:    desc,
				Tags:           tags,
				Version:        version,
			})
		}
	}
	// Optionally, mark as external if called by a remote server (extend logic as needed)
	serviceRegistry[svc.ServiceName] = svc
}

// Optionally pass a *commonpb.Metadata proto as the last argument to auto-populate event registry fields.
func RegisterEvent(evt EventRegistration, metaOpt ...interface{}) {
	regMutex.Lock()
	defer regMutex.Unlock()
	var meta *commonpb.Metadata
	if len(metaOpt) > 0 {
		if m, ok := metaOpt[len(metaOpt)-1].(*commonpb.Metadata); ok {
			meta = m
		}
	}
	if meta != nil {
		metaMap := metadata.ProtoToMap(meta)
		if desc, ok := metaMap["description"].(string); ok {
			evt.Description = desc
		}
		if tags, ok := metaMap["tags"].([]interface{}); ok {
			var tagStrs []string
			for _, t := range tags {
				if s, ok := t.(string); ok {
					tagStrs = append(tagStrs, s)
				}
			}
			evt.Tags = tagStrs
		}
		if v, ok := metaMap["version"].(string); ok {
			evt.Version = v
		}
		if owner, ok := metaMap["owner"].(string); ok {
			evt.Owner = owner
		}
		if ext, ok := metaMap["external"].(bool); ok {
			evt.External = ext
		}
		if perms, ok := metaMap["permissions"].([]interface{}); ok {
			var permsStr []string
			for _, t := range perms {
				if s, ok := t.(string); ok {
					permsStr = append(permsStr, s)
				}
			}
			evt.Permissions = permsStr
		}
		if af, ok := metaMap["all_fields"].([]interface{}); ok {
			var afs []string
			for _, t := range af {
				if s, ok := t.(string); ok {
					afs = append(afs, s)
				}
			}
			evt.AllFields = afs
		}
		if rf, ok := metaMap["required_fields"].([]interface{}); ok {
			var rfs []string
			for _, t := range rf {
				if s, ok := t.(string); ok {
					rfs = append(rfs, s)
				}
			}
			evt.RequiredFields = rfs
		}
	}
	eventRegistry[evt.EventName] = evt
}

func SaveRegistriesToDisk(dir string) error {
	regMutex.RLock()
	defer regMutex.RUnlock()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// Save services
	svcFile := dir + "/services.json"
	svcData, err := json.MarshalIndent(serviceRegistry, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(svcFile, svcData, 0644); err != nil {
		return err
	}
	// Save events
	evtFile := dir + "/events.json"
	evtData, err := json.MarshalIndent(eventRegistry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(evtFile, evtData, 0644)
}

func LoadRegistriesFromDisk(dir string) error {
	regMutex.Lock()
	defer regMutex.Unlock()
	// Load services
	svcFile := dir + "/services.json"
	if svcData, err := os.ReadFile(svcFile); err == nil {
		_ = json.Unmarshal(svcData, &serviceRegistry)
	}
	// Load events
	evtFile := dir + "/events.json"
	if evtData, err := os.ReadFile(evtFile); err == nil {
		_ = json.Unmarshal(evtData, &eventRegistry)
	}
	return nil
}

func GetServiceRegistry() map[string]ServiceRegistration {
	regMutex.RLock()
	defer regMutex.RUnlock()
	copy := make(map[string]ServiceRegistration, len(serviceRegistry))
	for k, v := range serviceRegistry {
		copy[k] = v
	}
	return copy
}

func GetEventRegistry() map[string]EventRegistration {
	regMutex.RLock()
	defer regMutex.RUnlock()
	copy := make(map[string]EventRegistration, len(eventRegistry))
	for k, v := range eventRegistry {
		copy[k] = v
	}
	return copy
}
