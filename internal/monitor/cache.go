package monitor

import (
	"fmt"
	"sync"
	"time"
)

// CachedSource 为 DataSource 增加短时内存缓存，减轻 logs 聚合压力。
type CachedSource struct {
	inner DataSource
	ttl   time.Duration
	mu    sync.RWMutex
	items map[string]cacheItem
}

type cacheItem struct {
	expire time.Time
	value  any
}

func NewCachedSource(inner DataSource, ttlSeconds int) DataSource {
	if ttlSeconds <= 0 {
		return inner
	}
	return &CachedSource{
		inner: inner,
		ttl:   time.Duration(ttlSeconds) * time.Second,
		items: make(map[string]cacheItem),
	}
}

func (c *CachedSource) get(key string) (any, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(item.expire) {
		return nil, false
	}
	return item.value, true
}

func (c *CachedSource) set(key string, value any) {
	c.mu.Lock()
	c.items[key] = cacheItem{expire: time.Now().Add(c.ttl), value: value}
	// 简单清理：条目过多时清掉过期项
	if len(c.items) > 256 {
		now := time.Now()
		for k, v := range c.items {
			if now.After(v.expire) {
				delete(c.items, k)
			}
		}
	}
	c.mu.Unlock()
}

func (c *CachedSource) Dashboard(label string, hours int) (DashboardData, error) {
	key := fmt.Sprintf("dashboard:%s:%d", label, hours)
	if v, ok := c.get(key); ok {
		return v.(DashboardData), nil
	}
	data, err := c.inner.Dashboard(label, hours)
	if err != nil {
		return DashboardData{}, err
	}
	c.set(key, data)
	return data, nil
}

func (c *CachedSource) Models(hours int) ([]map[string]string, error) {
	key := fmt.Sprintf("models:%d", hours)
	if v, ok := c.get(key); ok {
		return v.([]map[string]string), nil
	}
	data, err := c.inner.Models(hours)
	if err != nil {
		return nil, err
	}
	c.set(key, data)
	return data, nil
}

func (c *CachedSource) ModelLogs(name string, hours int) (map[string]any, error) {
	key := fmt.Sprintf("model_logs:%s:%d", name, hours)
	if v, ok := c.get(key); ok {
		return v.(map[string]any), nil
	}
	data, err := c.inner.ModelLogs(name, hours)
	if err != nil {
		return nil, err
	}
	c.set(key, data)
	return data, nil
}

func (c *CachedSource) KeyQuota(key string, hours int) (KeyData, error) {
	cacheKey := fmt.Sprintf("key:%s:%d", key, hours)
	if v, ok := c.get(cacheKey); ok {
		return v.(KeyData), nil
	}
	data, err := c.inner.KeyQuota(key, hours)
	if err != nil {
		return KeyData{}, err
	}
	c.set(cacheKey, data)
	return data, nil
}

func (c *CachedSource) ChannelRecords(id int, hours int) (ChannelData, error) {
	cacheKey := fmt.Sprintf("channel:%d:%d", id, hours)
	if v, ok := c.get(cacheKey); ok {
		return v.(ChannelData), nil
	}
	data, err := c.inner.ChannelRecords(id, hours)
	if err != nil {
		return ChannelData{}, err
	}
	c.set(cacheKey, data)
	return data, nil
}
