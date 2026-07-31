package server

import (
	"strings"
	"testing"
	"time"
)

// TestFormatLogfmtLineSkipsEmptyFieldsAndIncludesTimestamp 之前 error.csv 靠固定列位置对齐
// 空字段（留空占位），logfmt 每个字段自带 key，空字段应该被跳过而不是留下一个裸的 "key="。
func TestFormatLogfmtLineSkipsEmptyFieldsAndIncludesTimestamp(t *testing.T) {
	ts := time.Date(2026, 7, 31, 10, 23, 45, 0, time.UTC)
	line := formatLogfmtLine(ts, []string{"event", "sid", "ip"}, []string{"rate_limit", "", "1.2.3.4"})
	if !strings.HasPrefix(line, ts.Format(time.RFC3339)) {
		t.Fatalf("line should start with RFC3339 timestamp, got %q", line)
	}
	if !strings.Contains(line, "event=rate_limit") {
		t.Fatalf("expected event=rate_limit in line, got %q", line)
	}
	if !strings.Contains(line, "ip=1.2.3.4") {
		t.Fatalf("expected ip=1.2.3.4 in line, got %q", line)
	}
	if strings.Contains(line, "sid=") {
		t.Fatalf("empty field should be skipped entirely, got %q", line)
	}
}

// TestLogfmtQuoteEscapesNewlineToPreventLogInjection userAgent/错误信息等字段完全来自客户端
// 可控输入；如果不转义换行，攻击者可以在里面塞入伪造的 "假日志行"，让一条记录在文本编辑器
// 里看起来像多条、甚至伪造出不存在的事件类型。
func TestLogfmtQuoteEscapesNewlineToPreventLogInjection(t *testing.T) {
	malicious := "Mozilla/5.0\n2026-01-01T00:00:00Z event=admin_login_locked ip=127.0.0.1"
	line := formatLogfmtLine(time.Now(), []string{"userAgent"}, []string{malicious})
	if strings.Contains(line, "\n") {
		t.Fatalf("output line must not contain a raw newline, got %q", line)
	}
	if !strings.Contains(line, "\\n") {
		t.Fatalf("newline should be escaped as literal \\n, got %q", line)
	}
}

// TestLogfmtQuoteDoesNotTreatLeadingEqualsAsFormula 换成纯文本 logfmt 之后，以 "="/"+"/"-"/"@"
// 开头的字段值就是普通文本——不再有 CSV 公式注入的风险面（不会被电子表格软件误当公式执行），
// 这里确认这类值原样出现在输出里，没有做任何"看起来像公式就特殊处理"的操作。
func TestLogfmtQuoteDoesNotTreatLeadingEqualsAsFormula(t *testing.T) {
	formulaLike := "=cmd|'/c calc'!A1"
	line := formatLogfmtLine(time.Now(), []string{"err"}, []string{formulaLike})
	if !strings.Contains(line, formulaLike) {
		t.Fatalf("formula-like value should appear as plain text, got %q", line)
	}
}

// TestLogfmtQuoteWrapsValuesWithSpaces 含空格的字段（如完整 User-Agent 字符串）应该被双引号
// 包裹，保证每行仍然是清晰的 key=value 序列，人眼在文本编辑器里就能对齐着看。
func TestLogfmtQuoteWrapsValuesWithSpaces(t *testing.T) {
	got := logfmtQuote("Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	if !strings.HasPrefix(got, "\"") || !strings.HasSuffix(got, "\"") {
		t.Fatalf("value containing spaces should be quoted, got %q", got)
	}
}

func TestLogfmtQuoteLeavesSimpleValuesBare(t *testing.T) {
	got := logfmtQuote("rate_limit")
	if got != "rate_limit" {
		t.Fatalf("simple value should not be quoted, got %q", got)
	}
}
