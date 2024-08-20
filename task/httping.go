package task

import (
	//"crypto/tls"
	//"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	Httping           bool
	HttpingStatusCode int
	HttpingCFColo     string
	HttpingCFColomap  *sync.Map
	OutRegexp         = regexp.MustCompile(`[A-Z]{3}`)
)

// pingReceived pingTotalTime
func (p *Ping) httping(ip *net.IPAddr) (int, time.Duration) {
	hc := http.Client{
		Timeout: time.Second * 2,
		Transport: &http.Transport{
			DialContext: getDialContext(ip),
			//TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Lewati verifikasi sertifikat
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Cegah pengalihan
		},
	}

	// Kunjungi sekali dulu untuk mendapatkan kode status HTTP dan Cloudflare Colo
	{
		requ, err := http.NewRequest(http.MethodHead, URL, nil)
		if err != nil {
			return 0, 0
		}
		requ.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, seperti Gecko) Chrome/98.0.4758.80 Safari/537.36")
		resp, err := hc.Do(requ)
		if err != nil {
			return 0, 0
		}
		defer resp.Body.Close()

		//fmt.Println("IP:", ip, "StatusCode:", resp.StatusCode, resp.Request.URL)
		// Jika kode status HTTP tidak ditentukan, atau kode status yang ditentukan tidak valid, maka default hanya 200, 301, 302 yang dianggap berhasil
		if HttpingStatusCode == 0 || HttpingStatusCode < 100 && HttpingStatusCode > 599 {
			if resp.StatusCode != 200 && resp.StatusCode != 301 && resp.StatusCode != 302 {
				return 0, 0
			}
		} else {
			if resp.StatusCode != HttpingStatusCode {
				return 0, 0
			}
		}

		io.Copy(io.Discard, resp.Body)

		// Hanya jika daerah tertentu ditentukan maka cocokkan dengan kode tiga huruf bandara
		if HttpingCFColo != "" {
			// Menentukan Cloudflare atau AWS CloudFront berdasarkan header Server dan menetapkan cfRay ke kode tiga huruf bandara masing-masing
			cfRay := func() string {
				if resp.Header.Get("Server") == "cloudflare" {
					return resp.Header.Get("CF-RAY") // Contoh cf-ray: 7bd32409eda7b020-SJC
				}
				return resp.Header.Get("x-amz-cf-pop") // Contoh X-Amz-Cf-Pop: SIN52-P1
			}()
			colo := p.getColo(cfRay)
			if colo == "" { // Jika tidak cocok dengan kode tiga huruf atau tidak sesuai dengan daerah tertentu, akhiri pengujian IP ini
				return 0, 0
			}
		}

	}

	// Ulangi pengujian untuk menghitung latensi
	success := 0
	var delay time.Duration
	for i := 0; i < PingTimes; i++ {
		requ, err := http.NewRequest(http.MethodHead, URL, nil)
		if err != nil {
			log.Fatal("Kesalahan yang tidak terduga, harap laporkan: ", err)
			return 0, 0
		}
		requ.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, seperti Gecko) Chrome/98.0.4758.80 Safari/537.36")
		if i == PingTimes-1 {
			requ.Header.Set("Connection", "close")
		}
		startTime := time.Now()
		resp, err := hc.Do(requ)
		if err != nil {
			continue
		}
		success++
		io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		duration := time.Since(startTime)
		delay += duration

	}

	return success, delay

}

func MapColoMap() *sync.Map {
	if HttpingCFColo == "" {
		return nil
	}
	// Ubah kode tiga huruf daerah yang ditentukan menjadi huruf besar dan formatkan
	colos := strings.Split(strings.ToUpper(HttpingCFColo), ",")
	colomap := &sync.Map{}
	for _, colo := range colos {
		colomap.Store(colo, colo)
	}
	return colomap
}

func (p *Ping) getColo(b string) string {
	if b == "" {
		return ""
	}
	// Cocokkan dengan kode tiga huruf bandara menggunakan regex dan kembalikan hasilnya
	out := OutRegexp.FindString(b)

	if HttpingCFColomap == nil {
		return out
	}
	// Cocokkan kode tiga huruf bandara dengan daerah yang ditentukan
	_, ok := HttpingCFColomap.Load(out)
	if ok {
		return out
	}

	return ""
}