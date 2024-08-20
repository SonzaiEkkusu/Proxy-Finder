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
	requestURL  = "speed.cloudflare.com/cdn-cgi/trace" // URL trace permintaan
	timeout     = 1 * time.Second                      // Waktu timeout
	maxDuration = 2 * time.Second                      // Durasi maksimum
)

var (
	File         = flag.String("file", "ip.txt", "Nama file alamat IP")                                  // Nama file alamat IP
	outFile      = flag.String("outfile", "ip.csv", "Nama file output")                                  // Nama file output
	defaultPort  = flag.Int("port", 443, "Port")                                                         // Port
	maxThreads   = flag.Int("max", 100, "Jumlah maksimum goroutine permintaan bersamaan")                // Jumlah maksimum goroutine
	speedTest    = flag.Int("speedtest", 5, "Jumlah goroutine uji kecepatan unduh, setel ke 0 untuk menonaktifkan uji kecepatan") // Jumlah goroutine uji kecepatan unduh
	speedTestURL = flag.String("url", "speed.cloudflare.com/__down?bytes=500000000", "URL file uji kecepatan") // URL file uji kecepatan
	enableTLS    = flag.Bool("tls", true, "Apakah mengaktifkan TLS")                                     // Apakah mengaktifkan TLS
)

type result struct {
	ip          string        // Alamat IP
	port        int           // Port
	dataCenter  string        // Pusat data
	region      string        // Wilayah
	city        string        // Kota
	latency     string        // Latensi
	tcpDuration time.Duration // Latensi permintaan TCP
}

type speedtestresult struct {
	result
	downloadSpeed float64 // Kecepatan unduh
}

type location struct {
	Iata   string  `json:"iata"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Cca2   string  `json:"cca2"`
	Region string  `json:"region"`
	City   string  `json:"city"`
}

// Mencoba meningkatkan batas deskriptor file
func increaseMaxOpenFiles() {
	fmt.Println("Sedang mencoba meningkatkan batas deskriptor file...")
	cmd := exec.Command("bash", "-c", "ulimit -n 10000")
	_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Terjadi kesalahan saat meningkatkan batas deskriptor file: %v\n", err)
	} else {
		fmt.Printf("Batas deskriptor file berhasil ditingkatkan!\n")
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
		fmt.Println("File locations.json tidak ada\nSedang mengunduh locations.json dari https://speed.cloudflare.com/locations")
		resp, err := http.Get("https://speed.cloudflare.com/locations")
		if err != nil {
			fmt.Printf("Tidak dapat mengambil JSON dari URL: %v\n", err)
			return
		}

		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Tidak dapat membaca body respons: %v\n", err)
			return
		}

		err = json.Unmarshal(body, &locations)
		if err != nil {
			fmt.Printf("Tidak dapat mengurai JSON: %v\n", err)
			return
		}
		file, err := os.Create("locations.json")
		if err != nil {
			fmt.Printf("Tidak dapat membuat file: %v\n", err)
			return
		}
		defer file.Close()

		_, err = file.Write(body)
		if err != nil {
			fmt.Printf("Tidak dapat menulis ke file: %v\n", err)
			return
		}
	} else {
		fmt.Println("File locations.json sudah ada, tidak perlu mengunduh ulang")
		file, err := os.Open("locations.json")
		if err != nil {
			fmt.Printf("Tidak dapat membuka file: %v\n", err)
			return
		}
		defer file.Close()

		body, err := ioutil.ReadAll(file)
		if err != nil {
			fmt.Printf("Tidak dapat membaca file: %v\n", err)
			return
		}

		err = json.Unmarshal(body, &locations)
		if err != nil {
			fmt.Printf("Tidak dapat mengurai JSON: %v\n", err)
			return
		}
	}

	locationMap := make(map[string]location)
	for _, loc := range locations {
		locationMap[loc.Iata] = loc
	}

	ips, err := readIPs(*File)
	if err != nil {
		fmt.Printf("Tidak dapat membaca IP dari file: %v\n", err)
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
				fmt.Printf("Selesai: %d Total: %d Selesai: %.2f%%\r", count, total, percentage)
				if count == total {
					fmt.Printf("Selesai: %d Total: %d Selesai: %.2f%%\n", count, total, percentage)
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

			// Menambahkan user agent
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
			// Membuat timeout untuk operasi baca
			timeout := time.After(maxDuration)
			// Menggunakan goroutine untuk membaca body respons
			done := make(chan bool)
			go func() {
				_, err := io.Copy(buf, resp.Body)
				done <- true
				if err != nil {
					return
				}
			}()
			// Menunggu operasi baca selesai atau timeout
			select {
			case <-done:
				// Operasi baca selesai
			case <-timeout:
				// Operasi baca timeout
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
						fmt.Printf("Ditemukan IP valid %s dengan informasi lokasi %s dan latensi %d milidetik\n", ip, loc.City, tcpDuration.Milliseconds())
						resultChan <- result{ip, *defaultPort, dataCenter, loc.Region, loc.City, fmt.Sprintf("%d ms", tcpDuration.Milliseconds()), tcpDuration}
					} else {
						fmt.Printf("Ditemukan IP valid %s dengan informasi lokasi tidak diketahui, latensi %d milidetik\n", ip, tcpDuration.Milliseconds())
						resultChan <- result{ip, *defaultPort, dataCenter, "", "", fmt.Sprintf("%d ms", tcpDuration.Milliseconds()), tcpDuration}
					}
				}
			}
		}(ip)
	}

	wg.Wait()
	close(resultChan)

	if len(resultChan) == 0 {
		// Membersihkan output
		fmt.Print("\033[2J")
		fmt.Println("Tidak ditemukan IP yang valid")
		return
	}
	var results []speedtestresult
	if *speedTest > 0 {
		fmt.Printf("Mulai uji kecepatan\n")
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
					fmt.Printf("Selesai: %.2f%%\r", percentage)
					if count == total {
						fmt.Printf("Selesai: %.2f%%\033[0\n", percentage)

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
		fmt.Printf("Tidak dapat membuat file: %v\n", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if *speedTest > 0 {
		writer.Write([]string{"Alamat IP", "Port", "TLS", "Pusat Data", "Wilayah", "Kota", "Latensi Jaringan", "Kecepatan Unduh"})
	} else {
		writer.Write([]string{"Alamat IP", "Port", "TLS", "Pusat Data", "Wilayah", "Kota", "Latensi Jaringan"})
	}
	for _, res := range results {
		if *speedTest > 0 {
			writer.Write([]string{res.result.ip, strconv.Itoa(res.result.port), strconv.FormatBool(*enableTLS), res.result.dataCenter, res.result.region, res.result.city, res.result.latency, fmt.Sprintf("%.0f kB/s", res.downloadSpeed)})
		} else {
			writer.Write([]string{res.result.ip, strconv.Itoa(res.result.port), strconv.FormatBool(*enableTLS), res.result.dataCenter, res.result.region, res.result.city, res.result.latency})
		}
	}
