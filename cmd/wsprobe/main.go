package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/coder/websocket"
	"github.com/doumiao/newRPS/internal/wire"
	"google.golang.org/protobuf/proto"
)

func main() {
	resp, err := http.Post("http://127.0.0.1:9988/api/session", "application/json", nil)
	must(err)
	var sess struct {
		Token string `json:"token"`
	}
	must(json.NewDecoder(resp.Body).Decode(&sess))
	resp.Body.Close()

	u, _ := url.Parse("ws://127.0.0.1:9988/ws?token=" + url.QueryEscape(sess.Token))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, u.String(), nil)
	must(err)
	defer conn.Close(websocket.StatusNormalClosure, "")

	for i := 0; i < 2; i++ {
		_, data, err := conn.Read(ctx)
		must(err)
		var env wire.Envelope
		must(proto.Unmarshal(data, &env))
		fmt.Printf("push event=%s kind=%v channel=%s full=%d ops=%d\n", env.Event, env.Kind, env.Channel, len(env.Full), len(env.Ops))
		if env.Event == "lobby:update" && env.Kind == wire.PayloadKind_KIND_FULL {
			var lobby map[string]any
			must(json.Unmarshal(env.Full, &lobby))
			players, _ := lobby["players"].([]any)
			fmt.Printf("lobby players=%d\n", len(players))
			if len(players) > 0 {
				p0 := players[0].(map[string]any)
				if _, ok := p0["giveawayVoteCount"]; ok {
					fmt.Println("FAIL private field present")
					os.Exit(1)
				}
			}
		}
	}

	raw, _ := json.Marshal(map[string]any{
		"name": "流量测试", "genderId": "male",
		"playerId": "pid-probe-x", "playerSecret": "sec-probe-x-sec-probe-x",
	})
	req, _ := proto.Marshal(&wire.Envelope{Event: "player:join", Id: 1, Kind: wire.PayloadKind_KIND_RAW, Raw: raw})
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
		fmt.Printf("side event=%s kind=%v ops=%d full=%d\n", env.Event, env.Kind, len(env.Ops), len(env.Full))
	}
	// wait lobby delta
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
		fmt.Printf("after event=%s kind=%v ops=%d full=%d\n", env.Event, env.Kind, len(env.Ops), len(env.Full))
		if env.Event == "lobby:update" && env.Kind == wire.PayloadKind_KIND_DELTA {
			fmt.Println("OK DELTA lobby")
			break
		}
	}
	fmt.Println("SMOKE OK")
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
