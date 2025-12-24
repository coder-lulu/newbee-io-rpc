package cache

import (
	"context"
	"container/list"
	"fmt"
	"strings"
	"sync"
	"time"
)

// LocalCacheManager 本地缓存管理器
// 基于LRU/LFU算法实现的内存缓存
type LocalCacheManager struct {
	config   *LocalCacheConfig
	items    map[string]*localCacheItem
	lruList  *list.List // LRU链表
	mu       sync.RWMutex
	stopCh   chan struct{}
	cleanupTicker *time.Ticker
}

// LocalCacheConfig 本地缓存配置
type LocalCacheConfig struct {
	MaxSize        int           `json:"maxSize"`        // 最大条目数
	DefaultTTL     time.Duration `json:"defaultTTL"`     // 默认TTL
	EvictionPolicy string        `json:"evictionPolicy"` // 淘汰策略: lru, lfu, fifo
	CleanupInterval time.Duration `json:"cleanupInterval"` // 清理间隔
}

// localCacheItem 本地缓存项
type localCacheItem struct {
	key        string
	value      interface{}
	expireAt   time.Time
	accessCount int64
	createdAt  time.Time
	accessedAt time.Time
	element    *list.Element // LRU链表元素
}

// NewLocalCacheManager 创建本地缓存管理器
func NewLocalCacheManager(config *LocalCacheConfig) *LocalCacheManager {
	if config == nil {
		config = getDefaultLocalCacheConfig()
	}
	
	manager := &LocalCacheManager{
		config:  config,
		items:   make(map[string]*localCacheItem),
		lruList: list.New(),
		stopCh:  make(chan struct{}),
	}
	
	// 启动清理协程
	manager.startCleanup()
	
	return manager
}

// Get 获取缓存项
func (m *LocalCacheManager) Get(ctx context.Context, key string) (interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	item, exists := m.items[key]
	if !exists {
		return nil, fmt.Errorf("缓存项不存在: %s", key)
	}
	
	// 检查是否过期
	if time.Now().After(item.expireAt) {
		m.removeItemUnsafe(key)
		return nil, fmt.Errorf("缓存项已过期: %s", key)
	}
	
	// 更新访问信息
	item.accessCount++
	item.accessedAt = time.Now()
	
	// 更新LRU位置
	if item.element != nil {
		m.lruList.MoveToFront(item.element)
	}
	
	return item.value, nil
}

// Set 设置缓存项
func (m *LocalCacheManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if ttl == 0 {
		ttl = m.config.DefaultTTL
	}
	
	now := time.Now()
	expireAt := now.Add(ttl)
	
	// 检查是否已存在
	if existingItem, exists := m.items[key]; exists {
		// 更新现有项
		existingItem.value = value
		existingItem.expireAt = expireAt
		existingItem.accessedAt = now
		
		// 更新LRU位置
		if existingItem.element != nil {
			m.lruList.MoveToFront(existingItem.element)
		}
		
		return nil
	}
	
	// 检查是否需要淘汰
	m.evictIfNeeded()
	
	// 创建新项
	item := &localCacheItem{
		key:         key,
		value:       value,
		expireAt:    expireAt,
		accessCount: 1,
		createdAt:   now,
		accessedAt:  now,
	}
	
	// 添加到LRU链表
	item.element = m.lruList.PushFront(key)
	
	// 添加到映射
	m.items[key] = item
	
	return nil
}

// Delete 删除缓存项
func (m *LocalCacheManager) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.removeItemUnsafe(key)
	return nil
}

// InvalidatePattern 删除匹配模式的缓存项
func (m *LocalCacheManager) InvalidatePattern(ctx context.Context, pattern string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 收集匹配的键
	var keysToDelete []string
	for key := range m.items {
		if m.matchPattern(key, pattern) {
			keysToDelete = append(keysToDelete, key)
		}
	}
	
	// 删除匹配的项
	for _, key := range keysToDelete {
		m.removeItemUnsafe(key)
	}
	
	return nil
}

// Size 获取缓存大小
func (m *LocalCacheManager) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return len(m.items)
}

// Clear 清空缓存
func (m *LocalCacheManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.items = make(map[string]*localCacheItem)
	m.lruList.Init()
}

