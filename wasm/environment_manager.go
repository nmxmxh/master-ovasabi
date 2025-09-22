package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"sync"
	"time"
)

// EnvironmentManager manages environment chunks and streaming
type EnvironmentManager struct {
	chunks        map[string]*EnvironmentChunk
	loadedChunks  map[string]bool
	chunksMutex   sync.RWMutex
	proximity     *ProximityGrid
	campaignID    string
	rules         *CampaignPhysicsRules
	downloadQueue chan string
	active        bool
	stopChan      chan bool
	workerCount   int
	performance   *EnvironmentPerformance
}

// EnvironmentPerformance tracks environment system performance
type EnvironmentPerformance struct {
	LoadedChunks     int       `json:"loaded_chunks"`
	TotalChunks      int       `json:"total_chunks"`
	MemoryUsage      int64     `json:"memory_usage"`
	DownloadSpeed    float32   `json:"download_speed"`
	CompressionRatio float32   `json:"compression_ratio"`
	LastUpdate       time.Time `json:"last_update"`
}

// EnvironmentLODLevel defines level of detail for environment chunks
type EnvironmentLODLevel struct {
	Level        int     `json:"level"`
	Distance     float32 `json:"distance"`
	PolygonCount int     `json:"polygon_count"`
	TextureSize  int     `json:"texture_size"`
	Compressed   bool    `json:"compressed"`
}

// NewEnvironmentManager creates a new environment manager
func NewEnvironmentManager(campaignID string, rules *CampaignPhysicsRules) *EnvironmentManager {
	em := &EnvironmentManager{
		chunks:        make(map[string]*EnvironmentChunk),
		loadedChunks:  make(map[string]bool),
		proximity:     NewProximityGrid(rules.ChunkSize, rules.WorldBounds),
		campaignID:    campaignID,
		rules:         rules,
		downloadQueue: make(chan string, 100),
		active:        false,
		stopChan:      make(chan bool),
		workerCount:   2,
		performance:   &EnvironmentPerformance{},
	}

	// Start download workers
	for i := 0; i < em.workerCount; i++ {
		go em.downloadWorker(i)
	}

	return em
}

// Start starts the environment manager
func (em *EnvironmentManager) Start() {
	em.active = true
	log.Printf("[EnvironmentManager] Started for campaign %s", em.campaignID)
}

// Stop stops the environment manager
func (em *EnvironmentManager) Stop() {
	em.active = false
	close(em.stopChan)
	log.Printf("[EnvironmentManager] Stopped for campaign %s", em.campaignID)
}

// UpdateProximity updates proximity-based chunk loading
func (em *EnvironmentManager) UpdateProximity(position Vector3, viewDirection Vector3) {
	if !em.active {
		return
	}

	// Get chunks within view distance
	nearbyChunks := em.getChunksInRange(position, viewDirection, em.rules.ChunkSize*5)

	// Download missing chunks
	for _, chunkID := range nearbyChunks {
		if !em.isChunkLoaded(chunkID) {
			em.queueChunkDownload(chunkID)
		}
	}

	// Unload distant chunks
	em.unloadDistantChunks(position, em.rules.ChunkSize*10)

	// Update performance metrics
	em.updatePerformance()
}

// getChunksInRange gets chunk IDs within a certain range
func (em *EnvironmentManager) getChunksInRange(position Vector3, viewDirection Vector3, maxDistance float32) []string {
	var chunks []string

	// Calculate grid bounds
	chunkSize := em.rules.ChunkSize
	radius := int(maxDistance / chunkSize)

	centerX := int(position.X / chunkSize)
	centerY := int(position.Y / chunkSize)
	centerZ := int(position.Z / chunkSize)

	// Get chunks in a sphere around the position
	for x := centerX - radius; x <= centerX+radius; x++ {
		for y := centerY - radius; y <= centerY+radius; y++ {
			for z := centerZ - radius; z <= centerZ+radius; z++ {
				chunkPos := Vector3{
					X: float32(x) * chunkSize,
					Y: float32(y) * chunkSize,
					Z: float32(z) * chunkSize,
				}

				// Check if within sphere
				distance := em.calculateDistance(position, chunkPos)
				if distance <= maxDistance {
					chunkID := em.getChunkID(chunkPos)
					chunks = append(chunks, chunkID)
				}
			}
		}
	}

	return chunks
}

// calculateDistance calculates distance between two positions
func (em *EnvironmentManager) calculateDistance(pos1, pos2 Vector3) float32 {
	dx := pos1.X - pos2.X
	dy := pos1.Y - pos2.Y
	dz := pos1.Z - pos2.Z
	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
}

// getChunkID gets the chunk ID for a position
func (em *EnvironmentManager) getChunkID(position Vector3) string {
	chunkSize := em.rules.ChunkSize
	x := int(position.X / chunkSize)
	y := int(position.Y / chunkSize)
	z := int(position.Z / chunkSize)
	return fmt.Sprintf("%s_%d_%d_%d", em.campaignID, x, y, z)
}

