package transform

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/zeromicro/go-zero/core/logx"
)

// LookupResolver 查找表解析器
// 职责:
// 1. 解析lookup_table JSON配置
// 2. 执行值映射转换
// 3. 缓存查找表以提升性能
type LookupResolver struct {
	logger logx.Logger

	// 缓存配置
	enableCache bool
	cacheTTL    time.Duration

	// 查找表缓存
	cacheMu sync.RWMutex
	cache   map[uint64]*lookupCacheEntry // key: mapping_id
}

// lookupCacheEntry 缓存条目
type lookupCacheEntry struct {
	table      LookupTable
	cachedAt   time.Time
	expiresAt  time.Time
}

// NewLookupResolver 创建查找表解析器
func NewLookupResolver(enableCache bool, cacheTTL time.Duration, logger logx.Logger) *LookupResolver {
	return &LookupResolver{
		logger:      logger,
		enableCache: enableCache,
		cacheTTL:    cacheTTL,
		cache:       make(map[uint64]*lookupCacheEntry),
	}
}

// Resolve 执行查找表转换
func (r *LookupResolver) Resolve(sourceValue interface{}, mapping *ent.FieldMapping) (interface{}, error) {
	if mapping.LookupTable == "" {
		return sourceValue, nil
	}

	// 转换sourceValue为字符串（lookup key）
	key := fmt.Sprintf("%v", sourceValue)

	// 获取查找表
	lookupTable, err := r.getLookupTable(mapping)
	if err != nil {
		return nil, fmt.Errorf("failed to get lookup table: %w", err)
	}

	// 查找映射值
	if targetValue, exists := lookupTable[key]; exists {
		r.logger.Debugw("Lookup resolved",
			logx.Field("mapping_id", mapping.ID),
			logx.Field("source_value", sourceValue),
			logx.Field("target_value", targetValue))
		return targetValue, nil
	}

	// 如果找不到，使用默认值或返回原值
	if mapping.DefaultValue != "" {
		r.logger.Debugw("Lookup not found, using default value",
			logx.Field("mapping_id", mapping.ID),
			logx.Field("source_value", sourceValue),
			logx.Field("default_value", mapping.DefaultValue))
		return mapping.DefaultValue, nil
	}

	// 如果不允许null且没有默认值，返回错误
	if !mapping.AllowNull {
		return nil, fmt.Errorf("lookup failed: key '%s' not found in lookup table", key)
	}

	// 允许null，返回原值
	r.logger.Debugw("Lookup not found, returning source value",
		logx.Field("mapping_id", mapping.ID),
		logx.Field("source_value", sourceValue))
	return sourceValue, nil
}

// getLookupTable 获取查找表（带缓存）
func (r *LookupResolver) getLookupTable(mapping *ent.FieldMapping) (LookupTable, error) {
	// 如果启用了缓存，先尝试从缓存获取
	if r.enableCache {
		if table := r.getFromCache(mapping.ID); table != nil {
			return *table, nil
		}
	}

	// 解析查找表
	table, err := r.parseLookupTable(mapping.LookupTable)
	if err != nil {
		return nil, err
	}

	// 存入缓存
	if r.enableCache {
		r.putToCache(mapping.ID, table)
	}

	return table, nil
}

// parseLookupTable 解析查找表JSON
func (r *LookupResolver) parseLookupTable(jsonStr string) (LookupTable, error) {
	var table LookupTable
	if err := json.Unmarshal([]byte(jsonStr), &table); err != nil {
		return nil, fmt.Errorf("failed to parse lookup table: %w", err)
	}
	return table, nil
}

// getFromCache 从缓存获取查找表
func (r *LookupResolver) getFromCache(mappingID uint64) *LookupTable {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	entry, exists := r.cache[mappingID]
	if !exists {
		return nil
	}

	// 检查是否过期
	if time.Now().After(entry.expiresAt) {
		// 过期了，但不在这里删除（避免写锁）
		return nil
	}

	r.logger.Debugw("Lookup table cache hit",
		logx.Field("mapping_id", mappingID))
	return &entry.table
}

// putToCache 将查找表存入缓存
func (r *LookupResolver) putToCache(mappingID uint64, table LookupTable) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	now := time.Now()
	r.cache[mappingID] = &lookupCacheEntry{
		table:     table,
		cachedAt:  now,
		expiresAt: now.Add(r.cacheTTL),
	}

	r.logger.Debugw("Lookup table cached",
		logx.Field("mapping_id", mappingID),
		logx.Field("expires_at", now.Add(r.cacheTTL)))
}

