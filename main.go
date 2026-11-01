package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type PortRange struct {
	Start int
	End   int
}

const MaxPort = 65535

func parsePortRange(rawPorts string) (PortRange, error) {
	parts := strings.Split(rawPorts, "-")
	if len(parts) != 2 {
		return PortRange{}, fmt.Errorf("invalid port range: %q", rawPorts)
	}

	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return PortRange{}, fmt.Errorf("invalid start port: %w", err)
	}

	end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return PortRange{}, fmt.Errorf("invalid end port: %w", err)
	}

	if start < 1 || end > MaxPort || start > end {
		return PortRange{}, fmt.Errorf("invalid port range: %d-%d", start, end)
	}

	return PortRange{Start: start, End: end}, nil

}

type Config struct {
	Host      string
	PortRange PortRange
	Workers   int
	Timeout   time.Duration
}

func scanPort(host string, port int, timeout time.Duration) bool {
	address := net.JoinHostPort(host, strconv.Itoa(port))

	conn, err := net.DialTimeout("tcp", address, timeout)

	if err != nil {
		return false
	}
	defer conn.Close()
	return true

}

func scanPorts(cfg *Config) {

	for port := cfg.PortRange.Start; port <= cfg.PortRange.End; port++ {
		scan := scanPort(cfg.Host, port, cfg.Timeout)
		if scan {
			fmt.Printf("Open port %d\n", port)
		}
	}
}

func parseConfig() (Config, error) {

	rawHost := flag.String("host", "127.0.0.1", "Target hostname or IP")
	rawPorts := flag.String("ports", "1-65535", "Port range to scan")
	workers := flag.Int("workers", 1, "Number of concurrent workers")
	timeout := flag.Duration("timeout", 10*time.Millisecond, "Connection timeout (10ms)")

	flag.Parse()

	if flag.NArg() > 0 {
		return Config{}, fmt.Errorf("unexpected positional arguments: %q", flag.Args())
	}

	portRange, err := parsePortRange(*rawPorts)
	if err != nil {
		return Config{}, err
	}

	if *workers <= 0 {
		return Config{}, fmt.Errorf("worker number must be positive, got %d", *workers)
	}

	if *timeout <= 0 {
		return Config{}, fmt.Errorf("timeout must be positive, got %s", *timeout)
	}

	host := strings.TrimSpace(*rawHost)
	if host == "" {
		return Config{}, fmt.Errorf("host cannot be empty")
	}

	return Config{Host: host, PortRange: portRange, Workers: *workers, Timeout: *timeout}, nil

}

func main() {

	cfg, err := parseConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

	fmt.Println("Runing scanner with config:")
	fmt.Printf("Host: %s\n", cfg.Host)
	fmt.Printf("Ports: %+v\n", cfg.PortRange)
	fmt.Printf("Workers: %d\n", cfg.Workers)
	fmt.Printf("Timeout: %s\n", cfg.Timeout)

	scanPorts(&cfg)
}
