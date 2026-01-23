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

// --- DEĞİŞİKLİK 1: Discord gitti, C# Backend geldi ---
const HQ_API_URL = "http://localhost:5000/api/Alert"

var client *genai.Client

// --- DEĞİŞİKLİK 2: C# Modeli ile Eşleşen Struct ---
type SecurityAlert struct {
	ServiceName string    `json:"serviceName"` // JSON tag'leri C# property'leriyle aynı
	Message     string    `json:"message"`
	ActionTaken string    `json:"actionTaken"`
	Timestamp   time.Time `json:"timestamp"`
}

// Eski yapılarını koruyoruz (LogEntry vs.)
type LogEntry struct {
	Message  string
	Source   string
	Priority int
}

func main() {
	setupAI() // Discord kontrolünü buradan çıkardık

	// ... (Log okuma kısımları aynı kalıyor) ...
	var currentLog LogEntry
	if len(os.Args) > 1 {
		currentLog = getLogInput()
	} else {
		fmt.Println("No input, reading file...")
		currentLog = readLogData()
	}

	fmt.Println("Analyzing:", currentLog.Message)

	if currentLog.Priority < 3 {
		fmt.Println("AI is not required.")
		return
	}

	decision := askGemini(currentLog.Message)
	fmt.Println("Gemini Decision:", decision)

	// --- DEĞİŞİKLİK 3: executeAction artık Discord'a değil Merkeze yazacak ---
	executeAction(decision, currentLog.Message)
}

// Parametre olarak orijinal log mesajını da ekledik ki raporda görünsün
func executeAction(decision string, originalLog string) {

	// ... (Ansible ve karar mantığı aynı kalıyor) ...

	var serviceName string
	var actionTaken string

	if strings.Contains(decision, "ACTION_RESTART") {
		serviceName = "Critical Service"
		actionTaken = "Restart Triggered via Ansible"
		runPlaybook("playbooks/fix_service.yml")
	} else if strings.Contains(decision, "ACTION_CLEAN") {
		serviceName = "Disk Cleaner"
		actionTaken = "Cleanup Triggered via Ansible"
		runPlaybook("playbooks/clean_logs.yml")
	} else {
		serviceName = "System Watcher"
		actionTaken = "No Action / Ignored"
	}

	// --- DEĞİŞİKLİK 4: Discord yerine Raporlama ---
	reportToHQ(serviceName, decision, actionTaken)
}

// --- DEĞİŞİKLİK 5: Yeni Haberleşme Fonksiyonu ---
func reportToHQ(service string, aiMsg string, action string) {
	alert := SecurityAlert{
		ServiceName: service,
		Message:     aiMsg,
		ActionTaken: action,
		Timestamp:   time.Now(),
	}

	// JSON'a paketle (Marshalling)
	jsonData, _ := json.Marshal(alert)

	// Postala
	resp, err := http.Post(HQ_API_URL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("⚠️  UYARI: C# Backend'e ulaşılamadı:", err)
		return
	}
	defer resp.Body.Close()
	fmt.Println("📡 Rapor yollandı. Durum:", resp.Status)
}

// ... (setupAI, askGemini, runPlaybook fonksiyonları aynı kalacak, sadece setupAI içindeki Discord check silinecek) ...

// setupAI fonksiyonunun temizlenmiş hali:
func setupAI() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}
	apiKey := os.Getenv("GOOGLE_API_KEY")
	// Discord webhook kontrolünü sildik
	if apiKey == "" {
		log.Fatal("GOOGLE_API_KEY not found!")
	}

	ctx := context.Background()
	client, err = genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
}

// Diğer yardımcı fonksiyonlar (getLogInput, readLogData, askGemini, runPlaybook) senin yazdığın gibi kalabilir.

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

func askForConfirmation() bool {
	var response string
	fmt.Print("\n SECURITY CHECK: Are you sure for this action? [y/N]: ")
	_, err := fmt.Scanln(&response)
	if err != nil || len(response) == 0 {
		return false
	}

	return strings.ToLower(response)[0] == 'y'
}