// ClearCache 清除所有缓存
func (r *LookupResolver) ClearCache() {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	r.cache = make(map[uint64]*lookupCacheEntry)
	r.logger.Info("Lookup table cache cleared")
}

// ClearCacheForMapping 清除指定映射的缓存
func (r *LookupResolver) ClearCacheForMapping(mappingID uint64) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	delete(r.cache, mappingID)
	r.logger.Debugw("Lookup table cache cleared for mapping",
		logx.Field("mapping_id", mappingID))
}

// CleanExpiredCache 清理过期缓存
func (r *LookupResolver) CleanExpiredCache() {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	now := time.Now()
	expiredCount := 0

	for mappingID, entry := range r.cache {
		if now.After(entry.expiresAt) {
			delete(r.cache, mappingID)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		r.logger.Infow("Cleaned expired lookup table cache",
			logx.Field("expired_count", expiredCount))
	}
}

// GetCacheStats 获取缓存统计信息
func (r *LookupResolver) GetCacheStats() CacheStats {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	now := time.Now()
	activeCount := 0
	expiredCount := 0

	for _, entry := range r.cache {
		if now.After(entry.expiresAt) {
			expiredCount++
		} else {
			activeCount++
		}
	}

	return CacheStats{
		TotalEntries:   len(r.cache),
		ActiveEntries:  activeCount,
		ExpiredEntries: expiredCount,
	}
}

// CacheStats 缓存统计信息
type CacheStats struct {
	TotalEntries   int
	ActiveEntries  int
	ExpiredEntries int
}

// ResolveBatch 批量查找
func (r *LookupResolver) ResolveBatch(sourceValues []interface{}, mapping *ent.FieldMapping) ([]interface{}, error) {
	results := make([]interface{}, len(sourceValues))

	// 获取查找表（只解析一次）
	lookupTable, err := r.getLookupTable(mapping)
	if err != nil {
		return nil, fmt.Errorf("failed to get lookup table: %w", err)
	}

	// 批量查找
	for i, sourceValue := range sourceValues {
		key := fmt.Sprintf("%v", sourceValue)

		if targetValue, exists := lookupTable[key]; exists {
			results[i] = targetValue
		} else if mapping.DefaultValue != "" {
			results[i] = mapping.DefaultValue
		} else if !mapping.AllowNull {
			return nil, fmt.Errorf("lookup failed at index %d: key '%s' not found", i, key)
		} else {
			results[i] = sourceValue
		}
	}

	return results, nil
}

// ReverseResolve 反向查找（从目标值查找源值）
func (r *LookupResolver) ReverseResolve(targetValue interface{}, mapping *ent.FieldMapping) (interface{}, error) {
	if mapping.LookupTable == "" {
		return targetValue, nil
	}

	// 获取查找表
	lookupTable, err := r.getLookupTable(mapping)
	if err != nil {
		return nil, fmt.Errorf("failed to get lookup table: %w", err)
	}

	// 反向查找
	targetStr := fmt.Sprintf("%v", targetValue)
	for sourceKey, mappedValue := range lookupTable {
		if mappedValue == targetStr {
			r.logger.Debugw("Reverse lookup resolved",
				logx.Field("mapping_id", mapping.ID),
				logx.Field("target_value", targetValue),
				logx.Field("source_value", sourceKey))
			return sourceKey, nil
		}
	}

	// 找不到
	if !mapping.AllowNull {
		return nil, fmt.Errorf("reverse lookup failed: value '%s' not found in lookup table", targetStr)
	}

	return targetValue, nil
}

// BuildLookupTable 构建查找表（工具方法）
func BuildLookupTable(pairs map[string]string) string {
	bytes, _ := json.Marshal(pairs)
	return string(bytes)
}

// ValidateLookupTable 验证查找表格式（工具方法）
func ValidateLookupTable(jsonStr string) error {
	var table LookupTable
	if err := json.Unmarshal([]byte(jsonStr), &table); err != nil {
		return fmt.Errorf("invalid lookup table JSON: %w", err)
	}

	if len(table) == 0 {
		return fmt.Errorf("lookup table is empty")
	}

	return nil
}

// MergeLookupTables 合并多个查找表（工具方法）
func MergeLookupTables(tables ...LookupTable) LookupTable {
	merged := make(LookupTable)
	for _, table := range tables {
		for k, v := range table {
			merged[k] = v
		}
	}
	return merged
}
