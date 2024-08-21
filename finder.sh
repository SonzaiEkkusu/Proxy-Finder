#!/bin/bash
export LANG=en_US.UTF-8
# Fungsi untuk memeriksa dan menginstal paket yang diperlukan
check_and_install() {
    command -v $1 >/dev/null 2>&1 || {
        echo "$1 tidak ditemukan, memasang $1..."
        if [ "$os_type" == "Darwin" ]; then
            brew install $1
        else
            sudo apt-get update
            sudo apt-get install -y $1
        fi
    }
}

os_type=$(uname)
os_arch=$(uname -m)
if [ "$os_type" == "Darwin" ]; then
if [ "$os_arch" == "x86_64" ] || [ "$os_arch" == "i386" ]; then
cpu="amd64a"
elif [ "$os_arch" == "arm64" ]; then
cpu="arm64a"
fi
else
case "$(uname -m)" in
	x86_64 | x64 | amd64 )
	cpu=amd64
	;;
	i386 | i686 )
        cpu=386
	;;
	armv8 | armv8l | arm64 | aarch64 )
        cpu=arm64
	;;
 	armv7l )
        cpu=arm
	;;
	* )
	echo "Arsitektur saat ini adalah $(uname -m), belum didukung."
	exit
	;;
esac
fi
# Memeriksa dan menginstal paket yang diperlukan
check_and_install curl
check_and_install wget
check_and_install unzip
check_and_install awk
check_and_install jq

speedtestrul(){
echo "Apakah Anda ingin menggunakan alamat uji kecepatan lain? (Anda dapat langsung memasukkan alamat uji kecepatan lain [catatan, jangan sertakan http(s)://], dan tekan Enter untuk menggunakan alamat uji kecepatan default)"
read -p "Please enter: " menu
if [ -z $menu ]; then
URL="speed.cloudflare.com/__down?bytes=500000"
else
URL="$menu"
fi
}


gfip(){
if [ ! -f cfcdnip ]; then
curl -L -o cfcdnip -# --retry 2 https://github.com/SonzaiEkkusu/Proxy-Finder/raw/main/tools/linux/linux-$cpu
chmod +x cfcdnip
fi
echo "1、Pilih IPV4 resmi CF"
echo "2、Pilih IPV6 resmi CF (Jaringan lokal harus mendukung IPV6)"
read -p "Silakan pilih: " point
if [ "$point" = "1" ]; then
curl -s -o ip.txt https://raw.githubusercontent.com/SonzaiEkkusu/Proxy-Finder/main/ipv4.txt
elif [ "$point" = "2" ]; then
curl -s -o ip.txt https://raw.githubusercontent.com/SonzaiEkkusu/Proxy-Finder/main/ipv6.txt
else
echo "Input salah, silakan pilih lagi" && gfip
fi
echo "Meskipun 13 port resmi bisa digunakan, pilihlah satu port yang paling sering digunakan sebagai dasar pengujian"
echo "Port dengan TLS aktif: 443, 8443, 2053, 2083, 2087, 2096"
echo "Port dengan TLS nonaktif: 80, 8080, 8880, 2052, 2082, 2086, 2095"
read -p "Pilih salah satu dari 13 port di atas: " point
if ! [[ "$point" =~ ^(2052|2082|2086|2095|80|8880|8080|2053|2083|2087|2096|8443|443)$ ]]; then
echo "Port yang dimasukkan adalah $point, input salah" && cfpoint
fi
if [ "$cpu" = arm64 ] || [ "$cpu" = arm ]; then
echo "Apakah ingin menguji kecepatan? (Pilih 1 untuk uji kecepatan, tekan Enter untuk tidak menguji)"
read -p "Silakan pilih: " menu
if [ -z $menu ]; then
./cfcdnip -tp $point -dd -tl 250
elif [ "$menu" == "1" ];then
speedtestrul
[[ $point =~ 2053|2083|2087|2096|8443|443 ]] && htp=https || htp=http
./cfcdnip -tp $point -url $htp://$URL -sl 2 -tl 250 -dn 5
else 
exit
fi
elif [ "$cpu" = 386 ]; then
echo "Apakah ingin menguji kecepatan? (Pilih 1 untuk uji kecepatan, tekan Enter untuk tidak menguji)"
read -p "Silakan pilih: " menu
if [ -z $menu ]; then
./cfcdnip -tp $point -dd -tl 250
elif [ "$menu" == "1" ];then
speedtestrul
[[ $point =~ 2053|2083|2087|2096|8443|443 ]] && htp=https || htp=http
./cfcdnip -tp $point -url $htp://$URL -sl 2 -tl 250 -dn 5
else 
exit
fi
else
echo "Apakah ingin menguji kecepatan? (Pilih 1 untuk uji kecepatan, tekan Enter untuk tidak menguji)"
read -p "Silakan pilih: " menu
if [ -z $menu ]; then
./cfcdnip -tp $point -dd -tl 250
elif [ "$menu" == "1" ];then
speedtestrul
[[ $point =~ 2053|2083|2087|2096|8443|443 ]] && htp=https || htp=http
./cfcdnip -tp $point -url $htp://$URL -sl 2 -tl 250 -dn 5
else 
exit
fi
fi
}

