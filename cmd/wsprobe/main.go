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
	"github.com/doumiao/newRPS/internal/pbconv"
	"github.com/doumiao/newRPS/internal/wire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
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
