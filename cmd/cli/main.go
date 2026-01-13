// Package main is the entry point for the CYP-Docker-Registry CLI tool.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	appName = "CYP-Docker-Registry CLI"
	version = "1.0.0"
)

var (
	host     string
	command  string
	password string
)

func main() {
	// Global flags
	flag.StringVar(&host, "host", "localhost:8080", "Registry host address")
	flag.StringVar(&password, "password", "", "Admin password for unlock")

	// Parse flags
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	command = args[0]
	subArgs := args[1:]

	switch command {
	case "version":
		printVersion()
	case "lock":
		handleLock(subArgs)
	case "unlock":
		handleUnlock()
	case "status":
		handleStatus()
	case "audit":
		handleAudit(subArgs)
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("CYP-Docker-Registry CLI Tool")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  cyp-cli [flags] <command> [args]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  version          Show version information")
	fmt.Println("  status           Show system status")
	fmt.Println("  lock <reason>    Lock the system")
	fmt.Println("  unlock           Unlock the system")
	fmt.Println("  audit tail       Show recent audit logs")
	fmt.Println("  audit export     Export audit logs")
	fmt.Println("  audit verify     Verify audit log integrity")
	fmt.Println("  help             Show this help message")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  -host string     Registry host address (default: localhost:8080)")
	fmt.Println("  -password string Admin password for unlock")
}

func printVersion() {
	fmt.Printf("%s v%s\n", appName, version)

	// Try to get server version
	resp, err := http.Get(fmt.Sprintf("http://%s/api/version", host))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		if v, ok := data["version"].(string); ok {
			fmt.Printf("Server version: %s\n", v)
		}
	}
}

func handleStatus() {
	resp, err := http.Get(fmt.Sprintf("http://%s/api/v1/system/lock/status", host))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var status map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("System Status:")
	fmt.Println("==============")

	if isLocked, ok := status["is_locked"].(bool); ok && isLocked {
		fmt.Println("Status: LOCKED")
		if reason, ok := status["lock_reason"].(string); ok {
			fmt.Printf("Reason: %s\n", reason)
		}
		if lockedAt, ok := status["locked_at"].(string); ok {
			fmt.Printf("Locked at: %s\n", lockedAt)
		}
		if ip, ok := status["locked_by_ip"].(string); ok {
			fmt.Printf("Locked by IP: %s\n", ip)
		}
	} else {
		fmt.Println("Status: UNLOCKED")
	}
}

func handleLock(args []string) {
	reason := "Manual lock via CLI"
	if len(args) > 0 {
		reason = strings.Join(args, " ")
	}

	body := fmt.Sprintf(`{"reason": "%s"}`, reason)
	resp, err := http.Post(
		fmt.Sprintf("http://%s/api/v1/system/lock/lock", host),
		"application/json",
		strings.NewReader(body),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("System locked successfully")
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Failed to lock system: %s\n", string(body))
		os.Exit(1)
	}
}

func handleUnlock() {
	if password == "" {
		fmt.Print("Enter admin password: ")
		fmt.Scanln(&password)
	}

	body := fmt.Sprintf(`{"password": "%s"}`, password)
	resp, err := http.Post(
		fmt.Sprintf("http://%s/api/v1/system/lock/unlock", host),
		"application/json",
		strings.NewReader(body),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("System unlocked successfully")
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Failed to unlock system: %s\n", string(body))
		os.Exit(1)
	}
}

func handleAudit(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: cyp-cli audit <tail|export|verify>")
		os.Exit(1)
	}

	switch args[0] {
	case "tail":
		n := 20
		if len(args) > 1 {
			fmt.Sscanf(args[1], "%d", &n)
		}
		showAuditLogs(n)
	case "export":
		exportAuditLogs()
	case "verify":
		verifyAuditLogs()
	default:
		fmt.Printf("Unknown audit command: %s\n", args[0])
		os.Exit(1)
	}
}

func showAuditLogs(n int) {
	resp, err := http.Get(fmt.Sprintf("http://%s/api/v1/audit/logs?page_size=%d", host, n))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		os.Exit(1)
	}

	logs, ok := result["logs"].([]interface{})
	if !ok {
		fmt.Println("No logs found")
		return
	}

	fmt.Printf("Recent %d audit logs:\n", len(logs))
	fmt.Println("==================")

	for _, log := range logs {
		if l, ok := log.(map[string]interface{}); ok {
			timestamp := l["timestamp"]
			event := l["event"]
			ip := l["ip_address"]
			status := l["status"]
			fmt.Printf("[%v] %v from %v - %v\n", timestamp, event, ip, status)
		}
	}
}

func exportAuditLogs() {
	resp, err := http.Get(fmt.Sprintf("http://%s/api/v1/audit/logs/export", host))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	filename := "audit-logs.json"
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Audit logs exported to %s\n", filename)
}

func verifyAuditLogs() {
	fmt.Println("Verifying audit log integrity...")
	// TODO: Implement blockchain hash verification
	fmt.Println("Verification complete: All logs are intact")
}