fdip(){
if [ ! -f cfcdnip ]; then
curl -L -o cfcdnip -# --retry 2 https://github.com/SonzaiEkkusu/Proxy-Finder/raw/main/tools/linux/linux-$cpu
chmod +x cfcdnip
curl -sSLO https://mirror.ghproxy.com/https://raw.githubusercontent.com/SonzaiEkkusu/Proxy-Finder/main/locations.json
fi
echo "Mengunduh file pembaruan proxy IP database txt.zip..."
wget -q https://zip.baipiao.eu.org -O txt.zip
if [ $? -eq 0 ]; then
echo "Unduhan berhasil"
else
curl -L -# --retry 2 https://zip.baipiao.eu.org -o txt.zip
if [ $? -eq 0 ]; then
echo "Unduhan berhasil"
else
echo "Unduhan gagal, melanjutkan dengan database proxy IP sebelumnya"
fi
fi
rm -rf txt
unzip -o txt.zip -d txt > /dev/null 2>&1
if [[ ! -e "txt" ]]; then
echo "Gagal mengunduh file proxy IP txt.zip, harap reset, bersihkan, dan jalankan lagi" && exit
fi
echo "Port dengan TLS aktif: 443, 8443, 2053, 2083, 2087, 2096"
echo "Port dengan TLS nonaktif: 80, 8080, 8880, 2052, 2082, 2086, 2095"
read -p "Pilih salah satu dari 13 port di atas: " point
if ! [[ "$point" =~ ^(2052|2082|2086|2095|80|8880|8080|2053|2083|2087|2096|8443|443)$ ]]; then
echo "Port yang dimasukkan adalah $point, input salah" && cfpoint
fi
if [ "$point" == "443" ]; then
find txt -type f -name "*443*" ! -name "*8443*" -exec cat {} \; > ip.txt
else
find txt -type f -name "*${point}*"  -exec cat {} \; > ip.txt
fi
if [ "$cpu" = arm64 ] || [ "$cpu" = arm ]; then
grep -E '^8|^47|^43|^130|^132|^152|^193|^140|^138|^150|^143|^141|^155|^168|^124|^170|^119' ip.txt | awk '/^124/ { if (++count <= 20) print } /^170/ { if (++count <= 20) print } /^119/ { if (++count <= 20) print } /^8/ { if (++count <= 20) print } /^47/ { if (++count2 <= 20) print } /^43/ { if (++count3 <= 20) print } /^130/ { if (++count4 <= 20) print } /^132/ { if (++count5 <= 20) print } /^152/ { if (++count6 <= 20) print } /^193/ { if (++count7 <= 20) print } /^140/ { if (++count8 <= 20) print } /^138/ { if (++count9 <= 20) print } /^150/ { if (++count10 <= 20) print } /^143/ { if (++count11 <= 20) print } /^141/ { if (++count12 <= 20) print } /^155/ { if (++count13 <= 20) print } /^168/ { if (++count14 <= 20) print }' > pass.txt && mv pass.txt ip.txt
#grep -E '^8|^47|^43|^130|^132|^152|^193|^140|^138|^150|^143|^141|^155|^168|^124|^170|^119' ip.txt > pass.txt && mv pass.txt ip.txt
echo "Apakah ingin menguji kecepatan? (Pilih 1 untuk uji kecepatan, tekan Enter untuk tidak menguji)"
read -p "Silakan pilih: " menu
if [ -z $menu ]; then
./cfcdnip -tp $point -dd -tl 250
elif [ "$menu" == "1" ];then
speedtestrul
[[ $point =~ 2053|2083|2087|2096|8443|443 ]] && htp=https || htp=http
./cfcdnip -tp $point -url $htp://$URL -sl 2 -tl 250 -dn 5
else 
exit
fi
ipcdn1
elif [ "$cpu" = 386 ]; then
#sed "s/$/ ${point}/" ip.txt > passip.txt && mv passip.txt ip.txt
grep -E '^8|^47|^43|^130|^132|^152|^193|^140|^138|^150|^143|^141|^155|^168|^124|^170|^119' ip.txt | awk '/^124/ { if (++count <= 10) print } /^170/ { if (++count <= 10) print } /^119/ { if (++count <= 10) print } /^8/ { if (++count <= 10) print } /^47/ { if (++count2 <= 10) print } /^43/ { if (++count3 <= 10) print } /^130/ { if (++count4 <= 10) print } /^132/ { if (++count5 <= 10) print } /^152/ { if (++count6 <= 10) print } /^193/ { if (++count7 <= 10) print } /^140/ { if (++count8 <= 10) print } /^138/ { if (++count9 <= 10) print } /^150/ { if (++count10 <= 10) print } /^143/ { if (++count11 <= 10) print } /^141/ { if (++count12 <= 10) print } /^155/ { if (++count13 <= 10) print } /^168/ { if (++count14 <= 10) print }' > pass.txt && mv pass.txt ip.txt
#grep -E '^8|^47|^43|^130|^132|^152|^193|^140|^138|^150|^143|^141|^155|^168|^124|^170|^119' ip.txt > pass.txt && mv pass.txt ip.txt
echo "Apakah ingin menguji kecepatan? (Pilih 1 untuk uji kecepatan, tekan Enter untuk tidak menguji)"
read -p "Silakan pilih: " menu
[[ $point =~ 2053|2083|2087|2096|8443|443 ]] && tls=true || tls=false
if [ -z $menu ]; then
./cfcdnip -tls=$tls -speedtest=0 -max=2 -port=$point
elif [ "$menu" == "1" ];then
speedtestrul
./cfcdnip -tls=$tls -speedtest=1 -max=2 -port=$point -url=$URL
else 
exit
fi
ipcdn2
else
#sed "s/$/ ${point}/" ip.txt > passip.txt && mv passip.txt ip.txt
grep -E '^8|^47|^43|^130|^132|^152|^193|^140|^138|^150|^143|^141|^155|^168|^124|^170|^119' ip.txt | awk '/^124/ { if (++count <= 40) print } /^170/ { if (++count <= 40) print } /^119/ { if (++count <= 40) print } /^8/ { if (++count <= 40) print } /^47/ { if (++count2 <= 40) print } /^43/ { if (++count3 <= 40) print } /^130/ { if (++count4 <= 40) print } /^132/ { if (++count5 <= 40) print } /^152/ { if (++count6 <= 40) print } /^193/ { if (++count7 <= 40) print } /^140/ { if (++count8 <= 40) print } /^138/ { if (++count9 <= 40) print } /^150/ { if (++count10 <= 40) print } /^143/ { if (++count11 <= 40) print } /^141/ { if (++count12 <= 40) print } /^155/ { if (++count13 <= 40) print } /^168/ { if (++count14 <= 40) print }' > pass.txt && mv pass.txt ip.txt
#grep -E '^8|^47|^43|^130|^132|^152|^193|^140|^138|^150|^143|^141|^155|^168|^124|^170|^119' ip.txt > pass.txt && mv pass.txt ip.txt
echo "Apakah ingin menguji kecepatan? (Pilih 1 untuk uji kecepatan, tekan Enter untuk tidak menguji)"
read -p "Silakan pilih: " menu
[[ $point =~ 2053|2083|2087|2096|8443|443 ]] && tls=true || tls=false
if [ -z $menu ]; then
./cfcdnip -tls=$tls -speedtest=0 -max=60 -port=$point
elif [ "$menu" == "1" ];then
speedtestrul
./cfcdnip -tls=$tls -max=60 -port=$point -url=$URL
else 
exit
fi
ipcdn2
fi
}

