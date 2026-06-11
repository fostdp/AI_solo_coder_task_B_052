package lora

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"
	"sync"
	"time"
)

type BloomFilter struct {
	bitSet    []uint64
	size      uint64
	numHashes uint
	mu        sync.RWMutex
	itemCount uint64
	createdAt time.Time
}

type PacketDeduplicator struct {
	filter       *BloomFilter
	cache        map[string]time.Time
	cacheTTL     time.Duration
	maxCacheSize int
	mu           sync.RWMutex
}

func NewBloomFilter(expectedItems uint64, falsePositiveRate float64) *BloomFilter {
	if falsePositiveRate <= 0 {
		falsePositiveRate = 0.01
	}
	if expectedItems == 0 {
		expectedItems = 100000
	}

	size := optimalSize(expectedItems, falsePositiveRate)
	numHashes := optimalNumHashes(size, expectedItems)

	return &BloomFilter{
		bitSet:    make([]uint64, (size+63)/64),
		size:      size,
		numHashes: numHashes,
		createdAt: time.Now(),
	}
}

func optimalSize(n uint64, p float64) uint64 {
	m := -float64(n) * math.Log(p) / (math.Log(2) * math.Log(2))
	return uint64(m)
}

func optimalNumHashes(m, n uint64) uint {
	k := float64(m) / float64(n) * math.Log(2)
	if k < 1 {
		return 1
	}
	return uint(k)
}

func (bf *BloomFilter) hashValues(data []byte) []uint64 {
	hashes := make([]uint64, bf.numHashes)

	h1 := fnv.New64a()
	h1.Write(data)
	hash1 := h1.Sum64()

	sha := sha256.Sum256(data)
	hash2 := binary.BigEndian.Uint64(sha[:8])

	for i := uint(0); i < bf.numHashes; i++ {
		combined := hash1 + uint64(i)*hash2
		hashes[i] = combined % bf.size
	}

	return hashes
}

func (bf *BloomFilter) Add(data []byte) {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	hashes := bf.hashValues(data)
	for _, h := range hashes {
		wordIdx := h / 64
		bitIdx := h % 64
		bf.bitSet[wordIdx] |= 1 << bitIdx
	}
	bf.itemCount++
}

func (bf *BloomFilter) AddString(s string) {
	bf.Add([]byte(s))
}

func (bf *BloomFilter) Test(data []byte) bool {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	hashes := bf.hashValues(data)
	for _, h := range hashes {
		wordIdx := h / 64
		bitIdx := h % 64
		if bf.bitSet[wordIdx]&(1<<bitIdx) == 0 {
			return false
		}
	}
	return true
}

func (bf *BloomFilter) TestString(s string) bool {
	return bf.Test([]byte(s))
}

func (bf *BloomFilter) TestAndAdd(data []byte) bool {
	exists := bf.Test(data)
	if !exists {
		bf.Add(data)
	}
	return exists
}

func (bf *BloomFilter) TestAndAddString(s string) bool {
	return bf.TestAndAdd([]byte(s))
}

func (bf *BloomFilter) Count() uint64 {
	bf.mu.RLock()
	defer bf.mu.RUnlock()
	return bf.itemCount
}

func (bf *BloomFilter) FalsePositiveProbability() float64 {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	k := float64(bf.numHashes)
	n := float64(bf.itemCount)
	m := float64(bf.size)

	exponent := -k * n / m
	return math.Pow(1-math.Pow(math.E, exponent), k)
}

func (bf *BloomFilter) Reset() {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	bf.bitSet = make([]uint64, (bf.size+63)/64)
	bf.itemCount = 0
	bf.createdAt = time.Now()
}

func NewPacketDeduplicator(expectedItems int, cacheTTL time.Duration, maxCacheSize int) *PacketDeduplicator {
	if expectedItems <= 0 {
		expectedItems = 100000
	}
	if cacheTTL <= 0 {
		cacheTTL = 24 * time.Hour
	}
	if maxCacheSize <= 0 {
		maxCacheSize = 50000
	}

	return &PacketDeduplicator{
		filter:       NewBloomFilter(uint64(expectedItems), 0.001),
		cache:        make(map[string]time.Time),
		cacheTTL:     cacheTTL,
		maxCacheSize: maxCacheSize,
	}
}

