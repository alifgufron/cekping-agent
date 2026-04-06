package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/awandataindonesia/cekping-agent/internal/config"
	"github.com/awandataindonesia/cekping-agent/internal/executor"
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

	// Local Testing Flags
	pingTarget := flag.String("ping", "", "Run a local ping test against a target (IPv4 or IPv6)")
	mtrTarget := flag.String("mtr", "", "Run a local MTR test against a target (IPv4 or IPv6)")
	count := flag.Int("count", 4, "Number of packets/cycles for local test")

	flag.Parse()

	// Handle Local Testing
	if *pingTarget != "" {
		runLocalPing(*pingTarget, *count)
		return
	}
	if *mtrTarget != "" {
		runLocalMTR(*mtrTarget, *count)
		return
	}

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

func runLocalPing(target string, count int) {
	fmt.Printf("Pinging %s with %d packets...\n", target, count)
	stats, err := executor.DoPing(context.Background(), target, count, func(seq, ttl int, rtt float64) {
		if rtt > 0 {
			fmt.Printf("%d bytes from %s: icmp_seq=%d time=%.2f ms\n", 64, target, seq, rtt)
		} else {
			fmt.Printf("Request timeout for icmp_seq %d\n", seq)
		}
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\n--- %s ping statistics ---\n", target)
	fmt.Printf("%d packets transmitted, %d packets received, %.1f%% packet loss\n",
		count, len(stats.Rtts), stats.PacketLoss)
	if len(stats.Rtts) > 0 {
		fmt.Printf("round-trip min/avg/max/stddev = %.3f/%.3f/%.3f/%.3f ms\n",
			stats.Min, stats.Avg, stats.Max, stats.StdDev)
	}
}

func runLocalMTR(target string, count int) {
	fmt.Printf("MTR to %s...\n", target)

	// Collect latest stats for each hop
	latestHops := make(map[int]executor.MTRHopStats)

	err := executor.DoMTR(context.Background(), target, count, func(stats executor.MTRHopStats) {
		latestHops[stats.Hop] = stats
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Print final clean table
	fmt.Printf("\n%-3s %-30s %-6s %-6s %-6s %-6s %-6s\n", "Hop", "Hostname/IP", "Loss%", "Last", "Avg", "Best", "Wrst")
	for i := 1; i <= 30; i++ {
		if h, ok := latestHops[i]; ok && h.IP != "" {
			fmt.Printf("%-3d %-30s %-6.1f %-6.1f %-6.1f %-6.1f %-6.1f\n",
				h.Hop, h.IP, h.Loss, h.Last, h.Avg, h.Best, h.Worst)
			
			// If we reached the target (IP matches target), we can stop printing further hops if we want, 
			// but usually MTR shows all discovered hops.
		}
	}
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
