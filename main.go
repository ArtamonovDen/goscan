package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type PortRange struct {
	Start int
	End   int
}

type ScanResult struct {
	Host string
	Port int
	Open bool
}

type ScanJob struct {
	Host    string
	Port    int
	Timeout time.Duration
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
		// TODO: handle error more specifically
		return false
	}
	defer conn.Close()
	return true

}

func scanPortsSequentially(cfg *Config) {

	timerStart := time.Now()

	for port := cfg.PortRange.Start; port <= cfg.PortRange.End; port++ {
		scan := scanPort(cfg.Host, port, cfg.Timeout)
		if scan {
			fmt.Printf("Open port %d\n", port)
		}
	}

	scanTime := time.Since(timerStart)
	fmt.Printf("Scan finished in %s\n", scanTime)
}

func scanPortWorker(jobs <-chan ScanJob, result chan<- ScanResult) {

	for job := range jobs {
		portOpen := scanPort(job.Host, job.Port, job.Timeout)
		result <- ScanResult{
			Host: job.Host,
			Port: job.Port,
			Open: portOpen,
		}
	}

}

func scanPorts(cfg *Config) {

	timerStart := time.Now()

	jobsChan := make(chan ScanJob)
	resultChan := make(chan ScanResult)

	// Run job worker
	var wg sync.WaitGroup
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			scanPortWorker(jobsChan, resultChan)
		}()

	}

	// Run producer
	go func() {
		defer close(jobsChan)
		for port := cfg.PortRange.Start; port <= cfg.PortRange.End; port++ {
			jobsChan <- ScanJob{
				Host:    cfg.Host,
				Port:    port,
				Timeout: cfg.Timeout,
			}
		}
	}()

	go func() {
		defer close(resultChan)
		wg.Wait()
	}()

	for result := range resultChan {
		if result.Open {
			fmt.Printf("Open port %s:%d\n", result.Host, result.Port)
		}
	}

	scanTime := time.Since(timerStart)
	fmt.Printf("Scan finished in %s\n", scanTime)

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

	//  Sync mode
	// scanPortsSequentially(&cfg)

	// Concurrent mode
	scanPorts(&cfg)

}
