package metadata

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

// EnrichAndHashMetadata automatically enriches metadata with a hash and audit info.
// - Computes SHA256 hash of the marshaled metadata and stores in ServiceSpecific["hash"].
// - Appends/updates audit info (timestamp, context) in meta.Audit.
// - Optionally, if prevHash is provided, stores it as ServiceSpecific["prev_hash"].
func EnrichAndHashMetadata(meta *commonpb.Metadata, context string, prevHash ...string) {
	if meta == nil {
		return
	}
	// --- SCORING & TAX ENRICHMENT ---
	// Always add (never deduct), errors and successes both increment score
	metaMap := ProtoToMap(meta)
	calc, ok := metaMap["calculation"].(map[string]interface{})
	if !ok {
		// Handle gracefully: log or return if critical
		// For now, just return if calculation is required
		return
	}
	// 1. OneValueScore: count relationships (if present)
	if rels, ok := metaMap["relationships"].([]interface{}); ok {
		count := len(rels)
		score := OneValueScore(count)
		if prev, ok := calc["score"].(float64); ok {
			calc["score"] = prev + score
		} else {
			calc["score"] = score
		}
	}
	// 2. TASScore: if tas_list present
	if tasArr, ok := metaMap["tas_list"].([]interface{}); ok {
		tasList := []TAS{}
		for _, t := range tasArr {
			m, ok := t.(map[string]interface{})
			if !ok {
				continue
			}
			tas := TAS{}
			if v, ok := m["trust"].(float64); ok {
				tas.Trust = v
			}
			if v, ok := m["activity"].(float64); ok {
				tas.Activity = v
			}
			if v, ok := m["strength"].(float64); ok {
				tas.Strength = v
			}
			tasList = append(tasList, tas)
		}
		// Default weights: 0.5, 0.3, 0.2
		tasScore := TASScore(tasList, 0.5, 0.3, 0.2)
		if prev, ok := calc["score"].(float64); ok {
			calc["score"] = prev + tasScore
		} else {
			calc["score"] = tasScore
		}
		calc["tas"] = tasList
	}
	// 3. Tax: if connectors present
	if ss, ok := metaMap["service_specific"].(map[string]interface{}); ok {
		if taxation, ok := ss["taxation"].(map[string]interface{}); ok {
			if connectors, ok := taxation["connectors"].([]interface{}); ok {
				var connList []map[string]interface{}
				for _, c := range connectors {
					if m, ok := c.(map[string]interface{}); ok {
						connList = append(connList, m)
					}
				}
				tax := CalculateTotalTax(connList)
				calc["tax"] = tax
				taxation["total_tax"] = tax
				ss["taxation"] = taxation
			}
		}
	}
	metaMap["calculation"] = calc
	// Marshal back to proto
	updated := MapToProto(metaMap)
	// Assign fields explicitly to avoid copying a struct with sync.Mutex
	meta.Tags = updated.Tags
	meta.Features = updated.Features
	meta.CustomRules = updated.CustomRules
	meta.Audit = updated.Audit
	meta.Scheduling = updated.Scheduling
	meta.ServiceSpecific = updated.ServiceSpecific
	meta.KnowledgeGraph = updated.KnowledgeGraph
	// Marshal metadata to JSON for hashing
	b, err := json.Marshal(meta)
	if err == nil {
		h := sha256.Sum256(b)
		hashStr := hex.EncodeToString(h[:])
		if meta.ServiceSpecific == nil {
			meta.ServiceSpecific = &structpb.Struct{Fields: map[string]*structpb.Value{}}
		}
		meta.ServiceSpecific.Fields["hash"] = structpb.NewStringValue(hashStr)
		if len(prevHash) > 0 {
			meta.ServiceSpecific.Fields["prev_hash"] = structpb.NewStringValue(prevHash[0])
		}
	}
	// Append/update audit info
	auditMap := map[string]interface{}{
		"enriched_at": time.Now().Format(time.RFC3339),
		"context":     context,
	}
	if meta.Audit == nil {
		meta.Audit = MapToStruct(auditMap)
	} else {
		// Merge audit fields
		existing := StructToMap(meta.Audit)
		for k, v := range auditMap {
			existing[k] = v
		}
		meta.Audit = MapToStruct(existing)
	}
}