func GeneratePacketID(deviceID string, timestamp time.Time, sequence uint64) string {
	if sequence == 0 {
		sequence = uint64(time.Now().UnixNano())
	}

	h := fnv.New64a()
	h.Write([]byte(deviceID))
	h.Write([]byte(fmt.Sprintf("%d", timestamp.UnixNano())))
	h.Write([]byte(fmt.Sprintf("%d", sequence)))

	return fmt.Sprintf("%s-%d-%x", deviceID, timestamp.Unix(), h.Sum64())
}

func (pd *PacketDeduplicator) IsDuplicate(packetID string) bool {
	if packetID == "" {
		return false
	}

	pd.mu.Lock()
	defer pd.mu.Unlock()

	pd.cleanupExpired()

	if ts, exists := pd.cache[packetID]; exists {
		if time.Since(ts) < pd.cacheTTL {
			return true
		}
	}

	if pd.filter.TestString(packetID) {
		if _, exists := pd.cache[packetID]; !exists {
			pd.cache[packetID] = time.Now()
		}
		return true
	}

	pd.filter.AddString(packetID)
	pd.cache[packetID] = time.Now()

	if len(pd.cache) > pd.maxCacheSize {
		pd.compressCache()
	}

	return false
}

func (pd *PacketDeduplicator) cleanupExpired() {
	now := time.Now()
	for id, ts := range pd.cache {
		if now.Sub(ts) > pd.cacheTTL {
			delete(pd.cache, id)
		}
	}
}

func (pd *PacketDeduplicator) compressCache() {
	type entry struct {
		id string
		ts time.Time
	}

	entries := make([]entry, 0, len(pd.cache))
	for id, ts := range pd.cache {
		entries = append(entries, entry{id, ts})
	}

	for i := 0; i < len(entries)-1; i++ {
		for j := 0; j < len(entries)-i-1; j++ {
			if entries[j].ts.Before(entries[j+1].ts) {
				entries[j], entries[j+1] = entries[j+1], entries[j]
			}
		}
	}

	targetSize := pd.maxCacheSize * 3 / 4
	pd.cache = make(map[string]time.Time)
	for i := 0; i < targetSize && i < len(entries); i++ {
		pd.cache[entries[i].id] = entries[i].ts
	}
}

func (pd *PacketDeduplicator) GetStats() map[string]interface{} {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	return map[string]interface{}{
		"bloom_filter_count":     pd.filter.Count(),
		"bloom_filter_fp_rate":   pd.filter.FalsePositiveProbability(),
		"cache_size":             len(pd.cache),
		"cache_ttl_seconds":      pd.cacheTTL.Seconds(),
		"max_cache_size":         pd.maxCacheSize,
	}
}

func (pd *PacketDeduplicator) GetAllKeys() []string {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	keys := make([]string, 0, len(pd.cache))
	for k := range pd.cache {
		keys = append(keys, k)
	}
	return keys
}

func (pd *PacketDeduplicator) Reset() {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	pd.filter.Reset()
	pd.cache = make(map[string]time.Time)
}

type DedupResult struct {
	IsDuplicate bool   `json:"is_duplicate"`
	PacketID    string `json:"packet_id"`
	Reason      string `json:"reason,omitempty"`
}

func (pd *PacketDeduplicator) CheckPacket(packetID, deviceID string, timestamp time.Time) DedupResult {
	if packetID == "" {
		packetID = GeneratePacketID(deviceID, timestamp, 0)
		return DedupResult{
			IsDuplicate: false,
			PacketID:    packetID,
			Reason:      "packet_id_generated",
		}
	}

	isDup := pd.IsDuplicate(packetID)
	if isDup {
		return DedupResult{
			IsDuplicate: true,
			PacketID:    packetID,
			Reason:      "duplicate_detected",
		}
	}

	return DedupResult{
		IsDuplicate: false,
		PacketID:    packetID,
		Reason:      "new_packet",
	}
}
