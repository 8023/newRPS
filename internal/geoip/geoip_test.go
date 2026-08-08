package geoip

import (
	"os"
	"path/filepath"
	"testing"
)

// repoXdbDir 从当前包目录向上找到仓库根下的 config/xdb（xdb 文件不入 git，本地缺失时跳过）。
func repoXdbDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "config", "xdb")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	t.Skip("config/xdb not found (xdb not fetched locally)")
	return ""
}

func TestInitAndLookupV4(t *testing.T) {
	xdbDir := repoXdbDir(t)
	path := filepath.Join(xdbDir, "ip2region_v4.xdb")
	if _, err := os.Stat(path); err != nil {
		t.Skip("ip2region_v4.xdb not fetched locally, run npm run fetch-geoip")
	}
	if err := Init(path); err != nil {
		t.Fatalf("Init(v4) failed: %v", err)
	}
	defer Disable()

	if !Enabled() {
		t.Fatal("Enabled() should be true after Init")
	}
	// 114.114.114.114：114DNS，公开可查的中国大陆出口 IP。字段顺序是
	// 国家|省份|城市|ISP|iso-code——固定断言 Province="江苏省"、City="南京市"，
	// 防止 parseRegion 的字段索引再次错位（曾经把「城市」错当成「省份」）。
	r := Lookup("114.114.114.114")
	if r.Country != "中国" {
		t.Fatalf("expected country=中国 for a public IPv4, got %+v", r)
	}
	if r.Province != "江苏省" {
		t.Fatalf("expected province=江苏省 for 114.114.114.114, got %+v", r)
	}
	if r.City != "南京市" {
		t.Fatalf("expected city=南京市 for 114.114.114.114, got %+v", r)
	}
	// 一个已知有真实 ISP 记录（非"0"占位符）的公网 IP，断言 ISP 不是国家两字母简称。
	rISP := Lookup("202.96.134.133")
	if rISP.ISP == "" || rISP.ISP == "CN" {
		t.Fatalf("expected a real ISP name (not the iso-alpha2 code) for 202.96.134.133, got %+v", rISP)
	}
	// 内网/回环应固定返回“本地”，不查表。
	if r := Lookup("127.0.0.1"); r.Country != "本地" {
		t.Fatalf("loopback should resolve to 本地, got %+v", r)
	}
	if r := Lookup("192.168.1.1"); r.Country != "本地" {
		t.Fatalf("private IPv4 should resolve to 本地, got %+v", r)
	}
}

func TestInitV6AndLookup(t *testing.T) {
	xdbDir := repoXdbDir(t)
	v4Path := filepath.Join(xdbDir, "ip2region_v4.xdb")
	v6Path := filepath.Join(xdbDir, "ip2region_v6.xdb")
	if _, err := os.Stat(v4Path); err != nil {
		t.Skip("ip2region_v4.xdb not fetched locally, run npm run fetch-geoip")
	}
	if _, err := os.Stat(v6Path); err != nil {
		t.Skip("ip2region_v6.xdb not fetched locally, run npm run fetch-geoip")
	}
	if err := Init(v4Path); err != nil {
		t.Fatalf("Init(v4) failed: %v", err)
	}
	defer Disable()
	if err := InitV6(v6Path); err != nil {
		t.Fatalf("InitV6 failed: %v", err)
	}

	// 2400:3200::1：阿里云公共 DNS，公开可查的 IPv6 地址。
	r := Lookup("2400:3200::1")
	if r.Country == "" {
		t.Fatalf("expected non-empty country for a public IPv6, got %+v", r)
	}
	// 回环 IPv6 应固定返回“本地”。
	if r := Lookup("::1"); r.Country != "本地" {
		t.Fatalf("IPv6 loopback should resolve to 本地, got %+v", r)
	}
	// v4 库应仍照常工作，未被 v6 初始化影响。
	if r := Lookup("114.114.114.114"); r.Country == "" {
		t.Fatalf("expected v4 lookup to keep working after InitV6, got %+v", r)
	}
}

func TestLookupWithoutInitReturnsZeroValue(t *testing.T) {
	Disable() // 确保没有残留的全局状态（测试间共享包级变量）
	if r := Lookup("114.114.114.114"); r != (Region{}) {
		t.Fatalf("expected zero Region before Init, got %+v", r)
	}
	if Enabled() {
		t.Fatal("Enabled() should be false before Init")
	}
}
