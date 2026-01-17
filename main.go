package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"google.golang.org/genai"
)

const DiscordWebhookURL = "WEBHOOK_URL"

type LogEntry struct {
	Message  string
	Source   string
	Priority int
}

var client *genai.Client

func main() {

	setupAI()
	var currentLog LogEntry

	if len(os.Args) > 1 {
		currentLog = getLogInput()
	} else {

		fmt.Println("No terminal input, Reading 'server.log' ...")
		currentLog = readLogData()
	}

	fmt.Println("Analyzing:", currentLog.Message)
	fmt.Printf("Source: %s | Priority Point: %d\n", currentLog.Source, currentLog.Priority)

	if currentLog.Priority < 3 {
		fmt.Println("AI is not required for this action.")
		return
	}

	decision := askGemini(currentLog.Message)
	fmt.Println("Decision:", decision)

	executeAction(decision)
}

func setupAI() {
	var err error
	ctx := context.Background()
	client, err = genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func getLogInput() LogEntry {
	if len(os.Args) < 2 {
		log.Fatal("Argument Error")
	}

	logData := LogEntry{
		Message:  os.Args[1],
		Source:   "Terminal",
		Priority: 3,
	}
	return logData
}

func readLogData() LogEntry {
	fileData, err := os.ReadFile("server.log")
	if err != nil {
		log.Fatal(err)
	}

	logData := LogEntry{
		Message:  string(fileData),
		Source:   "File",
		Priority: 2,
	}
	return logData
}

func askGemini(incomingLog string) string {
	ctx := context.Background()

	prompt := fmt.Sprintf(`
	Act as a DevOps Decision Engine. Analyze the following LOG entry and provide ONLY a single command output.
	
	LOG: "%s"
	
	RULES:
	1. If the log contains keywords "Disk", "Space", "Storage", "Usage", "Capacity" or the "%%" sign -> output "ACTION_CLEAN".
	2. If the log contains "Connection", "Timeout", "Refused", "Dead", "Unresponsive", "Critical" -> output "ACTION_RESTART".
	3. If none of the above -> output "IGNORE".
	
	NOTE: If both Disk and Critical errors are present, "ACTION_RESTART" takes priority.
	Output only the command, do not provide any explanation.
	`, incomingLog)

	resp, err := client.Models.GenerateContent(ctx, "gemini-3-flash-preview", genai.Text(prompt), nil)
	if err != nil {
		log.Fatal("AI Error:", err)
	}
	return resp.Text()
}

func executeAction(decision string) {
	if strings.Contains(decision, "ACTION_RESTART") {
		fmt.Println("Service Restarting...")
		runPlaybook("playbooks/fix_service.yml")

		// YENİ: Bildirim Gönder
		sendDiscordAlert("Critical Error! Used `fix_service.yml` ve service restarting. ✅")

	} else if strings.Contains(decision, "ACTION_CLEAN") {
		fmt.Println("Cleaning Disc...")
		runPlaybook("playbooks/clean_logs.yml")

		sendDiscordAlert("Disc is full. Cleaned using `clean_logs.yml` ")

	} else {
		fmt.Println("✅ No .")
	}
}

func runPlaybook(playbookName string) {
	cmd := exec.Command("ansible-playbook", playbookName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("❌ Error:", err)
	} else {
		fmt.Println("✅ Success.")
	}
}

func sendDiscordAlert(message string) {

	payload := map[string]string{
		"content": "🚨 **InfraMinds Report:** " + message,
	}

	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(DiscordWebhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Println("❌ Can't send Discord notification:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("📨 Sented Discord notification.")
}
