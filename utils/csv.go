package utils

import (
	"encoding/csv"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

const (
	defaultOutput         = "result.csv"
	maxDelay              = 9999 * time.Millisecond
	minDelay              = 0 * time.Millisecond
	maxLossRate   float32 = 1.0
)

var (
	InputMaxDelay    = maxDelay
	InputMinDelay    = minDelay
	InputMaxLossRate = maxLossRate
	Output           = defaultOutput
	PrintNum         = 10
)

// Apakah akan mencetak hasil pengujian
func NoPrintResult() bool {
	return PrintNum == 0
}

// Apakah akan menulis hasil ke file
func noOutput() bool {
	return Output == "" || Output == " "
}

type PingData struct {
	IP       *net.IPAddr
	Sended   int
	Received int
	Delay    time.Duration
}

type CloudflareIPData struct {
	*PingData
	lossRate      float32
	DownloadSpeed float64
}

// Menghitung tingkat kehilangan paket
func (cf *CloudflareIPData) getLossRate() float32 {
	if cf.lossRate == 0 {
		pingLost := cf.Sended - cf.Received
		cf.lossRate = float32(pingLost) / float32(cf.Sended)
	}
	return cf.lossRate
}

func (cf *CloudflareIPData) toString() []string {
	result := make([]string, 6)
	result[0] = cf.IP.String()
	result[1] = strconv.Itoa(cf.Sended)
	result[2] = strconv.Itoa(cf.Received)
	result[3] = strconv.FormatFloat(float64(cf.getLossRate()), 'f', 2, 32)
	result[4] = strconv.FormatFloat(cf.Delay.Seconds()*1000, 'f', 2, 32)
	result[5] = strconv.FormatFloat(cf.DownloadSpeed/1024/1024, 'f', 2, 32)
	return result
}

func ExportCsv(data []CloudflareIPData) {
	if noOutput() || len(data) == 0 {
		return
	}
	fp, err := os.Create(Output)
	if err != nil {
		log.Fatalf("Gagal membuat file [%s]: %v", Output, err)
		return
	}
	defer fp.Close()
	w := csv.NewWriter(fp) // Membuat stream penulisan file baru
	_ = w.Write([]string{"Alamat IP", "Terkirim", "Diterima", "Tingkat Kehilangan Paket", "Rata-rata Latensi", "Kecepatan Unduh (MB/s)"})
	_ = w.WriteAll(convertToString(data))
	w.Flush()
}

func convertToString(data []CloudflareIPData) [][]string {
	result := make([][]string, 0)
	for _, v := range data {
		result = append(result, v.toString())
	}
	return result
}

// Pengurutan latensi dan kehilangan paket
type PingDelaySet []CloudflareIPData

// Penyaringan berdasarkan kondisi latensi
func (s PingDelaySet) FilterDelay() (data PingDelaySet) {
	if InputMaxDelay > maxDelay || InputMinDelay < minDelay { // Ketika kondisi latensi yang dimasukkan tidak dalam batasan default, tidak dilakukan penyaringan
		return s
	}
	if InputMaxDelay == maxDelay && InputMinDelay == minDelay { // Ketika kondisi latensi yang dimasukkan adalah nilai default, tidak dilakukan penyaringan
		return s
	}
	for _, v := range s {
		if v.Delay > InputMaxDelay { // Batas atas rata-rata latensi, jika latensi melebihi nilai maksimum yang ditetapkan, data berikutnya tidak memenuhi syarat, langsung keluar dari loop
			break
		}
		if v.Delay < InputMinDelay { // Batas bawah rata-rata latensi, jika latensi kurang dari nilai minimum yang ditetapkan, tidak memenuhi syarat, lewati
			continue
		}
		data = append(data, v) // Jika latensi memenuhi syarat, tambahkan ke array baru
	}
	return
}

// Penyaringan berdasarkan kondisi kehilangan paket
func (s PingDelaySet) FilterLossRate() (data PingDelaySet) {
	if InputMaxLossRate >= maxLossRate { // Ketika kondisi kehilangan paket yang dimasukkan adalah nilai default, tidak dilakukan penyaringan
		return s
	}
	for _, v := range s {
		if v.getLossRate() > InputMaxLossRate { // Batas atas tingkat kehilangan paket
			break
		}
		data = append(data, v) // Jika tingkat kehilangan paket memenuhi syarat, tambahkan ke array baru
	}
	return
}

func (s PingDelaySet) Len() int {
	return len(s)
}
func (s PingDelaySet) Less(i, j int) bool {
	iRate, jRate := s[i].getLossRate(), s[j].getLossRate()
	if iRate != jRate {
		return iRate < jRate
	}
	return s[i].Delay < s[j].Delay
}
func (s PingDelaySet) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Pengurutan kecepatan unduh
type DownloadSpeedSet []CloudflareIPData

func (s DownloadSpeedSet) Len() int {
	return len(s)
}
func (s DownloadSpeedSet) Less(i, j int) bool {
	return s[i].DownloadSpeed > s[j].DownloadSpeed
}
func (s DownloadSpeedSet) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s DownloadSpeedSet) Print() {
	if NoPrintResult() {
		return
	}
	if len(s) <= 0 { // Lanjutkan jika panjang array IP (jumlah IP) lebih besar dari 0
		fmt.Println("\n[Info] Jumlah hasil uji kecepatan lengkap adalah 0, melewati output hasil.")
		return
	}
	dateString := convertToString(s) // Konversi ke array multidimensi [][]String
	if len(dateString) < PrintNum {  // Jika panjang array IP (jumlah IP) kurang dari jumlah cetakan, maka jumlah cetakan diubah menjadi jumlah IP
		PrintNum = len(dateString)
	}
	headFormat := "%-16s%-5s%-5s%-5s%-6s%-11s\n"
	dataFormat := "%-18s%-8s%-8s%-8s%-10s%-15s\n"
	for i := 0; i < PrintNum; i++ { // Jika IP yang akan dicetak mencakup IPv6, maka perlu menyesuaikan spasi
		if len(dateString[i][0]) > 15 {
			headFormat = "%-40s%-5s%-5s%-5s%-6s%-11s\n"
			dataFormat = "%-42s%-8s%-8s%-8s%-10s%-15s\n"
			break
		}
	}
	fmt.Printf(headFormat, "Alamat IP", "Terkirim", "Diterima", "Tingkat Kehilangan Paket", "Rata-rata Latensi", "Kecepatan Unduh (MB/s)")
	for i := 0; i < PrintNum; i++ {
		fmt.Printf(dataFormat, dateString[i][0], dateString[i][1], dateString[i][2], dateString[i][3], dateString[i][4], dateString[i][5])
	}
	if !noOutput() {
		fmt.Printf("\nHasil uji kecepatan lengkap telah ditulis ke file %v, Anda dapat menggunakan Notepad/Perangkat Lunak Spreadsheet untuk melihatnya.\n", Output)
	}
}
