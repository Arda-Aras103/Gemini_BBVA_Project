# 🏦 InfraMinds: AI-Powered Self-Healing Infrastructure

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![Gemini](https://img.shields.io/badge/AI-Gemini%202.0-8E75B2?style=flat&logo=google)
![Ansible](https://img.shields.io/badge/Automation-Ansible-EE0000?style=flat&logo=ansible)
![Docker](https://img.shields.io/badge/Infrastructure-Docker-2496ED?style=flat&logo=docker)

**InfraMinds** is an autonomous DevOps agent designed for high-availability banking infrastructure. Powered by **Google Gemini 3 Flash**, it detects critical system failures, analyzes logs semantically, and executes remediation playbooks without human intervention.

> **Project Mission:** Built for the **Gemini API Developer Competition**, this project demonstrates how Google's **GenAI Go SDK** can bridge the gap between AI reasoning and Cloud-Native automation tools like Ansible & Docker.

---

## 🌟 Key Features

### 🧠 1. Semantic Failure Analysis (AI Brain)
Unlike traditional monitoring tools that rely on static Keywords (RegEx), InfraMinds uses **Gemini 3 Flash** to understand the *meaning* of an error.
* *Example:* It understands that "Disk usage is 99%" and "No space left on device" require the same action (`ACTION_CLEAN`).

### 💰 2. Smart Cost Optimization (Priority Filtering)
To optimize API costs and latency, the system uses a custom `LogEntry` struct and logic filter:
* **Low Priority (Score < 3):** Routine logs read from files are analyzed locally but **NOT** sent to the AI API to save costs.
* **High Priority (Score 3):** Critical alerts via CLI are immediately processed by Gemini for rapid resolution.

### ⚡ 3. Autonomous Remediation (Self-Healing)
The agent triggers specific **Ansible Playbooks** based on the AI's decision:
* **Service Down?** → Triggers `fix_service.yml` (Restarts Docker container).
* **Disk Full?** → Triggers `clean_logs.yml` (Clears temporary files).

### 💬 4. ChatOps Integration
Every action taken by the agent is reported in real-time to **Discord** via Webhooks, keeping the operations team in the loop.

---

## 🛠️ Architecture & Tech Stack

| Component | Technology | Responsibility |
|-----------|------------|----------------|
| **Core Logic** | **Go (Golang)** | High-performance backend, `struct` modeling, and process management. |
| **Intelligence** | **Gemini 2.0 Flash** | Decision making and reasoning. |
| **Executor** | **Ansible** | Infrastructure as Code (IaC) for safe remediation. |
| **Infrastructure** | **Docker & Nginx** | Simulated banking API gateway. |
| **Notification** | **Discord Webhooks** | Reporting and alerting. |

---

## 🚀 Installation & Setup

### 1. Clone the Repository
```bash
git clone [https://github.com/Arda-Aras103/InfraMinds.git](https://github.com/Arda-Aras103/InfraMinds.git)
cd InfraMinds
```

### 2. Install Dependencies
Ensure you have Go, Ansible, and Docker installed.
```bash
go mod tidy
sudo apt install ansible docker.io
```

### 3. Configure Environment
Set your Google Cloud API Key.
```bash
export GOOGLE_API_KEY="YOUR_GEMINI_API_KEY"
```

### 4. Start the Victim Server
Launch the Nginx container that we will simulate failures on.
```bash
docker compose up -d
```

## Scenario A: Cost Optimization (File Mode)

The system reads routine logs from 'server.log'.
```bash
go run main.go
```

Console Output: 

    Source: File | Priority: 2
    Priority low. AI not triggered. (Saves API Quota)


## Scenario B: Critical Incident - Disk Cleanup (Terminal Mode)

We simulate a disk space emergency.
```bash
go run main.go "WARNING: Disk usage at 99%, please cleanup."
```

Console Output:

    Source: Terminal | Priority: 3 
    🤖 Decision: ACTION_CLEAN 
    ⚡ Ansible: Running clean_logs.yml... 
    
    📨 Discord: Notification sent.


## Scenario C: Critical Incident - Service Restart (Terminal Mode)
We simulate a service crash (Nginx unresponsive).
```bash
# First, manually stop the server to simulate a crash
docker stop bank_api_gateway

# Run the agent
go run main.go "CRITICAL: Nginx is unresponsive and connection refused."
```

Result:

    🤖 Decision: ACTION_RESTART ⚡ Ansible: Running fix_service.yml... ✅ Service Restored.
