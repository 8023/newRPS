// Package geoip 基于 ip2region 离线库解析 IP 归属地（国家/省/市/ISP）。
// 数据文件默认放在 config/xdb/ip2region_v4.xdb（必需）与 config/xdb/ip2region_v6.xdb
// （可选），不嵌入二进制。
package geoip

import (
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

// Region 是一次解析结果。内网/回环时 Country="本地"；解析失败为零值。
type Region struct {
	Country  string
	Province string
	City     string
	ISP      string
}

var (
	mu         sync.Mutex
	searcherV4 *xdb.Searcher
	searcherV6 *xdb.Searcher
	enabled    bool // 与现有语义一致：至少 v4 xdb 就绪即视为「归属地解析已启用」
)

// Init 载入 IPv4 xdb 数据文件到内存（buffer 模式，零文件 I/O）。这是必需库——
// 文件缺失/损坏返回 error，由调用方决定是否致命（见 README 升级说明）。
func Init(path string) error {
	s, err := loadSearcher(xdb.IPv4, path)
	if err != nil {
		return err
	}
	mu.Lock()
	if searcherV4 != nil {
		searcherV4.Close()
	}
	searcherV4 = s
	enabled = true
	mu.Unlock()
	return nil
}

// InitV6 载入可选的 IPv6 xdb。文件缺失/损坏只返回 error 供调用方记非致命日志——
// 不影响已加载的 v4 库，此时 IPv6 地址的 Lookup 会退化为零值 Region。
func InitV6(path string) error {
	s, err := loadSearcher(xdb.IPv6, path)
	if err != nil {
		return err
	}
	mu.Lock()
	if searcherV6 != nil {
		searcherV6.Close()
	}
	searcherV6 = s
	mu.Unlock()
	return nil
}

func loadSearcher(version *xdb.Version, path string) (*xdb.Searcher, error) {
	cBuff, err := xdb.LoadContentFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("load xdb %s: %w", path, err)
	}
	s, err := xdb.NewWithBuffer(version, cBuff)
	if err != nil {
		return nil, fmt.Errorf("open xdb buffer: %w", err)
	}
	return s, nil
}

// Disable 关闭归属地解析（ANALYTICS_GEO_ENABLED=0 时用）。
func Disable() {
	mu.Lock()
	if searcherV4 != nil {
		searcherV4.Close()
		searcherV4 = nil
	}
	if searcherV6 != nil {
		searcherV6.Close()
		searcherV6 = nil
	}
	enabled = false
	mu.Unlock()
}

// Enabled 返回是否已载入可用的 v4 xdb（v6 是否就绪不影响这个总开关）。
func Enabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return enabled && searcherV4 != nil
}

// LookupIfEnabled 在 enabled 为 true 时调用 Lookup，否则返回零值（不查表）。
func LookupIfEnabled(enabled bool, ip string) Region {
	if !enabled {
		return Region{}
	}
	return Lookup(ip)
}

// Lookup 解析 IP 归属地。内网/回环返回 Region{Country: "本地"}；未初始化或失败返回零值。
// IPv4（含 IPv4-mapped IPv6）走 v4 库，纯 IPv6 走 v6 库；v6 库未加载时纯 IPv6 地址返回零值。
func Lookup(ip string) Region {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return Region{}
	}
	// 去掉 IPv4-mapped 前缀与端口残留
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	ip = strings.Trim(ip, "[]")

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return Region{}
	}
	if isPrivateOrLoopback(parsed) {
		return Region{Country: "本地"}
	}

	// Searcher 非线程安全，且 Init/Disable 可能替换或关闭它；把取指针和
	// Search 放在同一把锁内，避免拿到旧 searcher 后被并发 Close。
	mu.Lock()
	defer mu.Unlock()
	if !enabled {
		return Region{}
	}
	s := searcherV4
	if parsed.To4() == nil {
		s = searcherV6
	}
	if s == nil {
		return Region{}
	}
	region, err := s.Search(ip)
	if err != nil || region == "" {
		return Region{}
	}
	return parseRegion(region)
}

// parseRegion 解析 ip2region_v4/v6.xdb 返回串。字段格式是「国家|省份|城市|ISP|
// iso-alpha2-code」（ip2region 项目 README 对 data/ip2region_v4.xdb 的说明——中国的
// 定位信息全部为中文，非中国地区全部为英文），最后一段是国家两字母简称，我们不用。
// 注意：这不是 ip2region 旧版文档里常见的「国家|区域|省份|城市|ISP」5 段格式，
// 早期实现按那套顺序取过 parts[2]/parts[4]，实际会把「城市」错当成「省份」、把
// iso-alpha2-code 错当成「ISP」（因为非中国地区的城市段常年是占位符 "0"，反而让
// 省份看起来是空的）——parts[1]/parts[2]/parts[3] 才分别是真正的省份/城市/ISP。
func parseRegion(raw string) Region {
	parts := strings.Split(raw, "|")
	// 补齐到 5 段
	for len(parts) < 5 {
		parts = append(parts, "")
	}
	clean := func(s string) string {
		s = strings.TrimSpace(s)
		if s == "" || s == "0" {
			return ""
		}
		return s
	}
	return Region{
		Country:  clean(parts[0]),
		Province: clean(parts[1]),
		City:     clean(parts[2]),
		ISP:      clean(parts[3]),
	}
}

func isPrivateOrLoopback(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
		return true
	}
	// IPv4 链路本地 / CGNAT 等已由 IsPrivate 覆盖大部分；额外兜底 0.0.0.0/8
	if v4 := ip.To4(); v4 != nil && v4[0] == 0 {
		return true
	}
	return false
}

// DefaultPath 返回默认 IPv4 xdb 路径（相对 root 的 config/xdb/ip2region_v4.xdb）。
func DefaultPath(root string) string {
	if root == "" {
		return filepath.Join("config", "xdb", "ip2region_v4.xdb")
	}
	return filepath.Join(root, "config", "xdb", "ip2region_v4.xdb")
}

// DefaultPathV6 返回默认 IPv6 xdb 路径（相对 root 的 config/xdb/ip2region_v6.xdb）。
func DefaultPathV6(root string) string {
	if root == "" {
		return filepath.Join("config", "xdb", "ip2region_v6.xdb")
	}
	return filepath.Join(root, "config", "xdb", "ip2region_v6.xdb")
}
