// Package delta 提供通用 JSON 状态树的全量/增量同步原语。
// 广播侧：Diff + Hash；接收侧：Apply + 校验 Hash。
package delta

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"sort"
	"strconv"
	"strings"
)

// Op 表示一条路径级补丁（RFC6901 JSON Pointer）。
type Op struct {
	Path   string          `json:"path"`
	Value  json.RawMessage `json:"value,omitempty"`
	Remove bool            `json:"remove,omitempty"`
}

// Snapshot 将任意可 JSON 序列化的值规范化为 map/slice 树。
func Snapshot(v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Hash 对规范化树做稳定序列化后 CRC-32（IEEE），用于状态一致性探针（非密码学）。
// 输出 8 位小写 hex，与前端 crc32Hex 对齐。
func Hash(doc any) (string, error) {
	canon, err := canonicalize(doc)
	if err != nil {
		return "", err
	}
	sum := crc32.ChecksumIEEE(canon)
	return fmt.Sprintf("%08x", sum), nil
}

// MustHash 与 Hash 相同，失败返回空串。
func MustHash(doc any) string {
	h, err := Hash(doc)
	if err != nil {
		return ""
	}
	return h
}

// Diff 计算 from→to 的补丁列表。from 为 nil 时返回单条根替换（调用方应改用 FULL）。
func Diff(from, to any) ([]Op, error) {
	if from == nil {
		raw, err := marshalNoHTMLEscape(to)
		if err != nil {
			return nil, err
		}
		return []Op{{Path: "", Value: raw}}, nil
	}
	var ops []Op
	diffValue("", from, to, &ops)
	return ops, nil
}

// Apply 将 ops 应用到 base 的深拷贝上，返回新文档。
func Apply(base any, ops []Op) (any, error) {
	doc, err := deepCopy(base)
	if err != nil {
		return nil, err
	}
	for _, op := range ops {
		if op.Path == "" {
			if op.Remove {
				doc = nil
				continue
			}
			var v any
			if err := json.Unmarshal(op.Value, &v); err != nil {
				return nil, fmt.Errorf("root value: %w", err)
			}
			doc = v
			continue
		}
		if err := applyOne(doc, op); err != nil {
			return nil, fmt.Errorf("apply %s: %w", op.Path, err)
		}
	}
	return doc, nil
}

// ToJSON 序列化文档（不转义 HTML，与浏览器 JSON.stringify 对齐，避免哈希不一致）。
func ToJSON(doc any) ([]byte, error) {
	return marshalNoHTMLEscape(doc)
}

// FromJSON 反序列化文档。
func FromJSON(b []byte) (any, error) {
	var doc any
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func deepCopy(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func canonicalize(v any) ([]byte, error) {
	return marshalNoHTMLEscape(normalize(v))
}

// marshalNoHTMLEscape 关闭 encoding/json 默认的 &<> HTML 转义，
// 否则服务端哈希与前端 JSON.stringify 对含 & 的名字会永久不一致，触发无限 sync:full。
func marshalNoHTMLEscape(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	b := buf.Bytes()
	// Encode 会追加 '\n'
	if n := len(b); n > 0 && b[n-1] == '\n' {
		b = b[:n-1]
	}
	return b, nil
}

// normalize 排序 map key，保证哈希稳定。
func normalize(v any) any {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		// json.Marshal map 本身不保证顺序；用有序切片包装不可行。
		// 标准库 encoding/json 对 map 会按 key 排序输出，所以直接返回 map 即可。
		out := make(map[string]any, len(t))
		for _, k := range keys {
			out[k] = normalize(t[k])
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i := range t {
			out[i] = normalize(t[i])
		}
		return out
	default:
		return v
	}
}

// equalJSON 比较两棵 JSON 树是否语义相等。
// 不调用 normalize：map 键序不影响相等性，递归浅层类型分派即可；
// 旧实现每节点 normalize+DeepEqual 会使 Diff 接近 O(N²)。
func equalJSON(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			ov, ok := bv[k]
			if !ok || !equalJSON(v, ov) {
				return false
			}
		}
		return true
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !equalJSON(av[i], bv[i]) {
				return false
			}
		}
		return true
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case float64:
		bv, ok := b.(float64)
		return ok && av == bv
	case json.Number:
		switch bv := b.(type) {
		case json.Number:
			return av == bv
		case float64:
			af, err := av.Float64()
			return err == nil && af == bv
		case int64:
			ai, err := av.Int64()
			return err == nil && ai == bv
		default:
			return false
		}
	case int:
		return scalarNumberEqual(float64(av), b)
	case int64:
		return scalarNumberEqual(float64(av), b)
	case int32:
		return scalarNumberEqual(float64(av), b)
	case uint64:
		return scalarNumberEqual(float64(av), b)
	default:
		// JSON 树几乎不会落到这里；不相等则 Diff 整段替换，语义仍正确。
		return false
	}
}

