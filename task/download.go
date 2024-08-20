package task

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/SonzaiEkkusu/Proxy-Finder/utils"

	"github.com/VividCortex/ewma"
)

const (
	bufferSize                     = 1024
	defaultURL                     = "https://cf.xiu2.xyz/url"
	defaultTimeout                 = 10 * time.Second
	defaultDisableDownload         = false
	defaultTestNum                 = 10
	defaultMinSpeed        float64 = 0.0
)

var (
	URL     = defaultURL
	Timeout = defaultTimeout
	Disable = defaultDisableDownload

	TestCount = defaultTestNum
	MinSpeed  = defaultMinSpeed
)

func checkDownloadDefault() {
	if URL == "" {
		URL = defaultURL
	}
	if Timeout <= 0 {
		Timeout = defaultTimeout
	}
	if TestCount <= 0 {
		TestCount = defaultTestNum
	}
	if MinSpeed <= 0.0 {
		MinSpeed = defaultMinSpeed
	}
}

func TestDownloadSpeed(ipSet utils.PingDelaySet) (speedSet utils.DownloadSpeedSet) {
	checkDownloadDefault()
	if Disable {
		return utils.DownloadSpeedSet(ipSet)
	}
	if len(ipSet) <= 0 { // IP array panjang (jumlah IP) lebih besar dari 0 baru melanjutkan tes kecepatan unduh
		fmt.Println("\n[Informasi] Hasil tes ping jumlah IP adalah 0, melewati tes kecepatan unduh.")
		return
	}
	testNum := TestCount
	if len(ipSet) < TestCount || MinSpeed > 0 { // Jika panjang array IP (jumlah IP) lebih kecil dari jumlah tes kecepatan unduh (-dn), maka jumlahnya disesuaikan dengan jumlah IP
		testNum = len(ipSet)
	}
	if testNum < TestCount {
		TestCount = testNum
	}

	fmt.Printf("Mulai tes kecepatan unduh (batas bawah: %.2f MB/s, jumlah: %d, antrian: %d)\n", MinSpeed, TestCount, testNum)
	// Mengatur panjang progress bar tes kecepatan unduh dan tes ping agar sesuai (obsesif-kompulsif)
	bar_a := len(strconv.Itoa(len(ipSet)))
	bar_b := "     "
	for i := 0; i < bar_a; i++ {
		bar_b += " "
	}
	bar := utils.NewBar(TestCount, bar_b, "")
	for i := 0; i < testNum; i++ {
		speed := downloadHandler(ipSet[i].IP)
		ipSet[i].DownloadSpeed = speed
		// Setelah setiap IP diuji kecepatan unduhnya, filter hasil berdasarkan [batas bawah kecepatan unduh]
		if speed >= MinSpeed*1024*1024 {
			bar.Grow(1, "")
			speedSet = append(speedSet, ipSet[i]) // Jika lebih tinggi dari batas bawah kecepatan unduh, tambahkan ke array baru
			if len(speedSet) == TestCount {       // Setelah cukup jumlah IP yang memenuhi syarat (jumlah tes kecepatan unduh -dn), keluar dari loop
				break
			}
		}
	}
	bar.Done()
	if len(speedSet) == 0 { // Tidak ada data yang memenuhi batas kecepatan, kembalikan semua data tes
		speedSet = utils.DownloadSpeedSet(ipSet)
	}
	// Urutkan berdasarkan kecepatan
	sort.Sort(speedSet)
	return
}

func getDialContext(ip *net.IPAddr) func(ctx context.Context, network, address string) (net.Conn, error) {
	var fakeSourceAddr string
	if isIPv4(ip.String()) {
		fakeSourceAddr = fmt.Sprintf("%s:%d", ip.String(), TCPPort)
	} else {
		fakeSourceAddr = fmt.Sprintf("[%s]:%d", ip.String(), TCPPort)
	}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, fakeSourceAddr)
	}
}

// Mengembalikan kecepatan unduh
func downloadHandler(ip *net.IPAddr) float64 {
	client := &http.Client{
		Transport: &http.Transport{DialContext: getDialContext(ip)},
		Timeout:   Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 10 { // Batasi maksimal 10 kali pengalihan
				return http.ErrUseLastResponse
			}
			if req.Header.Get("Referer") == defaultURL { // Saat menggunakan URL unduh default, pengalihan tidak membawa Referer
				req.Header.Del("Referer")
			}
			return nil
		},
	}
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return 0.0
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36")

	response, err := client.Do(req)
	if err != nil {
		return 0.0
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return 0.0
	}
	timeStart := time.Now()           // Waktu mulai (sekarang)
	timeEnd := timeStart.Add(Timeout) // Tambahkan waktu tes kecepatan unduh untuk mendapatkan waktu selesai

	contentLength := response.ContentLength // Ukuran file
	buffer := make([]byte, bufferSize)

	var (
		contentRead     int64 = 0
		timeSlice             = Timeout / 100
		timeCounter           = 1
		lastContentRead int64 = 0
	)

	var nextTime = timeStart.Add(timeSlice * time.Duration(timeCounter))
	e := ewma.NewMovingAverage()

	// Loop untuk menghitung, jika file selesai diunduh (keduanya sama), keluar dari loop (hentikan tes kecepatan)
	for contentLength != contentRead {
		currentTime := time.Now()
		if currentTime.After(nextTime) {
			timeCounter++
			nextTime = timeStart.Add(timeSlice * time.Duration(timeCounter))
			e.Add(float64(contentRead - lastContentRead))
			lastContentRead = contentRead
		}
		// Jika melebihi waktu tes kecepatan unduh, keluar dari loop (hentikan tes kecepatan)
		if currentTime.After(timeEnd) {
			break
		}
		bufferRead, err := response.Body.Read(buffer)
		if err != nil {
			if err != io.EOF { // Jika terjadi kesalahan saat mengunduh file (misalnya Timeout), dan bukan karena file sudah selesai diunduh, keluar dari loop (hentikan tes kecepatan)
				break
			} else if contentLength == -1 { // File selesai diunduh dan ukuran file tidak diketahui, keluar dari loop (hentikan tes kecepatan), misalnya: https://speed.cloudflare.com/__down?bytes=200000000 jika diunduh dalam 10 detik, dapat menyebabkan hasil tes kecepatan sangat rendah atau bahkan menunjukkan 0.00 (kecepatan unduh terlalu cepat)
				break
			}
			// Dapatkan potongan waktu sebelumnya
			last_time_slice := timeStart.Add(timeSlice * time.Duration(timeCounter-1))
			// Jumlah data yang diunduh / (gunakan waktu saat ini - potongan waktu sebelumnya / potongan waktu)
			e.Add(float64(contentRead-lastContentRead) / (float64(currentTime.Sub(last_time_slice)) / float64(timeSlice)))
		}
		contentRead += int64(bufferRead)
	}
	return e.Value() / (Timeout.Seconds() / 120)
}
