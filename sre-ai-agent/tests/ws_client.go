package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	gorilla "github.com/gorilla/websocket"
)

func main() {
	url := "ws://localhost:8080/api/v1/analyze/ws"
	if len(os.Args) > 1 {
		url = os.Args[1]
	}

	conn, _, err := gorilla.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	req := map[string]any{
		"logs": `2025/06/25 10:00:01 [error] 1234#1234: *1 upstream timed out while connecting to upstream, client: 10.0.0.1, server: example.com, upstream: "http://127.0.0.1:3000"
2025/06/25 10:00:02 [error] 1234#1234: *2 upstream timed out while connecting to upstream, client: 10.0.0.2, server: example.com, upstream: "http://127.0.0.1:3000"
2025/06/25 10:00:03 [error] 1234#1234: *3 upstream timed out while connecting to upstream, client: 10.0.0.3, server: example.com, upstream: "http://127.0.0.1:3000"`,
		"codebase_path": "tests/data/code/sample-app",
		"top_k":         3,
	}

	if err := conn.WriteJSON(req); err != nil {
		log.Fatalf("write: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("read error (stream ended): %v", err)
			return
		}

		var evt map[string]any
		if err := json.Unmarshal(msg, &evt); err != nil {
			log.Printf("parse error: %v", err)
			continue
		}

		typ, _ := evt["type"].(string)
		switch typ {
		case "progress":
			stage, _ := evt["stage"].(string)
			pct, _ := evt["pct"].(float64)
			fmt.Printf("[%s] %s (%d%%)\n", time.Now().Format("15:04:05"), stage, int(pct))
		case "incident":
			fmt.Printf("[%s] incident detected\n", time.Now().Format("15:04:05"))
		case "hypothesis":
			fmt.Printf("[%s] hypothesis received\n", time.Now().Format("15:04:05"))
		case "fix":
			fmt.Printf("[%s] fix received\n", time.Now().Format("15:04:05"))
		case "error":
			errMsg, _ := evt["error"].(string)
			fmt.Printf("[%s] ERROR: %s\n", time.Now().Format("15:04:05"), errMsg)
			return
		case "result":
			fmt.Printf("[%s] analysis complete!\n", time.Now().Format("15:04:05"))
			data, _ := json.MarshalIndent(evt["data"], "", "  ")
			fmt.Println(string(data))
			return
		case "done":
			fmt.Printf("[%s] done signal received\n", time.Now().Format("15:04:05"))
		default:
			fmt.Printf("[%s] unknown event: %s\n", time.Now().Format("15:04:05"), typ)
		}
	}
}
