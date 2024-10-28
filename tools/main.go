package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	requestURL  = "speed.cloudflare.com/cdn-cgi/trace"
	timeout     = 1 * time.Second
	maxDuration = 2 * time.Second
)

var (
	File         = flag.String("file", "ip.txt", "Name of the IP address file")
	outFile      = flag.String("outfile", "ip.csv", "Output file name")
	defaultPort  = flag.Int("port", 443, "Port")
	maxThreads   = flag.Int("max", 100, "Maximum number of concurrent request goroutines")
	speedTest    = flag.Int("speedtest", 5, "Number of download speed test goroutines; set to 0 to disable speed testing")
	speedTestURL = flag.String("url", "speed.cloudflare.com/__down?bytes=500000000", "URL for the speed test file")
	enableTLS    = flag.Bool("tls", true, "Whether to enable TLS")
)

type result struct {
	ip          string
	port        int
	dataCenter  string
	region      string
	city        string
	latency     string
	tcpDuration time.Duration
}

type speedtestresult struct {
	result
	downloadSpeed float64
}

type location struct {
	Iata   string  `json:"iata"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Cca2   string  `json:"cca2"`
	Region string  `json:"region"`
	City   string  `json:"city"`
}

func increaseMaxOpenFiles() {
	fmt.Println("Attempting to increase the limit of open file descriptors...")
	cmd := exec.Command("bash", "-c", "ulimit -n 10000")
	_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error occurred while increasing the limit of open file descriptors: %v\n", err)
	} else {
		fmt.Printf("The limit of open file descriptors has been increased!\n")
	}
}

func main() {
	flag.Parse()

	startTime := time.Now()
	osType := runtime.GOOS
	if osType == "linux" {
		increaseMaxOpenFiles()
	}

	var locations []location
	if _, err := os.Stat("locations.json"); os.IsNotExist(err) {
		fmt.Println("The local locations.json does not exist.\nDownloading locations.json from https://speed.cloudflare.com/locations")
		resp, err := http.Get("https://speed.cloudflare.com/locations")
		if err != nil {
			fmt.Printf("Unable to fetch JSON from URL: %v\n", err)
			return
		}

		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Unable to read response body: %v\n", err)
			return
		}

		err = json.Unmarshal(body, &locations)
		if err != nil {
			fmt.Printf("Unable to parse JSON: %v\n", err)
			return
		}
		file, err := os.Create("locations.json")
		if err != nil {
			fmt.Printf("Unable to create file: %v\n", err)
			return
		}
		defer file.Close()

		_, err = file.Write(body)
		if err != nil {
			fmt.Printf("Unable to write to file: %v\n", err)
			return
		}
	} else {
		fmt.Println("The local locations.json already exists, no need to re-download.")
		file, err := os.Open("locations.json")
		if err != nil {
			fmt.Printf("Unable to open file: %v\n", err)
			return
		}
		defer file.Close()

		body, err := ioutil.ReadAll(file)
		if err != nil {
			fmt.Printf("Unable to read file: %v\n", err)
			return
		}

		err = json.Unmarshal(body, &locations)
		if err != nil {
			fmt.Printf("Unable to parse JSON: %v\n", err)
			return
		}
	}

	locationMap := make(map[string]location)
	for _, loc := range locations {
		locationMap[loc.Iata] = loc
	}

	ips, err := readIPs(*File)
	if err != nil {
		fmt.Printf("Unable to read IPs from file: %v\n", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(ips))

	resultChan := make(chan result, len(ips))

	thread := make(chan struct{}, *maxThreads)

	var count int
	total := len(ips)

	for _, ip := range ips {
		thread <- struct{}{}
		go func(ip string) {
			defer func() {
				<-thread
				wg.Done()
				count++
				percentage := float64(count) / float64(total) * 100
				fmt.Printf("Completed: %d Total: %d Percentage: %.2f%%\r", count, total, percentage)
				if count == total {
					fmt.Printf("Completed: %d Total: %d Percentage: %.2f%%\n", count, total, percentage)
				}
			}()

			dialer := &net.Dialer{
				Timeout:   timeout,
				KeepAlive: 0,
			}
			start := time.Now()
			conn, err := dialer.Dial("tcp", net.JoinHostPort(ip, strconv.Itoa(*defaultPort)))
			if err != nil {
				return
			}
			defer conn.Close()

			tcpDuration := time.Since(start)
			start = time.Now()

			client := http.Client{
				Transport: &http.Transport{
					Dial: func(network, addr string) (net.Conn, error) {
						return conn, nil
					},
				},
				Timeout: timeout,
			}

			var protocol string
			if *enableTLS {
				protocol = "https://"
			} else {
				protocol = "http://"
			}
			requestURL := protocol + requestURL

			req, _ := http.NewRequest("GET", requestURL, nil)

			// Add User-Agent
			req.Header.Set("User-Agent", "Mozilla/5.0")
			req.Close = true
			resp, err := client.Do(req)
			if err != nil {
				return
			}

			duration := time.Since(start)
			if duration > maxDuration {
				return
			}

			buf := &bytes.Buffer{}
			// Create a timeout for the read operation
			timeout := time.After(maxDuration)
			// Use a goroutine to read the response body
			done := make(chan bool)
			go func() {
				_, err := io.Copy(buf, resp.Body)
				done <- true
				if err != nil {
					return
				}
			}()
			// Wait for the read operation to complete or timeout
			select {
			case <-done:
				// Read operation completed
			case <-timeout:
				// Read operation timed out
				return
			}

			body := buf
			if err != nil {
				return
			}

			if strings.Contains(body.String(), "uag=Mozilla/5.0") {
				if matches := regexp.MustCompile(`colo=([A-Z]+)`).FindStringSubmatch(body.String()); len(matches) > 1 {
					dataCenter := matches[1]
					loc, ok := locationMap[dataCenter]
					if ok {
						fmt.Printf("Valid IP found %s Location Info: %s Latency: %d ms\n", ip, loc.City, tcpDuration.Milliseconds())
						resultChan <- result{ip, *defaultPort, dataCenter, loc.Region, loc.City, fmt.Sprintf("%d ms", tcpDuration.Milliseconds()), tcpDuration}
					} else {
						fmt.Printf("Valid IP found %s Location Info: Unknown Latency: %d ms\n", ip, tcpDuration.Milliseconds())
						resultChan <- result{ip, *defaultPort, dataCenter, "", "", fmt.Sprintf("%d ms", tcpDuration.Milliseconds()), tcpDuration}
					}
				}
			}
		}(ip)
	}
	wg.Wait()
	close(resultChan)

if len(resultChan) == 0 {
	// Clear output content
	fmt.Print("\033[2J")
	fmt.Println("No valid IPs found")
	return
}
var results []speedtestresult
if *speedTest > 0 {
	fmt.Printf("Starting speed test\n")
	var wg2 sync.WaitGroup
	wg2.Add(*speedTest)
	count = 0
	total := len(resultChan)
	results = []speedtestresult{}
	for i := 0; i < *speedTest; i++ {
		thread <- struct{}{}
		go func() {
			defer func() {
				<-thread
				wg2.Done()
			}()
			for res := range resultChan {

				downloadSpeed := getDownloadSpeed(res.ip)
				results = append(results, speedtestresult{result: res, downloadSpeed: downloadSpeed})

				count++
				percentage := float64(count) / float64(total) * 100
				fmt.Printf("Completed: %.2f%%\r", percentage)
				if count == total {
					fmt.Printf("Completed: %.2f%%\n", percentage)
				}
			}
		}()
	}
	wg2.Wait()
} else {
	for res := range resultChan {
		results = append(results, speedtestresult{result: res})
	}
}

if *speedTest > 0 {
	sort.Slice(results, func(i, j int) bool {
		return results[i].downloadSpeed > results[j].downloadSpeed
	})
} else {
	sort.Slice(results, func(i, j int) bool {
		return results[i].result.tcpDuration < results[j].result.tcpDuration
	})
}

file, err := os.Create(*outFile)
if err != nil {
	fmt.Printf("Unable to create file: %v\n", err)
	return
}
defer file.Close()

writer := csv.NewWriter(file)
if *speedTest > 0 {
	writer.Write([]string{"IP Address", "Port", "TLS", "Data Center", "Region", "City", "Network Latency", "Download Speed"})
} else {
	writer.Write([]string{"IP Address", "Port", "TLS", "Data Center", "Region", "City", "Network Latency"})
}
for _, res := range results {
	if *speedTest > 0 {
		writer.Write([]string{res.result.ip, strconv.Itoa(res.result.port), strconv.FormatBool(*enableTLS), res.result.dataCenter, res.result.region, res.result.city, res.result.latency, fmt.Sprintf("%.0f kB/s", res.downloadSpeed)})
	} else {
		writer.Write([]string{res.result.ip, strconv.Itoa(res.result.port), strconv.FormatBool(*enableTLS), res.result.dataCenter, res.result.region, res.result.city, res.result.latency})
	}
}

writer.Flush()
// Clear output content
fmt.Print("\033[2J")
fmt.Printf("Successfully written results to file %s, elapsed time %d seconds\n", *outFile, time.Since(startTime)/time.Second)
}

func readIPs(File string) ([]string, error) {
	file, err := os.Open(File)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var ips []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ipAddr := scanner.Text()
		// Check if the IP address is in CIDR format
		if strings.Contains(ipAddr, "/") {
			ip, ipNet, err := net.ParseCIDR(ipAddr)
			if err != nil {
				fmt.Printf("Unable to parse CIDR format IP: %v\n", err)
				continue
			}
			for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); inc(ip) {
				ips = append(ips, ip.String())
			}
		} else {
			ips = append(ips, ipAddr)
		}
	}
	return ips, scanner.Err()
}

// inc function increments the IP address
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// Speed test function
func getDownloadSpeed(ip string) float64 {
	var protocol string
	if *enableTLS {
		protocol = "https://"
	} else {
		protocol = "http://"
	}
	speedTestURL := protocol + *speedTestURL
	// Create request
	req, _ := http.NewRequest("GET", speedTestURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	// Create TCP connection
	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: 0,
	}
	conn, err := dialer.Dial("tcp", net.JoinHostPort(ip, strconv.Itoa(*defaultPort)))
	if err != nil {
		return 0
	}
	defer conn.Close()

	fmt.Printf("Testing IP %s on port %s\n", ip, strconv.Itoa(*defaultPort))
	startTime := time.Now()
	// Create HTTP client
	client := http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				return conn, nil
			},
		},
		// Set maximum time for speed test on a single IP to 5 seconds
		Timeout: 5 * time.Second,
	}
	// Send request
	req.Close = true
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Speed test invalid for IP %s on port %s\n", ip, strconv.Itoa(*defaultPort))
		return 0
	}
	defer resp.Body.Close()

	// Copy response body to /dev/null and calculate download speed
	written, _ := io.Copy(io.Discard, resp.Body)
	duration := time.Since(startTime)
	speed := float64(written) / duration.Seconds() / 1024

	// Output result
	fmt.Printf("IP %s on port %s download speed %.0f kB/s\n", ip, strconv.Itoa(*defaultPort), speed)
	return speed
}
