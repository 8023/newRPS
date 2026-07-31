package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/doumiao/newRPS/internal/config"
)

// 活动日志：纯文本分表，按 ISO 周目录存放，便于备份与清理，用文本编辑器直接查阅。
// 路径：work/logs/{YY}{WW}/xxx.log  例如 2026 年第 29 周 → work/logs/2629/error.log
//
// 曾经这里是 CSV（xxx.csv），但 CSV 里的字段（如 userAgent、错误信息）完全来自客户端可控
// 输入，一旦被人用 Excel/WPS 之类的电子表格软件打开，以 "="/"+"/"-"/"@" 开头的字段会被当成
// 公式执行（CSV 公式注入）。现在改成每行一条 logfmt 风格的纯文本（key=value，含特殊字符的
// value 加双引号转义），不再是"表格"，不会被电子表格误解析成公式；也不需要 CSV 的转义规则。

var activityLogMu sync.Mutex

// activityLogWeekDir 返回年份后两位 + ISO 周数，如 2629。
func activityLogWeekDir(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%02d%02d", year%100, week)
}

func activityLogBaseDir() string {
	return filepath.Join(config.GetRootDir(), "work", "logs")
}

type activityLogEntry struct {
	table  string
	header []string
	fields []string

	// connEvent 非空时代表这是一条 connections 事件，走 SQLite（activityDB）而不是 CSV，
	// table/header/fields 此时不使用。见 logConnectionEvent。
	connEvent *connectionEventPayload
}

// connectionEventPayload 描述一条已经结束的连接（正常断连 / 优雅关停批量收尾时才会构造），
// 由 logConnectionEvent 入队，runActivityLogConsumer 在后台协程里一次性 INSERT 进
// connection_events。connectedAt 来自 Client.connectedAt 快照，其余字段同理。
type connectionEventPayload struct {
	socketID                                                                           string
	connectedAt, disconnectedAt                                                        int64
	sessionSID, ip, device, fingerprint, userAgent, compression, playerID, closeReason string
}

// logConnectionEvent 非阻塞地把一条已结束的连接事件塞进 logCh，由后台消费者协程串行落盘到
// connection_events 表。调用点在 ws.go/server.go，均在 s.mu 持锁期间执行，这里绝不能同步写库
// （否则 SQLite 单连接的写竞争会让全服因为一次断连卡住，见 README 迁移说明）。
func (s *Server) logConnectionEvent(p connectionEventPayload) {
	entry := activityLogEntry{table: "connections", connEvent: &p}
	if s.logCh != nil {
		select {
		case s.logCh <- entry:
		default:
			// 队列满：丢弃，绝不阻塞业务锁
		}
		return
	}
	s.writeConnectionEvent(&p)
}

// writeConnectionEvent 实际执行 connection_events 的 INSERT（后台协程调用）。
func (s *Server) writeConnectionEvent(p *connectionEventPayload) {
	err := s.activityDB.insertConnectionEvent(
		p.socketID, p.connectedAt, p.disconnectedAt, p.sessionSID, p.ip, p.device,
		p.fingerprint, p.userAgent, p.compression, p.playerID, p.closeReason,
	)
	if err != nil {
		s.errorLog("connection_event_persist_failed", err.Error())
	}
}

// runActivityLogConsumer 后台串行落盘活动日志（由 Run 启动）。
func (s *Server) runActivityLogConsumer() {
	for e := range s.logCh {
		if e.connEvent != nil {
			s.writeConnectionEvent(e.connEvent)
			continue
		}
		writeActivityLog(e.table, e.header, e.fields)
	}
}

// activityLog 向指定表追加一行。table 为文件名（不含扩展名），如 chat / rooms / connections。
// header 仅在新建文件时写入；fields 顺序须与 header 一致。
// 业务路径通常持有 s.mu，这里只做非阻塞入队，磁盘写入交给后台消费者。
func (s *Server) activityLog(table string, header []string, fields []string) {
	if table == "" || len(fields) == 0 {
		return
	}
	if s.logCh != nil {
		select {
		case s.logCh <- activityLogEntry{table: table, header: header, fields: fields}:
		default:
			// 队列满：丢弃这条活动日志，绝不阻塞业务锁
		}
		return
	}
	writeActivityLog(table, header, fields)
}

// writeActivityLog 实际把一行追加进对应 .log 文件（后台协程 / 无队列时的同步兜底路径）。
func writeActivityLog(table string, header []string, fields []string) {
	if table == "" || len(fields) == 0 {
		return
	}
	now := time.Now()
	dir := filepath.Join(activityLogBaseDir(), activityLogWeekDir(now))
	path := filepath.Join(dir, table+".log")

	activityLogMu.Lock()
	defer activityLogMu.Unlock()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	_, _ = f.WriteString(formatLogfmtLine(now, header, fields) + "\n")
	_ = os.Chmod(path, 0o644)
}

// formatLogfmtLine 把 header/fields 对齐后的一行渲染成 logfmt 风格纯文本：
// `2026-07-31T10:23:45+08:00 event=rate_limit ip=1.2.3.4 wsEvent="room:create"`。
// 跳过空字段（CSV 时代靠固定列位置对齐，这里每个字段自带 key，不需要占位）。
func formatLogfmtLine(t time.Time, header []string, fields []string) string {
	var b strings.Builder
	b.WriteString(t.Format(time.RFC3339))
	for i, key := range header {
		if i >= len(fields) || fields[i] == "" {
			continue
		}
		b.WriteByte(' ')
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(logfmtQuote(fields[i]))
	}
	return b.String()
}

// logfmtQuote 视内容决定是否加引号；换行/回车/反斜杠/双引号一律转义——这几个字段
// （userAgent、错误信息等）完全来自客户端可控输入，换行不转义的话，攻击者能在里面
// 塞入伪造的日志行，让一条记录在文本编辑器里看起来像多条。
func logfmtQuote(v string) string {
	needsQuote := false
	for _, r := range v {
		if r <= ' ' || r == '"' || r == '=' {
			needsQuote = true
			break
		}
	}
	v = strings.ReplaceAll(v, "\\", "\\\\")
	v = strings.ReplaceAll(v, "\"", "\\\"")
	v = strings.ReplaceAll(v, "\n", "\\n")
	v = strings.ReplaceAll(v, "\r", "\\r")
	if !needsQuote {
		return v
	}
	return "\"" + v + "\""
}
