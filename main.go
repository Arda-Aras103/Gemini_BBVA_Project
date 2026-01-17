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
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/genai"
)

var DiscordWebhookURL string
var client *genai.Client

type LogEntry struct {
	Message  string
	Source   string
	Priority int
}

type IncidentState struct {
	LastAction string    `json:"last_action"`
	Timestamp  time.Time `json:"timestamp"`
	RetryCount int       `json:"retry_count"`
}

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
	err := godotenv.Load()
	if err != nil {
		log.Println("Didn't find .env file")
	}
	apiKey := os.Getenv("GOOGLE_API_KEY")
	DiscordWebhookURL = os.Getenv("DISCORD_WEBHOOK_URL")

	if apiKey == "" || DiscordWebhookURL == "" {
		log.Fatal(" GOOGLE_API_KEY veya DISCORD_WEBHOOK_URL did not find! Check .env file")
	}

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

	state := loadState()
	if time.Since(state.Timestamp) < 5*time.Minute && state.RetryCount >= 3 {
		fmt.Println("CIRCUIT BREAKER OPEN - Automation Halted.")
		sendDiscordAlert("SYSTEM HALTED: Too many retries.")
		return
	}

	fmt.Println("\nACTION REPORT:")

	var playbookFile string
	var actionType string
	var logMessage string

	if strings.Contains(decision, "ACTION_RESTART") {
		playbookFile = "playbooks/fix_service.yml"
		actionType = "RESTART SERVICE"
		logMessage = fmt.Sprintf("🚨 Service Restarted (Attempt: %d)", state.RetryCount+1)

		fmt.Println("Type: CRITICAL INCIDENT")
		fmt.Printf("Proposed Action: %s\n", actionType)

	} else if strings.Contains(decision, "ACTION_CLEAN") {
		playbookFile = "playbooks/clean_logs.yml"
		actionType = "CLEAN DISK"
		logMessage = "Disk Cleaned"

		fmt.Println("Type: MAINTENANCE REQUIRED")
		fmt.Printf("Proposed Action: %s\n", actionType)

	} else {
		fmt.Println("Type: UNKNOWN")
		fmt.Println("Action: No action required.")
		return
	}

	if !askForConfirmation() {
		fmt.Println("❌ Action Cancelled by User.")
		return
	}

	fmt.Println("Executing Playbook...")
	runPlaybook(playbookFile)

	sendDiscordAlert(logMessage)

	if actionType == "RESTART SERVICE" {
		saveState("RESTART", state.RetryCount+1)
	} else {
		saveState("CLEAN", 0)
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

func loadState() IncidentState {
	var state IncidentState
	fileData, err := os.ReadFile("state.json")
	if err != nil {

		return IncidentState{RetryCount: 0}
	}
	json.Unmarshal(fileData, &state)
	return state
}

func saveState(action string, count int) {
	state := IncidentState{
		LastAction: action,
		Timestamp:  time.Now(),
		RetryCount: count,
	}
	data, _ := json.Marshal(state)
	os.WriteFile("state.json", data, 0644)
}

func askForConfirmation() bool {
	var response string
	fmt.Print("\n SECURITY CHECK: Are you sure for this action? [y/N]: ")
	_, err := fmt.Scanln(&response)
	if err != nil || len(response) == 0 {
		return false
	}

	return strings.ToLower(response)[0] == 'y'
}
