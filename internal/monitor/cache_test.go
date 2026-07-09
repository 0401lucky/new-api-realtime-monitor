package monitor

import (
	"sync/atomic"
	"testing"
	"time"
)

type countingSource struct {
	hits atomic.Int32
}

func (c *countingSource) Dashboard(label string, hours int) (DashboardData, error) {
	c.hits.Add(1)
	return DashboardData{Hours: label, Overview: Overview{TotalRecords: hours}}, nil
}
func (c *countingSource) Models(hours int) ([]map[string]string, error) {
	c.hits.Add(1)
	return []map[string]string{{"name": "m"}}, nil
}
func (c *countingSource) ModelLogs(name string, hours int) (map[string]any, error) {
	c.hits.Add(1)
	return map[string]any{"name": name}, nil
}
func (c *countingSource) KeyQuota(key string, hours int) (KeyData, error) {
	c.hits.Add(1)
	return KeyData{}, nil
}
func (c *countingSource) ChannelRecords(id int, hours int) (ChannelData, error) {
	c.hits.Add(1)
	return ChannelData{}, nil
}

func TestCachedSourceHitsInnerOnce(t *testing.T) {
	inner := &countingSource{}
	cached := NewCachedSource(inner, 60)

	for i := 0; i < 3; i++ {
		if _, err := cached.Dashboard("24", 24); err != nil {
			t.Fatal(err)
		}
	}
	if got := inner.hits.Load(); got != 1 {
		t.Fatalf("expected 1 underlying call, got %d", got)
	}
}

func TestCachedSourceExpires(t *testing.T) {
	inner := &countingSource{}
	// ttl=0 表示不缓存，直接返回 inner
	passthrough := NewCachedSource(inner, 0)
	if passthrough != inner {
		t.Fatal("ttl<=0 should return original source")
	}

	// 用极短 TTL 验证过期后重新查询
	cs := &CachedSource{
		inner: inner,
		ttl:   20 * time.Millisecond,
		items: make(map[string]cacheItem),
	}
	if _, err := cs.Models(1); err != nil {
		t.Fatal(err)
	}
	if _, err := cs.Models(1); err != nil {
		t.Fatal(err)
	}
	if got := inner.hits.Load(); got != 1 {
		t.Fatalf("expected cache hit, got hits=%d", got)
	}
	time.Sleep(30 * time.Millisecond)
	if _, err := cs.Models(1); err != nil {
		t.Fatal(err)
	}
	if got := inner.hits.Load(); got != 2 {
		t.Fatalf("expected re-fetch after expiry, got hits=%d", got)
	}
}