// isChunkLoaded checks if a chunk is loaded
func (em *EnvironmentManager) isChunkLoaded(chunkID string) bool {
	em.chunksMutex.RLock()
	defer em.chunksMutex.RUnlock()
	return em.loadedChunks[chunkID]
}

// queueChunkDownload queues a chunk for download
func (em *EnvironmentManager) queueChunkDownload(chunkID string) {
	select {
	case em.downloadQueue <- chunkID:
		// Chunk queued for download
	default:
		log.Printf("[EnvironmentManager] Download queue full, dropping chunk: %s", chunkID)
	}
}

// downloadWorker processes chunk downloads
func (em *EnvironmentManager) downloadWorker(workerID int) {
	for {
		select {
		case chunkID := <-em.downloadQueue:
			em.downloadChunk(chunkID)
		case <-em.stopChan:
			return
		}
	}
}

// downloadChunk downloads a chunk from the server
func (em *EnvironmentManager) downloadChunk(chunkID string) {
	if em.isChunkLoaded(chunkID) {
		return
	}

	log.Printf("[EnvironmentManager] Downloading chunk: %s", chunkID)

	// Create chunk data (in real implementation, this would come from server)
	chunk := em.createChunkData(chunkID)

	// Compress chunk data
	compressedData, err := em.compressChunkData(chunk.Data)
	if err != nil {
		log.Printf("[EnvironmentManager] Failed to compress chunk %s: %v", chunkID, err)
		return
	}

	chunk.Data = compressedData
	chunk.Compressed = true
	chunk.Size = int64(len(compressedData))

	// Calculate checksum
	chunk.Checksum = em.calculateChecksum(compressedData)

	// Store chunk
	em.storeChunk(chunk)

	log.Printf("[EnvironmentManager] Downloaded chunk: %s (size: %d bytes)", chunkID, chunk.Size)
}

// createChunkData creates chunk data (placeholder implementation)
func (em *EnvironmentManager) createChunkData(chunkID string) *EnvironmentChunk {
	// Parse chunk ID to get position
	var x, y, z int
	fmt.Sscanf(chunkID, "%s_%d_%d_%d", &em.campaignID, &x, &y, &z)

	position := Vector3{
		X: float32(x) * em.rules.ChunkSize,
		Y: float32(y) * em.rules.ChunkSize,
		Z: float32(z) * em.rules.ChunkSize,
	}

	// Create chunk data (in real implementation, this would be loaded from server)
	chunkData := em.generateChunkData(position, em.rules.ChunkSize)

	return &EnvironmentChunk{
		ID:           chunkID,
		CampaignID:   em.campaignID,
		Position:     position,
		Bounds:       em.calculateChunkBounds(position, em.rules.ChunkSize),
		LOD:          0,
		Data:         chunkData,
		Dependencies: []string{},
		Size:         int64(len(chunkData)),
		Compressed:   false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Properties:   make(map[string]interface{}),
	}
}

// generateChunkData generates chunk data (placeholder implementation)
func (em *EnvironmentManager) generateChunkData(position Vector3, size float32) []byte {
	// This would generate actual 3D environment data
	// For now, we'll create a simple JSON structure
	chunkData := map[string]interface{}{
		"position":  position,
		"size":      size,
		"objects":   []interface{}{},
		"materials": []interface{}{},
		"lights":    []interface{}{},
		"physics":   map[string]interface{}{},
	}

	// Add some sample objects
	for i := 0; i < 10; i++ {
		obj := map[string]interface{}{
			"id":   fmt.Sprintf("obj_%d", i),
			"type": "cube",
			"position": Vector3{
				X: position.X + float32(i%5)*size/5,
				Y: position.Y + float32(i/5)*size/5,
				Z: position.Z,
			},
			"scale":    Vector3{X: 1, Y: 1, Z: 1},
			"material": "default",
		}
		chunkData["objects"] = append(chunkData["objects"].([]interface{}), obj)
	}

	// Convert to JSON
	jsonData, err := json.Marshal(chunkData)
	if err != nil {
		log.Printf("[EnvironmentManager] Failed to marshal chunk data: %v", err)
		return []byte{}
	}

	return jsonData
}

// calculateChunkBounds calculates the bounding box for a chunk
func (em *EnvironmentManager) calculateChunkBounds(position Vector3, size float32) BoundingBox {
	halfSize := size / 2
	return BoundingBox{
		Min: Vector3{
			X: position.X - halfSize,
			Y: position.Y - halfSize,
			Z: position.Z - halfSize,
		},
		Max: Vector3{
			X: position.X + halfSize,
			Y: position.Y + halfSize,
			Z: position.Z + halfSize,
		},
	}
}