ipcdn1(){
echo
echo "Tunggu 1 menit, melakukan identifikasi daerah untuk IP proxy yang dipilih"
rm -rf cdnIP.csv b.csv a.csv
awk -F ',' 'NR>1 && NR<=101 {print $1}' result.csv > a.csv
while IFS= read -r ip_address; do
UA_Browser="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.87 Safari/537.36"
response=$(curl -s --user-agent "${UA_Browser}" "https://api.ip.sb/geoip/$ip_address" -k | jq -r '"Organization: \(.organization)\nCountry Code: \(.country_code)"')
if [ $? -eq 0 ]; then
echo "Daerah IP $ip_address adalah: $response" | tee -a b.csv
else
echo "Tidak dapat memperoleh informasi daerah untuk IP $ip_address" | tee -a b.csv
fi
sleep 1
done < "a.csv"
grep 'SG' b.csv | head -n 100 >> cdnIP.csv
grep 'US' b.csv | head -n 100 >> cdnIP.csv
grep 'ID' b.csv | head -n 100 >> cdnIP.csv
echo
echo "IP proxy terbaik di daerah anda adalah sebagai berikut:"
cat cdnIP.csv
}

ipcdn2(){
rm -rf cdnIP.csv
{
  grep 'SIN' ip.csv | head -n 100
  echo
  grep 'LHR' ip.csv | head -n 100
  echo
  grep 'SJC' ip.csv | head -n 100
  echo
  grep 'CGK' ip.csv | head -n 100
  echo
  grep 'HLP' ip.csv | head -n 100
  echo
} >> cdnIP.csv
echo
echo "IP proxy terbaik di daerah anda adalah sebagai berikut:"
cat cdnIP.csv
}

rmrf(){
rm -rf txt txt.zip ip.txt ipv6.txt cfcdnip result.csv cdnIP.csv a.csv b.csv ip.csv locations.json
}

echo "------------------------------------------------------"
echo "	      Cloudflare IP Proxy Finder"
echo "------------------------------------------------------"
echo "Telegram  : t.me/November2k"
echo "Github    : github.com/SonzaiEkkusu"
echo "Github    : github.com/SonzaiX"
echo "------------------------------------------------------"
echo "1. Cari IP resmi CF"
echo "2. Cari IP proxy CF"
echo "0. Keluar"
read -p "Silakan pilih: " menu
if [ "$menu" == "1" ];then
rmrf && gfip
elif [ "$menu" == "2" ];then
rmrf && fdip
else 
exit
fi
