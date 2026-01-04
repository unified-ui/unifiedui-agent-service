package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load() // .env laden (ignoriert Fehler wenn nicht vorhanden)
	webhookURL := os.Getenv("N8N_WEBHOOK_URL")
	if webhookURL == "" {
		fmt.Println("❌ N8N_WEBHOOK_URL environment variable is required")
		os.Exit(1)
	}

	chatInput := "Give me a short recipe for pancakes."
	if len(os.Args) > 1 {
		chatInput = os.Args[1]
	}

	fmt.Printf("📤 Sending to: %s\n", webhookURL)
	fmt.Printf("💬 Message: %s\n\n", chatInput)

	// Send request to n8n
	payload, _ := json.Marshal(map[string]string{"chatInput": chatInput})
	req, _ := http.NewRequest("POST", webhookURL, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("❌ Request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ n8n returned %d: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	// Stream response
	fmt.Println("🤖 Response:")
	scanner := bufio.NewScanner(resp.Body)
	var meta []string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			continue
		}

		if content, ok := data["content"].(string); ok {
			fmt.Print(content)
		} else {
			meta = append(meta, line)
		}
	}

	fmt.Println("\n\n📊 Meta Information:")
	for _, m := range meta {
		fmt.Println(m)
	}
}
