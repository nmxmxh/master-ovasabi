package ai

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// AIRepo handles storage and retrieval of AI models/updates, with master table integration and distributed support.
type Repo struct {
	db         *sql.DB
	nodeID     string           // For distributed/multi-node operation
	masterRepo MasterRepository // Interface to master table for cross-entity analytics
	peerSync   *PeerSyncManager
}

// NewAIRepo creates a new AIRepo.
func NewAIRepo(db *sql.DB, nodeID string, masterRepo MasterRepository) *Repo {
	return &Repo{db: db, nodeID: nodeID, masterRepo: masterRepo}
}

// SaveModel stores a model or update in the _ai table, linking to the master table if needed.
func (r *Repo) SaveModel(ctx context.Context, typ string, model *Model) error {
	metaJSON, err := json.Marshal(model.Meta)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO _ai (type, data, meta, hash, version, parent_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`, typ, model.Data, metaJSON, model.Hash, model.Version, model.ParentHash)
	return err
}

// GetModelByHash retrieves a model or update by hash.
func (r *Repo) GetModelByHash(ctx context.Context, hash string) (*Model, error) {
	row := r.db.QueryRowContext(ctx, `SELECT data, meta, hash, version, parent_hash FROM _ai WHERE hash = $1`, hash)
	var data []byte
	var metaJSON []byte
	var hashVal, version, parentHash string
	if err := row.Scan(&data, &metaJSON, &hashVal, &version, &parentHash); err != nil {
		return nil, err
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(metaJSON, &meta); err != nil {
		return nil, err
	}
	return &Model{Data: data, Meta: meta, Hash: hashVal, Version: version, ParentHash: parentHash}, nil
}

// ListModels lists all models/updates, optionally filtered by type or node.
func (r *Repo) ListModels(ctx context.Context, typ string) ([]*Model, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT data, meta, hash, version, parent_hash FROM _ai WHERE type = $1 ORDER BY created_at DESC`, typ)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var models []*Model
	for rows.Next() {
		var data []byte
		var metaJSON []byte
		var hashVal, version, parentHash string
		if err := rows.Scan(&data, &metaJSON, &hashVal, &version, &parentHash); err != nil {
			return nil, err
		}
		var meta map[string]interface{}
		if err := json.Unmarshal(metaJSON, &meta); err != nil {
			return nil, err
		}
		models = append(models, &Model{Data: data, Meta: meta, Hash: hashVal, Version: version, ParentHash: parentHash})
	}
	return models, nil
}

// SyncWithPeers synchronizes with other peers in the network.
func (r *Repo) SyncWithPeers(ctx context.Context, _, restPeerURLs []string) error {
	// WebSocket broadcast
	if r.peerSync != nil {
		models, err := r.ListModels(ctx, "model")
		if err != nil {
			return err
		}
		for _, model := range models {
			r.peerSync.BroadcastModel(model)
		}
	}
	// REST push
	models, err := r.ListModels(ctx, "model")
	if err != nil {
		return err
	}
	for _, peerURL := range restPeerURLs {
		for _, model := range models {
			if err := r.PushModelToPeerREST(ctx, peerURL, model); err != nil {
				log.Printf("[AIRepo] Push to peer %s failed: %v", peerURL, err)
			}
		}
		// Optionally, pull from peer
		peerModels, err := r.PullModelsFromPeerREST(ctx, peerURL)
		if err == nil {
			for _, m := range peerModels {
				if err := r.SaveModel(ctx, "model", m); err != nil {
					log.Printf("[AIRepo] Save model from peer %s failed: %v", peerURL, err)
				}
			}
		}
	}
	return nil
}

// SaveHiddenMetadata stores advanced/hidden metadata for a model/update (e.g., for compliance, audit, or explainability).
func (r *Repo) SaveHiddenMetadata(ctx context.Context, hash string, hiddenMeta map[string]interface{}) error {
	hiddenJSON, err := json.Marshal(hiddenMeta)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE _ai SET meta = jsonb_set(meta, '{hidden}', $1::jsonb, true), updated_at = NOW() WHERE hash = $2
	`, hiddenJSON, hash)
	return err
}

// MasterRepository is a minimal interface for master table integration.
type MasterRepository interface {
	LinkAIModel(ctx context.Context, modelHash string, nodeID string, createdAt time.Time) error
}

// PeerSyncManager manages WebSocket connections to peers for distributed sync.
type PeerSyncManager struct {
	peers map[string]*websocket.Conn // peerID -> connection
	mu    sync.RWMutex
	repo  *Repo
}

func NewPeerSyncManager(repo *Repo) *PeerSyncManager {
	return &PeerSyncManager{
		peers: make(map[string]*websocket.Conn),
		repo:  repo,
	}
}

func (psm *PeerSyncManager) AddPeer(peerID string, conn *websocket.Conn) {
	psm.mu.Lock()
	defer psm.mu.Unlock()
	psm.peers[peerID] = conn
	go psm.listenPeer(peerID, conn)
}

func (psm *PeerSyncManager) BroadcastModel(model *Model) {
	psm.mu.RLock()
	defer psm.mu.RUnlock()
	msg, err := json.Marshal(model)
	if err != nil {
		log.Printf("[AIRepo] Broadcast marshal error: %v", err)
		return
	}
	for peerID, conn := range psm.peers {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("[AIRepo] Broadcast to %s failed: %v", peerID, err)
		}
	}
}

func (psm *PeerSyncManager) listenPeer(peerID string, conn *websocket.Conn) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[AIRepo] Peer %s disconnected: %v", peerID, err)
			psm.mu.Lock()
			delete(psm.peers, peerID)
			psm.mu.Unlock()
			return
		}
		var model Model
		if err := json.Unmarshal(msg, &model); err == nil {
			if err := psm.repo.SaveModel(context.Background(), "model", &model); err != nil {
				log.Printf("[AIRepo] Synced model from peer %s: %s, error: %v", peerID, model.Hash, err)
			}
		}
	}
}

// REST-based distributed sync: Push a model to a peer's REST endpoint.
func (r *Repo) PushModelToPeerREST(ctx context.Context, peerURL string, model *Model) error {
	msg, err := json.Marshal(model)
	if err != nil {
		return fmt.Errorf("failed to marshal model: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, peerURL+"/ai/model/sync", bytes.NewReader(msg))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to push model to peer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer returned non-200 status: %d", resp.StatusCode)
	}
	return nil
}

// REST-based distributed sync: Pull models from a peer's REST endpoint.
func (r *Repo) PullModelsFromPeerREST(ctx context.Context, peerURL string) ([]*Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, peerURL+"/ai/model/sync", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to pull models from peer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("peer returned non-200 status: %d", resp.StatusCode)
	}

	var models []*Model
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, fmt.Errorf("failed to decode models: %w", err)
	}
	return models, nil
}
