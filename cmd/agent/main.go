package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/awandataindonesia/cekping-agent/internal/config"
	"github.com/awandataindonesia/cekping-agent/internal/worker"
)

const serviceTemplate = `[Unit]
Description=CekPing Agent
After=network.target

[Service]
ExecStart=/usr/local/bin/cekping-agent
Restart=always
User=root
Environment=CEKPING_TOKEN={{TOKEN}}
Environment=CEKPING_SERVER={{SERVER}}
Environment=CEKPING_SECURE={{SECURE}}

[Install]
WantedBy=multi-user.target
`

func main() {
	// Flags
	install := flag.Bool("install", false, "Install the agent as a systemd service")
	token := flag.String("token", "", "Agent Token (required for install)")
	server := flag.String("server", "localhost:50051", "Server Address (for install)")
	secure := flag.Bool("secure", false, "Use secure connection (for install)")
	logFile := flag.String("logfile", "", "Log file path (optional, default: stdout)")
	flag.Parse()

	// Setup logging to file if specified
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	if *install {
		runInstall(*token, *server, *secure)
		return
	}

	log.Println("Starting CekPing Agent...")
	cfg := config.LoadConfig()

	if cfg.ServerAddr == "" || cfg.Token == "" {
		log.Fatal("Error: CEKPING_SERVER and CEKPING_TOKEN environment variables are required.")
	}

	w := worker.NewWorker(cfg)
	w.Start()
}

func runInstall(token, server string, secure bool) {
	if token == "" {
		log.Fatal("Error: -token is required for installation")
	}

	// Check Root
	if os.Geteuid() != 0 {
		log.Fatal("Error: Installation requires root privileges (sudo)")
	}

	binPath := "/usr/local/bin/cekping-agent"
	servicePath := "/etc/systemd/system/cekping-agent.service"

	// 1. Copy Binary
	log.Printf("Installing binary to %s...", binPath)
	selfPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to locate self: %v", err)
	}

	// Stop service first if running
	_ = exec.Command("systemctl", "stop", "cekping-agent").Run()

	input, err := os.ReadFile(selfPath)
	if err != nil {
		log.Fatalf("Failed to read self: %v", err)
	}
	if err := os.WriteFile(binPath, input, 0755); err != nil {
		log.Fatalf("Failed to copy binary: %v", err)
	}

	// 2. Create Service File
	log.Println("Creating systemd service...")

	secureStr := "false"
	if secure {
		secureStr = "true"
	}

	content := strings.ReplaceAll(serviceTemplate, "{{TOKEN}}", token)
	content = strings.ReplaceAll(content, "{{SERVER}}", server)
	content = strings.ReplaceAll(content, "{{SECURE}}", secureStr)

	if err := os.WriteFile(servicePath, []byte(content), 0644); err != nil {
		log.Fatalf("Failed to create service file: %v", err)
	}

	// 3. Enable & Start
	log.Println("Enabling and starting service...")
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		log.Fatalf("Daemon reload failed: %v", err)
	}
	if err := exec.Command("systemctl", "enable", "--now", "cekping-agent").Run(); err != nil {
		log.Fatalf("Failed to enable service: %v", err)
	}

	log.Println("Installation Successful! Service 'cekping-agent' is running.")
}
