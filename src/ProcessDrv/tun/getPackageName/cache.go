// Package getPackageName 在 Android TUN 模式下根据五元组查询流量来源包名。
// 查询结果带内存缓存，避免 SYN 重传、UDP 多包等场景反复调用 JNI。
package getPackageName

import (
	"fmt"
	"sync"
	"time"
)

const (
	// pkgCacheTTLPositive 命中包名时的缓存时长（连接存活期内五元组通常不变）。
	pkgCacheTTLPositive = 2 * time.Minute
	// pkgCacheTTLNegative 未查到包名时的缓存时长（较短，便于 socket 登记后尽快重试）。
	pkgCacheTTLNegative = 5 * time.Second
	// maxCacheEntries 缓存条数上限，防止长时间运行内存无限增长。
	maxCacheEntries = 2048
)

// cacheItem 单条缓存记录。
type cacheItem struct {
	result Result    // 查询结果（包名 + 是否已在 PackageManager 注册）
	expire time.Time // 过期时间
}

var (
	pkgCacheMu sync.RWMutex
	pkgCache   = make(map[string]cacheItem) // key 见 cacheKey
)

// cacheKey 生成五元组缓存键：protocol|srcIp|srcPort|dstIp|dstPort。
// src 为 TUN 上看到的客户端（应用侧），dst 为目标服务器。
func cacheKey(protocol int, srcIp string, srcPort int, dstIp string, dstPort int) string {
	return fmt.Sprintf("%d|%s|%d|%s|%d", protocol, srcIp, srcPort, dstIp, dstPort)
}

// getFromCache 读取缓存；第二个返回值为 true 表示命中且未过期。
func getFromCache(protocol int, srcIp string, srcPort int, dstIp string, dstPort int) (Result, bool) {
	key := cacheKey(protocol, srcIp, srcPort, dstIp, dstPort)
	now := time.Now()

	pkgCacheMu.RLock()
	item, ok := pkgCache[key]
	pkgCacheMu.RUnlock()
	if !ok {
		return Result{}, false
	}
	if now.After(item.expire) {
		// 惰性删除过期项，避免后台清扫协程
		pkgCacheMu.Lock()
		if cur, still := pkgCache[key]; still && now.After(cur.expire) {
			delete(pkgCache, key)
		}
		pkgCacheMu.Unlock()
		return Result{}, false
	}
	return item.result, true
}

// putCache 写入缓存；有包名与无包名使用不同 TTL。
func putCache(protocol int, srcIp string, srcPort int, dstIp string, dstPort int, r Result) {
	ttl := pkgCacheTTLNegative
	if r.Name != "" {
		ttl = pkgCacheTTLPositive
	}
	key := cacheKey(protocol, srcIp, srcPort, dstIp, dstPort)
	item := cacheItem{result: r, expire: time.Now().Add(ttl)}

	pkgCacheMu.Lock()
	pkgCache[key] = item
	if len(pkgCache) > maxCacheEntries {
		evictExpiredLocked(time.Now())
		// 仍超限则整表重建（极端流量下的兜底）
		if len(pkgCache) > maxCacheEntries {
			pkgCache = make(map[string]cacheItem)
		}
	}
	pkgCacheMu.Unlock()
}

// evictExpiredLocked 删除所有已过期项；调用方需已持有 pkgCacheMu 写锁。
func evictExpiredLocked(now time.Time) {
	for k, v := range pkgCache {
		if now.After(v.expire) {
			delete(pkgCache, k)
		}
	}
}

// ClearPackageNameCache 清空五元组包名查询缓存。
// 建议在 VPN 重启、OnTunSetFd 或切换用户后调用，避免沿用过期归属信息。
func ClearPackageNameCache() {
	pkgCacheMu.Lock()
	pkgCache = make(map[string]cacheItem)
	pkgCacheMu.Unlock()
}

// GetRequestPackageName 按五元组查询来源包名。
//
// 参数：
//   - protocol：IP 协议号，TCP=6，UDP=17
//   - srcIp/srcPort：TUN 包中的源地址（VPN 内应用侧，如 10.0.0.2:port）
//   - dstIp/dstPort：TUN 包中的目的地址
//
// 先查缓存，未命中则调用 queryPackageName（Android 下走 ConnectivityManager JNI）。
func GetRequestPackageName(protocol int, srcIp string, srcPort int, dstIp string, dstPort int) Result {
	if r, ok := getFromCache(protocol, srcIp, srcPort, dstIp, dstPort); ok {
		return r
	}
	r := queryPackageName(protocol, srcIp, srcPort, dstIp, dstPort)
	if r.Name == "" {
		return r
	}
	putCache(protocol, srcIp, srcPort, dstIp, dstPort, r)
	return r
}