func scalarNumberEqual(af float64, b any) bool {
	switch bv := b.(type) {
	case float64:
		return af == bv
	case int:
		return af == float64(bv)
	case int64:
		return af == float64(bv)
	case int32:
		return af == float64(bv)
	case uint64:
		return af == float64(bv)
	case json.Number:
		bf, err := bv.Float64()
		return err == nil && af == bf
	default:
		return false
	}
}

func diffValue(path string, from, to any, ops *[]Op) {
	// 类型不同或一方为标量：整体替换
	fromMap, fromIsMap := from.(map[string]any)
	toMap, toIsMap := to.(map[string]any)
	if fromIsMap && toIsMap {
		// 删除
		for k := range fromMap {
			if _, ok := toMap[k]; !ok {
				*ops = append(*ops, Op{Path: joinPath(path, k), Remove: true})
			}
		}
		// 新增/修改（容器不再做整树 equalJSON 预检，直接按 key 下钻，未变叶子自然无 op）
		for k, tv := range toMap {
			fv, ok := fromMap[k]
			if !ok {
				raw, _ := marshalNoHTMLEscape(tv)
				*ops = append(*ops, Op{Path: joinPath(path, k), Value: raw})
				continue
			}
			diffValue(joinPath(path, k), fv, tv, ops)
		}
		return
	}
	fromArr, fromIsArr := from.([]any)
	toArr, toIsArr := to.([]any)
	if fromIsArr && toIsArr {
		// 数组：等长则按 index 局部 patch；变长则整段替换（更稳）
		if len(fromArr) == len(toArr) {
			for i := range toArr {
				diffValue(joinPath(path, strconv.Itoa(i)), fromArr[i], toArr[i], ops)
			}
			return
		}
		raw, _ := marshalNoHTMLEscape(to)
		*ops = append(*ops, Op{Path: path, Value: raw})
		return
	}
	// 叶子 / 类型变化：相等则无 op
	if equalJSON(from, to) {
		return
	}
	raw, _ := marshalNoHTMLEscape(to)
	*ops = append(*ops, Op{Path: path, Value: raw})
}

func joinPath(base, key string) string {
	if base == "" {
		return "/" + escapeKey(key)
	}
	return base + "/" + escapeKey(key)
}

func escapeKey(k string) string {
	k = strings.ReplaceAll(k, "~", "~0")
	k = strings.ReplaceAll(k, "/", "~1")
	return k
}

func unescapeKey(k string) string {
	k = strings.ReplaceAll(k, "~1", "/")
	k = strings.ReplaceAll(k, "~0", "~")
	return k
}

func splitPointer(path string) []string {
	if path == "" || path == "/" {
		return nil
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	parts := strings.Split(path[1:], "/")
	for i := range parts {
		parts[i] = unescapeKey(parts[i])
	}
	return parts
}

func applyOne(doc any, op Op) error {
	parts := splitPointer(op.Path)
	if len(parts) == 0 {
		return fmt.Errorf("empty path")
	}
	// 导航到父节点
	var cur any = doc
	for i := 0; i < len(parts)-1; i++ {
		key := parts[i]
		switch node := cur.(type) {
		case map[string]any:
			next, ok := node[key]
			if !ok || next == nil {
				// 自动创建中间 map
				m := map[string]any{}
				node[key] = m
				cur = m
				continue
			}
			cur = next
		case []any:
			idx, err := strconv.Atoi(key)
			if err != nil || idx < 0 || idx >= len(node) {
				return fmt.Errorf("bad index %s", key)
			}
			cur = node[idx]
		default:
			return fmt.Errorf("cannot descend into %T at %s", cur, key)
		}
	}
	last := parts[len(parts)-1]
	switch parent := cur.(type) {
	case map[string]any:
		if op.Remove {
			delete(parent, last)
			return nil
		}
		var v any
		if err := json.Unmarshal(op.Value, &v); err != nil {
			return err
		}
		parent[last] = v
		return nil
	case []any:
		idx, err := strconv.Atoi(last)
		if err != nil {
			return err
		}
		if op.Remove {
			if idx < 0 || idx >= len(parent) {
				return nil
			}
			// 无法原地截断外层引用；数组 remove 改为置 null
			parent[idx] = nil
			return nil
		}
		var v any
		if err := json.Unmarshal(op.Value, &v); err != nil {
			return err
		}
		if idx >= 0 && idx < len(parent) {
			parent[idx] = v
			return nil
		}
		return fmt.Errorf("array index out of range %d", idx)
	default:
		return fmt.Errorf("parent is %T", cur)
	}
}
