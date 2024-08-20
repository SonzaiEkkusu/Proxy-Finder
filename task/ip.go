package task

import (
	"bufio"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultInputFile = "ip.txt"

var (
	// TestAll test all ip
	TestAll = false
	// IPFile is the filename of IP Rangs
	IPFile = defaultInputFile
	IPText string
)

func InitRandSeed() {
	rand.Seed(time.Now().UnixNano())
}

func isIPv4(ip string) bool {
	return strings.Contains(ip, ".")
}

func randIPEndWith(num byte) byte {
	if num == 0 { // Untuk /32 yang merupakan IP tunggal
		return byte(0)
	}
	return byte(rand.Intn(int(num)))
}

type IPRanges struct {
	ips     []*net.IPAddr
	mask    string
	firstIP net.IP
	ipNet   *net.IPNet
}

func newIPRanges() *IPRanges {
	return &IPRanges{
		ips: make([]*net.IPAddr, 0),
	}
}

// Jika itu adalah IP tunggal maka tambahkan subnet mask, jika tidak, maka dapatkan subnet mask (r.mask)
func (r *IPRanges) fixIP(ip string) string {
	// Jika tidak mengandung '/' maka itu bukan rentang IP, melainkan IP tunggal, sehingga perlu menambahkan /32 /128 subnet mask
	if i := strings.IndexByte(ip, '/'); i < 0 {
		if isIPv4(ip) {
			r.mask = "/32"
		} else {
			r.mask = "/128"
		}
		ip += r.mask
	} else {
		r.mask = ip[i:]
	}
	return ip
}

// Mem-parse rentang IP, mendapatkan IP, rentang IP, subnet mask
func (r *IPRanges) parseCIDR(ip string) {
	var err error
	if r.firstIP, r.ipNet, err = net.ParseCIDR(r.fixIP(ip)); err != nil {
		log.Fatalln("ParseCIDR err", err)
	}
}

func (r *IPRanges) appendIPv4(d byte) {
	r.appendIP(net.IPv4(r.firstIP[12], r.firstIP[13], r.firstIP[14], d))
}

func (r *IPRanges) appendIP(ip net.IP) {
	r.ips = append(r.ips, &net.IPAddr{IP: ip})
}

// Mengembalikan nilai minimum dari bagian keempat IP dan jumlah host yang dapat digunakan
func (r *IPRanges) getIPRange() (minIP, hosts byte) {
	minIP = r.firstIP[15] & r.ipNet.Mask[3] // Nilai minimum dari bagian keempat IP

	// Mendapatkan jumlah host berdasarkan subnet mask
	m := net.IPv4Mask(255, 255, 255, 255)
	for i, v := range r.ipNet.Mask {
		m[i] ^= v
	}
	total, _ := strconv.ParseInt(m.String(), 16, 32) // Jumlah IP yang dapat digunakan
	if total > 255 {                                 // Koreksi jumlah IP yang dapat digunakan pada bagian keempat
		hosts = 255
		return
	}
	hosts = byte(total)
	return
}

func (r *IPRanges) chooseIPv4() {
	if r.mask == "/32" { // Jika IP tunggal, tidak perlu acak, langsung tambahkan
		r.appendIP(r.firstIP)
	} else {
		minIP, hosts := r.getIPRange()    // Mengembalikan nilai minimum dari bagian keempat IP dan jumlah host yang dapat digunakan
		for r.ipNet.Contains(r.firstIP) { // Selama IP tidak melebihi rentang IP, lanjutkan looping untuk memilih secara acak
			if TestAll { // Jika menguji semua IP
				for i := 0; i <= int(hosts); i++ { // Iterasi dari nilai minimum hingga maksimum dari bagian keempat IP
					r.appendIPv4(byte(i) + minIP)
				}
			} else { // Bagian keempat IP acak 0.0.0.X
				r.appendIPv4(minIP + randIPEndWith(hosts))
			}
			r.firstIP[14]++ // 0.0.(X+1).X
			if r.firstIP[14] == 0 {
				r.firstIP[13]++ // 0.(X+1).X.X
				if r.firstIP[13] == 0 {
					r.firstIP[12]++ // (X+1).X.X.X
				}
			}
		}
	}
}

func (r *IPRanges) chooseIPv6() {
	if r.mask == "/128" { // Jika IP tunggal, tidak perlu acak, langsung tambahkan
		r.appendIP(r.firstIP)
	} else {
		var tempIP uint8                  // Variabel sementara, untuk menyimpan nilai dari digit sebelumnya
		for r.ipNet.Contains(r.firstIP) { // Selama IP tidak melebihi rentang IP, lanjutkan looping untuk memilih secara acak
			r.firstIP[15] = randIPEndWith(255) // Acak bagian terakhir dari IP
			r.firstIP[14] = randIPEndWith(255) // Acak bagian kedua terakhir dari IP

			targetIP := make([]byte, len(r.firstIP))
			copy(targetIP, r.firstIP)
			r.appendIP(targetIP) // Tambahkan ke kumpulan alamat IP

			for i := 13; i >= 0; i-- { // Mulai dari digit ketiga dari belakang, maju ke depan secara acak
				tempIP = r.firstIP[i]              // Simpan nilai dari digit sebelumnya
				r.firstIP[i] += randIPEndWith(255) // Acak 0~255, tambahkan ke digit saat ini
				if r.firstIP[i] >= tempIP {        // Jika nilai digit saat ini lebih besar dari atau sama dengan digit sebelumnya, berarti acak berhasil, dan dapat keluar dari loop
					break
				}
			}
		}
	}
}

func loadIPRanges() []*net.IPAddr {
	ranges := newIPRanges()
	if IPText != "" { // Dapatkan data rentang IP dari parameter
		IPs := strings.Split(IPText, ",") // Pisahkan dengan koma menjadi array dan iterasi
		for _, IP := range IPs {
			IP = strings.TrimSpace(IP) // Hapus karakter kosong (spasi, tab, newline, dll.) di awal dan akhir
			if IP == "" {              // Lewati jika kosong (yaitu, di awal, akhir, atau jika ada banyak ,,)
				continue
			}
			ranges.parseCIDR(IP) // Mem-parse rentang IP, mendapatkan IP, rentang IP, subnet mask
			if isIPv4(IP) {      // Menghasilkan semua alamat IPv4 / IPv6 untuk diuji (tunggal/acak/semua)
				ranges.chooseIPv4()
			} else {
				ranges.chooseIPv6()
			}
		}
	} else { // Dapatkan data rentang IP dari file
		if IPFile == "" {
			IPFile = defaultInputFile
		}
		file, err := os.Open(IPFile)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() { // Iterasi setiap baris file
			line := strings.TrimSpace(scanner.Text()) // Hapus karakter kosong (spasi, tab, newline, dll.) di awal dan akhir
			if line == "" {                           // Lewati jika kosong
				continue
			}
			ranges.parseCIDR(line) // Mem-parse rentang IP, mendapatkan IP, rentang IP, subnet mask
			if isIPv4(line) {      // Menghasilkan semua alamat IPv4 / IPv6 untuk diuji (tunggal/acak/semua)
				ranges.chooseIPv4()
			} else {
				ranges.chooseIPv6()
			}
		}
	}
	return ranges.ips
}

