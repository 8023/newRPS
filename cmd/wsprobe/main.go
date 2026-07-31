package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/coder/websocket"
	"github.com/doumiao/newRPS/internal/pbconv"
	"github.com/doumiao/newRPS/internal/wire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func main() {
	// 服务端要求非安全方法（含 WS 升级请求）必须带 Origin，且落在允许列表/同 Host 内，
	// 直连调试工具没有浏览器帮忙自动附加，需要手动设置。
	sessReq, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:9988/api/session", nil)
	must(err)
	sessReq.Header.Set("Content-Type", "application/json")
	sessReq.Header.Set("Origin", "http://127.0.0.1:9988")
	resp, err := http.DefaultClient.Do(sessReq)
	must(err)
	var sess struct {
		Token string `json:"token"`
	}
	must(json.NewDecoder(resp.Body).Decode(&sess))
	resp.Body.Close()
	if sess.Token == "" {
		panic("empty session token (check server Origin allow-list)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// token 通过 Sec-WebSocket-Protocol 传递，不再拼进 URL（见 internal/server/ws.go 的
	// parseWSAuthProtocols）：反向代理访问日志会记录完整请求 URL，token 出现在其中等于
	// 把 24 小时有效的身份凭据写进日志系统。
	conn, _, err := websocket.Dial(ctx, "ws://127.0.0.1:9988/ws", &websocket.DialOptions{
		HTTPHeader:   http.Header{"Origin": []string{"http://127.0.0.1:9988"}},
		Subprotocols: []string{"auth." + sess.Token},
	})
	must(err)
	defer conn.Close(websocket.StatusNormalClosure, "")

	for i := 0; i < 2; i++ {
		_, data, err := conn.Read(ctx)
		must(err)
		var env wire.Envelope
		must(proto.Unmarshal(data, &env))
		ops := 0
		if env.Delta != nil {
			ops = len(env.Delta.Ops)
		}
		hasFull := env.FullState != nil
		fmt.Printf("push event=%s kind=%v channel=%s full=%v ops=%d hash=%s\n",
			env.Event, env.Kind, env.Channel, hasFull, ops, shortHash(env.Hash))
		if env.Event == "lobby:update" && env.Kind == wire.PayloadKind_KIND_FULL && env.FullState != nil {
			tree, err := pbconv.StateDocToFront(env.FullState)
			must(err)
			lobby, _ := tree.(map[string]any)
			players, _ := lobby["players"].([]any)
			fmt.Printf("lobby players=%d\n", len(players))
			if len(players) > 0 {
				p0, _ := players[0].(map[string]any)
				if _, ok := p0["giveawayVoteCount"]; ok {
					fmt.Println("FAIL private field present")
					os.Exit(1)
				}
			}
		}
	}

	st, err := structpb.NewStruct(map[string]any{
		"name": "流量测试", "genderId": "male",
		"playerId": "pid-probe-x", "playerSecret": "sec-probe-x-sec-probe-x",
	})
	must(err)
	req, err := proto.Marshal(&wire.Envelope{
		Event: "player:join", Id: 1, Kind: wire.PayloadKind_KIND_RAW,
		RawBody: &wire.RawBody{Body: &wire.RawBody_Dynamic{Dynamic: st}},
	})
	must(err)
	must(conn.Write(ctx, websocket.MessageBinary, req))
	for {
		_, data, err := conn.Read(ctx)
		must(err)
		var env wire.Envelope
		must(proto.Unmarshal(data, &env))
		if env.Id == 1 {
			fmt.Printf("join err=%q\n", env.Err)
			if env.Err != "" {
				os.Exit(1)
			}
			break
		}
		fmt.Printf("side event=%s kind=%v\n", env.Event, env.Kind)
	}
	// wait lobby delta or full
	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()
	for {
		_, data, err := conn.Read(ctx2)
		if err != nil {
			fmt.Println("no more frames (ok if delta already seen)")
			break
		}
		var env wire.Envelope
		must(proto.Unmarshal(data, &env))
		ops := 0
		if env.Delta != nil {
			ops = len(env.Delta.Ops)
		}
		fmt.Printf("after event=%s kind=%v ops=%d full=%v\n", env.Event, env.Kind, ops, env.FullState != nil)
		if env.Event == "lobby:update" && env.Kind == wire.PayloadKind_KIND_DELTA {
			fmt.Println("OK DELTA lobby")
			break
		}
		if env.Event == "lobby:update" && env.Kind == wire.PayloadKind_KIND_FULL {
			fmt.Println("OK FULL lobby (delta may have been coalesced)")
			break
		}
	}
	fmt.Println("SMOKE OK")
}

func shortHash(h string) string {
	if len(h) <= 12 {
		return h
	}
	return h[:12]
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