// GetStats 获取缓存统计信息
func (m *LocalCacheManager) GetStats() *LocalCacheStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := &LocalCacheStats{
		Size:     len(m.items),
		MaxSize:  m.config.MaxSize,
		HitCount: 0,
		MissCount: 0,
	}
	
	// 计算命中次数
	for _, item := range m.items {
		stats.HitCount += item.accessCount
	}
	
	return stats
}

// Close 关闭缓存管理器
func (m *LocalCacheManager) Close() error {
	close(m.stopCh)
	if m.cleanupTicker != nil {
		m.cleanupTicker.Stop()
	}
	return nil
}

// 私有方法

// removeItemUnsafe 删除缓存项（不加锁）
func (m *LocalCacheManager) removeItemUnsafe(key string) {
	if item, exists := m.items[key]; exists {
		delete(m.items, key)
		
		// 从LRU链表中移除
		if item.element != nil {
			m.lruList.Remove(item.element)
		}
	}
}

// evictIfNeeded 根据需要淘汰缓存项
func (m *LocalCacheManager) evictIfNeeded() {
	if len(m.items) < m.config.MaxSize {
		return
	}
	
	switch m.config.EvictionPolicy {
	case "lru":
		m.evictLRU()
	case "lfu":
		m.evictLFU()
	case "fifo":
		m.evictFIFO()
	default:
		m.evictLRU()
	}
}

// evictLRU LRU淘汰策略
func (m *LocalCacheManager) evictLRU() {
	if m.lruList.Len() == 0 {
		return
	}
	
	// 获取最后一个元素（最久未使用）
	element := m.lruList.Back()
	if element != nil {
		key := element.Value.(string)
		m.removeItemUnsafe(key)
	}
}

// evictLFU LFU淘汰策略
func (m *LocalCacheManager) evictLFU() {
	if len(m.items) == 0 {
		return
	}
	
	var minKey string
	var minCount int64 = -1
	
	for key, item := range m.items {
		if minCount == -1 || item.accessCount < minCount {
			minCount = item.accessCount
			minKey = key
		}
	}
	
	if minKey != "" {
		m.removeItemUnsafe(minKey)
	}
}

// evictFIFO FIFO淘汰策略
func (m *LocalCacheManager) evictFIFO() {
	if len(m.items) == 0 {
		return
	}
	
	var oldestKey string
	var oldestTime time.Time
	
	for key, item := range m.items {
		if oldestTime.IsZero() || item.createdAt.Before(oldestTime) {
			oldestTime = item.createdAt
			oldestKey = key
		}
	}
	
	if oldestKey != "" {
		m.removeItemUnsafe(oldestKey)
	}
}

// matchPattern 检查键是否匹配模式
func (m *LocalCacheManager) matchPattern(key, pattern string) bool {
	// 简单的通配符匹配
	if pattern == "*" {
		return true
	}
	
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(key, prefix)
	}
	
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(key, suffix)
	}
	
	return key == pattern
}

// startCleanup 启动清理协程
func (m *LocalCacheManager) startCleanup() {
	if m.config.CleanupInterval <= 0 {
		return
	}
	
	m.cleanupTicker = time.NewTicker(m.config.CleanupInterval)
	
	go func() {
		for {
			select {
			case <-m.cleanupTicker.C:
				m.cleanup()
			case <-m.stopCh:
				return
			}
		}
	}()
}

// cleanup 清理过期项
func (m *LocalCacheManager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	now := time.Now()
	var expiredKeys []string
	
	// 收集过期的键
	for key, item := range m.items {
		if now.After(item.expireAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}
	
	// 删除过期项
	for _, key := range expiredKeys {
		m.removeItemUnsafe(key)
	}
}

// LocalCacheStats 本地缓存统计信息
type LocalCacheStats struct {
	Size      int   `json:"size"`
	MaxSize   int   `json:"maxSize"`
	HitCount  int64 `json:"hitCount"`
	MissCount int64 `json:"missCount"`
}

// 默认配置
func getDefaultLocalCacheConfig() *LocalCacheConfig {
	return &LocalCacheConfig{
		MaxSize:         1000,
		DefaultTTL:      5 * time.Minute,
		EvictionPolicy:  "lru",
		CleanupInterval: 1 * time.Minute,
	}
}