// compressChunkData compresses chunk data using gzip
func (em *EnvironmentManager) compressChunkData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	_, err := writer.Write(data)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decompressChunkData decompresses chunk data
func (em *EnvironmentManager) decompressChunkData(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// calculateChecksum calculates SHA256 checksum of data
func (em *EnvironmentManager) calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// storeChunk stores a chunk in memory
func (em *EnvironmentManager) storeChunk(chunk *EnvironmentChunk) {
	em.chunksMutex.Lock()
	defer em.chunksMutex.Unlock()

	em.chunks[chunk.ID] = chunk
	em.loadedChunks[chunk.ID] = true
}

// unloadDistantChunks unloads chunks that are too far away
func (em *EnvironmentManager) unloadDistantChunks(position Vector3, maxDistance float32) {
	em.chunksMutex.Lock()
	defer em.chunksMutex.Unlock()

	for chunkID, chunk := range em.chunks {
		distance := em.calculateDistance(position, chunk.Position)
		if distance > maxDistance {
			// Unload chunk
			delete(em.chunks, chunkID)
			delete(em.loadedChunks, chunkID)
			log.Printf("[EnvironmentManager] Unloaded distant chunk: %s", chunkID)
		}
	}
}

// GetChunk gets a chunk by ID
func (em *EnvironmentManager) GetChunk(chunkID string) *EnvironmentChunk {
	em.chunksMutex.RLock()
	defer em.chunksMutex.RUnlock()

	return em.chunks[chunkID]
}

// GetChunksInRange gets all chunks within a certain range
func (em *EnvironmentManager) GetChunksInRange(position Vector3, maxDistance float32) []*EnvironmentChunk {
	em.chunksMutex.RLock()
	defer em.chunksMutex.RUnlock()

	var chunks []*EnvironmentChunk

	for _, chunk := range em.chunks {
		distance := em.calculateDistance(position, chunk.Position)
		if distance <= maxDistance {
			chunks = append(chunks, chunk)
		}
	}

	return chunks
}

// updatePerformance updates performance metrics
func (em *EnvironmentManager) updatePerformance() {
	em.chunksMutex.RLock()
	defer em.chunksMutex.RUnlock()

	em.performance.LoadedChunks = len(em.loadedChunks)
	em.performance.TotalChunks = len(em.chunks)
	em.performance.LastUpdate = time.Now()

	// Calculate memory usage
	var totalSize int64
	for _, chunk := range em.chunks {
		totalSize += chunk.Size
	}
	em.performance.MemoryUsage = totalSize

	// Calculate compression ratio
	if len(em.chunks) > 0 {
		var totalCompressed int64
		var totalUncompressed int64

		for _, chunk := range em.chunks {
			totalCompressed += chunk.Size
			if chunk.Compressed {
				// Estimate uncompressed size (this is approximate)
				totalUncompressed += chunk.Size * 3
			} else {
				totalUncompressed += chunk.Size
			}
		}

		if totalUncompressed > 0 {
			em.performance.CompressionRatio = float32(totalCompressed) / float32(totalUncompressed)
		}
	}
}

// GetPerformance returns current performance metrics
func (em *EnvironmentManager) GetPerformance() *EnvironmentPerformance {
	em.chunksMutex.RLock()
	defer em.chunksMutex.RUnlock()

	return em.performance
}

// GetChunkData gets decompressed chunk data
func (em *EnvironmentManager) GetChunkData(chunkID string) ([]byte, error) {
	chunk := em.GetChunk(chunkID)
	if chunk == nil {
		return nil, fmt.Errorf("chunk not found: %s", chunkID)
	}

	if chunk.Compressed {
		return em.decompressChunkData(chunk.Data)
	}

	return chunk.Data, nil
}

// ValidateChunk validates chunk data integrity
func (em *EnvironmentManager) ValidateChunk(chunkID string) bool {
	chunk := em.GetChunk(chunkID)
	if chunk == nil {
		return false
	}

	// Calculate current checksum
	currentChecksum := em.calculateChecksum(chunk.Data)

	// Compare with stored checksum
	return currentChecksum == chunk.Checksum
}

// GetChunkLOD gets the appropriate LOD level for a chunk based on distance
func (em *EnvironmentManager) GetChunkLOD(position Vector3, chunkPosition Vector3) int {
	distance := em.calculateDistance(position, chunkPosition)

	// Determine LOD based on distance
	if distance <= em.rules.LODDistances[0] {
		return 0 // Highest detail
	} else if distance <= em.rules.LODDistances[1] {
		return 1 // Medium detail
	} else if distance <= em.rules.LODDistances[2] {
		return 2 // Low detail
	} else {
		return 3 // Culled
	}
}

// UpdateChunkLOD updates the LOD level for a chunk
func (em *EnvironmentManager) UpdateChunkLOD(chunkID string, lod int) {
	em.chunksMutex.Lock()
	defer em.chunksMutex.Unlock()

	chunk, exists := em.chunks[chunkID]
	if !exists {
		return
	}

	chunk.LOD = lod
	chunk.UpdatedAt = time.Now()
}
