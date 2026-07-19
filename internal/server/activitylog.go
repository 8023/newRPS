package server

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/doumiao/newRPS/internal/config"
)

// 活动日志：CSV 分表，按 ISO 周目录存放，便于备份与清理。
// 路径：work/logs/{YY}{WW}/xxx.csv  例如 2026 年第 29 周 → work/logs/2629/chat.csv

var (
	activityLogMu     sync.Mutex
	activityLogInited map[string]bool // path → header written
)

func init() {
	activityLogInited = map[string]bool{}
}

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

// writeActivityLog 实际把一行写入对应 CSV（后台协程 / 无队列时的同步兜底路径）。
func writeActivityLog(table string, header []string, fields []string) {
	if table == "" || len(fields) == 0 {
		return
	}
	now := time.Now()
	dir := filepath.Join(activityLogBaseDir(), activityLogWeekDir(now))
	path := filepath.Join(dir, table+".csv")

	activityLogMu.Lock()
	defer activityLogMu.Unlock()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}

	needHeader := !activityLogInited[path]
	if needHeader {
		if st, err := os.Stat(path); err == nil && st.Size() > 0 {
			needHeader = false
			activityLogInited[path] = true
		}
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if needHeader && len(header) > 0 {
		_ = w.Write(header)
		activityLogInited[path] = true
	}
	// 对齐列数
	row := make([]string, len(header))
	for i := range row {
		if i < len(fields) {
			row[i] = fields[i]
		}
	}
	if len(header) == 0 {
		row = fields
	}
	_ = w.Write(row)
	w.Flush()
	_ = os.Chmod(path, 0o644)
}
