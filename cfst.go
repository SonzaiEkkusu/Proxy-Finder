package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/XIU2/CloudflareSpeedTest/task"
	"github.com/XIU2/CloudflareSpeedTest/utils"
)

var (
	version, versionNew string
)

func init() {
	var printVersion bool
	var help = `
CloudflareSpeedTest ` + version + `
Mengukur latensi dan kecepatan semua IP Cloudflare CDN untuk mendapatkan IP tercepat (IPv4+IPv6)!
https://github.com/XIU2/CloudflareSpeedTest

Parameter:
    -n 200
        Jumlah thread pengujian latensi; semakin banyak thread, semakin cepat pengujian latensi, jangan terlalu tinggi pada perangkat dengan performa rendah (seperti router); (default 200, maksimum 1000)
    -t 4
        Jumlah pengujian latensi; jumlah pengujian latensi untuk satu IP; (default 4 kali)
    -dn 10
        Jumlah pengujian unduh; setelah mengurutkan latensi, jumlah pengujian unduh yang dilakukan dari latensi terendah; (default 10)
    -dt 10
        Waktu pengujian unduh; durasi maksimum pengujian unduh untuk satu IP, jangan terlalu singkat; (default 10 detik)
    -tp 443
        Port pengujian yang ditentukan; port yang digunakan untuk pengujian latensi/unduh; (default port 443)
    -url https://cf.xiu2.xyz/url
        Alamat pengujian yang ditentukan; alamat yang digunakan untuk pengujian latensi (HTTPing)/unduh, alamat default tidak dijamin ketersediaannya, disarankan untuk menggunakan alamat sendiri;

    -httping
        Ganti mode pengujian; ubah mode pengujian latensi menjadi protokol HTTP, alamat pengujian menggunakan parameter [-url]; (default TCPing)
    -httping-code 200
        Kode status yang valid; kode status HTTP yang valid untuk pengujian latensi HTTPing, hanya satu kode yang diperbolehkan; (default 200 301 302)
    -cfcolo HKG,KHH,NRT,LAX,SEA,SJC,FRA,MAD
        Cocokkan lokasi tertentu; nama lokasi menggunakan kode tiga huruf bandara lokal, dipisahkan dengan koma, hanya tersedia untuk mode HTTPing; (default semua lokasi)

    -tl 200
        Batas atas latensi rata-rata; hanya tampilkan IP dengan latensi rata-rata di bawah batas yang ditentukan, kondisi batas atas dan bawah dapat digunakan bersama; (default 9999 ms)
    -tll 40
        Batas bawah latensi rata-rata; hanya tampilkan IP dengan latensi rata-rata di atas batas yang ditentukan; (default 0 ms)
    -tlr 0.2
        Batas atas tingkat kehilangan paket; hanya tampilkan IP dengan tingkat kehilangan paket di bawah atau sama dengan batas yang ditentukan, rentang 0.00~1.00, 0 menghilangkan IP dengan kehilangan paket; (default 1.00)
    -sl 5
        Batas bawah kecepatan unduh; hanya tampilkan IP dengan kecepatan unduh di atas batas yang ditentukan, pengujian akan berhenti setelah mencapai jumlah yang ditentukan [-dn]; (default 0.00 MB/s)

    -p 10
        Jumlah hasil yang ditampilkan; setelah pengujian, langsung tampilkan jumlah hasil yang ditentukan, jika 0, tidak menampilkan hasil dan langsung keluar; (default 10 hasil)
    -f ip.txt
        File data rentang IP; jika path mengandung spasi, harap gunakan tanda kutip; mendukung rentang IP CDN lainnya; (default ip.txt)
    -ip 1.1.1.1,2.2.2.2/24,2606:4700::/32
        Data rentang IP yang ditentukan; langsung tentukan data rentang IP yang ingin diuji melalui parameter, dipisahkan dengan koma; (default kosong)
    -o result.csv
        Menulis file hasil; jika path mengandung spasi, harap gunakan tanda kutip; jika kosong, tidak menulis ke file [-o ""]; (default result.csv)

    -dd
        Nonaktifkan pengujian unduh; jika dinonaktifkan, hasil pengujian akan diurutkan berdasarkan latensi (default diurutkan berdasarkan kecepatan unduh); (default aktif)
    -allip
        Uji semua IP; melakukan pengujian untuk setiap IP dalam rentang IP (hanya mendukung IPv4); (default menguji satu IP acak per rentang /24)

    -v
        Tampilkan versi program + periksa pembaruan versi
    -h
        Tampilkan panduan bantuan
`
	var minDelay, maxDelay, downloadTime int
	var maxLossRate float64
	flag.IntVar(&task.Routines, "n", 200, "Jumlah thread pengujian latensi")
	flag.IntVar(&task.PingTimes, "t", 4, "Jumlah pengujian latensi")
	flag.IntVar(&task.TestCount, "dn", 10, "Jumlah pengujian unduh")
	flag.IntVar(&downloadTime, "dt", 10, "Durasi pengujian unduh")
	flag.IntVar(&task.TCPPort, "tp", 443, "Port pengujian yang ditentukan")
	flag.StringVar(&task.URL, "url", "https://cf.xiu2.xyz/url", "Alamat pengujian yang ditentukan")

	flag.BoolVar(&task.Httping, "httping", false, "Ganti mode pengujian")
	flag.IntVar(&task.HttpingStatusCode, "httping-code", 0, "Kode status yang valid")
	flag.StringVar(&task.HttpingCFColo, "cfcolo", "", "Cocokkan lokasi tertentu")

	flag.IntVar(&maxDelay, "tl", 9999, "Batas atas latensi rata-rata")
	flag.IntVar(&minDelay, "tll", 0, "Batas bawah latensi rata-rata")
	flag.Float64Var(&maxLossRate, "tlr", 1, "Batas atas tingkat kehilangan paket")
	flag.Float64Var(&task.MinSpeed, "sl", 0, "Batas bawah kecepatan unduh")

	flag.IntVar(&utils.PrintNum, "p", 10, "Jumlah hasil yang ditampilkan")
	flag.StringVar(&task.IPFile, "f", "ip.txt", "File data rentang IP")
	flag.StringVar(&task.IPText, "ip", "", "Data rentang IP yang ditentukan")
	flag.StringVar(&utils.Output, "o", "result.csv", "File hasil output")

	flag.BoolVar(&task.Disable, "dd", false, "Nonaktifkan pengujian unduh")
	flag.BoolVar(&task.TestAll, "allip", false, "Uji semua IP")

	flag.BoolVar(&printVersion, "v", false, "Tampilkan versi program")
	flag.Usage = func() { fmt.Print(help) }
	flag.Parse()

	if task.MinSpeed > 0 && time.Duration(maxDelay)*time.Millisecond == utils.InputMaxDelay {
		fmt.Println("[Tips] Saat menggunakan parameter [-sl], disarankan untuk menggunakan parameter [-tl] untuk menghindari pengujian terus-menerus karena jumlah [-dn] tidak mencukupi...")
	}
	utils.InputMaxDelay = time.Duration(maxDelay) * time.Millisecond
	utils.InputMinDelay = time.Duration(minDelay) * time.Millisecond
	utils.InputMaxLossRate = float32(maxLossRate)
	task.Timeout = time.Duration(downloadTime) * time.Second
	task.HttpingCFColomap = task.MapColoMap()

	if printVersion {
		println(version)
		fmt.Println("Memeriksa pembaruan versi...")
		checkUpdate()
		if versionNew != "" {
			fmt.Printf("*** Ditemukan versi baru [%s] ! Silakan periksa [https://github.com/XIU2/CloudflareSpeedTest] untuk memperbarui! ***", versionNew)
		} else {
			fmt.Println("Saat ini adalah versi terbaru [" + version + "]!")
		}
		os.Exit(0)
	}
}

func main() {
	task.InitRandSeed() // Inisialisasi seed random

	fmt.Printf("# XIU2/CloudflareSpeedTest %s \n\n", version)

	// Mulai pengujian latensi + filter latensi/kehilangan paket
	pingData := task.NewPing().Run().FilterDelay().FilterLossRate()
	// Mulai pengujian unduh
	speedData := task.TestDownloadSpeed(pingData)
	utils.ExportCsv(speedData) // Output file
	speedData.Print()          // Tampilkan hasil

	if versionNew != "" {
		fmt.Printf("\n*** Ditemukan versi baru [%s] ! Silakan periksa [https://github.com/XIU2/CloudflareSpeedTest] untuk memperbarui! ***\n", versionNew)
	}
	endPrint()
}

func endPrint() {
	if utils.NoPrintResult() {
		return
	}
	if runtime.GOOS == "windows" { // Jika sistem Windows, tekan Enter atau Ctrl+C untuk keluar (hindari menutup langsung saat pengujian selesai)
		fmt.Printf("Tekan Enter atau Ctrl+C untuk keluar.")
		fmt.Scanln()
	}
}

// Periksa pembaruan
func checkUpdate() {
	timeout := 10 * time.Second
	client := http.Client{Timeout: timeout}
	res, err := client.Get("https://api.xiu2.xyz/ver/cloudflarespeedtest.txt")
	if err != nil {
		return
	}
	// Baca data body resource: []byte
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}
	// Tutup resource stream
	defer res.Body.Close()
	if string(body) != version {
		versionNew = string(body)
	}
}
