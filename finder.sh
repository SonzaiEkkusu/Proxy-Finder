#!/bin/bash
export LANG=en_US.UTF-8
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
            echo "The current architecture is $(uname -m), which is not supported."
            exit
            ;;
    esac
fi

speedtestrul(){
    echo "Do you want to use another speed test address? (You can directly enter another speed test address [Note: do not include http(s)://], press Enter to use the default address)"
    read -p "Please enter:" menu
    if [ -z $menu ]; then
        URL="t2.geigei.gq"
    else
        URL="$menu"
    fi
}

fdip(){
    if [ ! -f cfcdnip ]; then
        curl -L -o cfcdnip -# --retry 2 https://gitlab.com/rwkgyg/CFwarp/-/raw/main/point/cpu2/$cpu
        chmod +x cfcdnip
        curl -sSLO https://mirror.ghproxy.com/https://raw.githubusercontent.com/yonggekkk/Cloudflare-workers-pages-vless/main/locations.json
    fi
    echo "Downloading updated reverse proxy IP library txt.zip file..."
    wget -q https://zip.baipiao.eu.org -O txt.zip
    if [ $? -eq 0 ]; then
        echo "Download successful"
    else
        curl -L -# --retry 2 https://cf.yg-kkk.gq -o txt.zip
        if [ $? -eq 0 ]; then
            echo "Download successful"
        else
            echo "Download failed, continuing with the previous reverse proxy IP library"
        fi
    fi
    rm -rf txt
    unzip -o txt.zip -d txt > /dev/null 2>&1
    if [[ ! -e "txt" ]]; then
        echo "Failed to download the reverse proxy IP library txt.zip file, please reset and clean up and run again" && exit
    fi
    echo "Ports with TLS enabled: 443, 8443, 2053, 2083, 2087, 2096"
    echo "Ports without TLS: 80, 8080, 8880, 2052, 2082, 2086, 2095"
    read -p "Please select one of the above 13 ports:" point
    if ! [[ "$point" =~ ^(2052|2082|2086|2095|80|8880|8080|2053|2083|2087|2096|8443|443)$ ]]; then
        echo "The entered port is $point, input error" && cfpoint
    fi
    if [ "$point" == "443" ]; then
        find txt -type f -name "*443*" ! -name "*8443*" -exec cat {} \; > ip.txt
    else
        find txt -type f -name "*${point}*"  -exec cat {} \; > ip.txt
    fi
    if [ "$cpu" = arm64 ] || [ "$cpu" = arm ]; then
        grep -E '^8|^47|^43|^130|^132|^152|^193|^140|^138|^150|^143|^141|^155|^168|^124|^170|^119' ip.txt | awk '/^124/ { if (++count <= 20) print } /^170/ { if (++count <= 20) print } /^119/ { if (++count <= 20) print } /^8/ { if (++count <= 20) print } /^47/ { if (++count2 <= 20) print } /^43/ { if (++count3 <= 20) print } /^130/ { if (++count4 <= 20) print } /^132/ { if (++count5 <= 20) print } /^152/ { if (++count6 <= 20) print } /^193/ { if (++count7 <= 20) print } /^140/ { if (++count8 <= 20) print } /^138/ { if (++count9 <= 20) print } /^150/ { if (++count10 <= 20) print } /^143/ { if (++count11 <= 20) print } /^141/ { if (++count12 <= 20) print } /^155/ { if (++count13 <= 20) print } /^168/ { if (++count14 <= 20) print }' > pass.txt && mv pass.txt ip.txt
        echo "Do you want to test speed? (Select 1 for speed test, press Enter for no speed test)"
        read -p "Please select:" menu
        if [ -z $menu ]; then
            ./cfcdnip -tls=$tls -speedtest=0 -max=2 -port=$point
        elif [ "$menu" == "1" ]; then
            speedtestrul
            ./cfcdnip -tls=$tls -speedtest=1 -max=2 -port=$point -url=$URL
        else
            exit
        fi
        ipcdn1
    elif [ "$cpu" = 386 ]; then
        grep -E '^8|^47|^43|^130|^132|^152|^193|^140|^138|^150|^143|^141|^155|^168|^124|^170|^119' ip.txt | awk '/^124/ { if (++count <= 10) print } /^170/ { if (++count <= 10) print } /^119/ { if (++count <= 10) print } /^8/ { if (++count <= 10) print } /^47/ { if (++count2 <= 10) print } /^43/ { if (++count3 <= 10) print } /^130/ { if (++count4 <= 10) print } /^132/ { if (++count5 <= 10) print } /^152/ { if (++count6 <= 10) print } /^193/ { if (++count7 <= 10) print } /^140/ { if (++count8 <= 10) print } /^138/ { if (++count9 <= 10) print } /^150/ { if (++count10 <= 10) print } /^143/ { if (++count11 <= 10) print } /^141/ { if (++count12 <= 10) print } /^155/ { if (++count13 <= 10) print } /^168/ { if (++count14 <= 10) print }' > pass.txt && mv pass.txt ip.txt
        echo "Do you want to test speed? (Select 1 for speed test, press Enter for no speed test)"
        read -p "Please select:" menu
        [[ $point =~ 2053|2083|2087|2096|8443|443 ]] && tls=true || tls=false
        if [ -z $menu ]; then
            ./cfcdnip -tls=$tls -speedtest=0 -max=2 -port=$point
        elif [ "$menu" == "1" ]; then
            speedtestrul
            ./cfcdnip -tls=$tls -speedtest=1 -max=2 -port=$point -url=$URL
        else 
            exit
        fi
        ipcdn2
    else
        grep -E '^8|^47|^43|^130|^132|^152|^193|^140|^138|^150|^143|^141|^155|^168|^124|^170|^119' ip.txt | awk '/^124/ { if (++count <= 40) print } /^170/ { if (++count <= 40) print } /^119/ { if (++count <= 40) print } /^8/ { if (++count <= 40) print } /^47/ { if (++count2 <= 40) print } /^43/ { if (++count3 <= 40) print } /^130/ { if (++count4 <= 40) print } /^132/ { if (++count5 <= 40) print } /^152/ { if (++count6 <= 40) print } /^193/ { if (++count7 <= 40) print } /^140/ { if (++count8 <= 40) print } /^138/ { if (++count9 <= 40) print } /^150/ { if (++count10 <= 40) print } /^143/ { if (++count11 <= 40) print } /^141/ { if (++count12 <= 40) print } /^155/ { if (++count13 <= 40) print } /^168/ { if (++count14 <= 40) print }' > pass.txt && mv pass.txt ip.txt
        echo "Do you want to test speed? (Select 1 for speed test, press Enter for no speed test)"
        read -p "Please select:" menu
        [[ $point =~ 2053|2083|2087|2096|8443|443 ]] && tls=true || tls=false
        if [ -z $menu ]; then
            ./cfcdnip -tls=$tls -speedtest=0 -max=60 -port=$point
        elif [ "$menu" == "1" ]; then
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
    echo "Please wait for 1 minute while we identify the regions of the preferred reverse proxy IPs and rank them"
    rm -rf cdnIP.csv b.csv a.csv
    awk -F ',' 'NR>1 && NR<=101 {print $1}' result.csv > a.csv
    while IFS= read -r ip_address; do
        UA_Browser="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.87 Safari/537.36"
        response=$(curl -s --user-agent "${UA_Browser}" "https://api.ip.sb/geoip/$ip_address" -k | awk -F "country_code" '{print $2}' | awk -F'":"|","|"' '{print $2}')
        if [ $? -eq 0 ]; then
            echo "The region of IP address $ip_address is: $response" | tee -a b.csv
        else
            echo "Unable to retrieve region information for IP address $ip_address" | tee -a b.csv
        fi
        sleep 1
    done < "a.csv"
    
    # Adding countries with respective regions to the CSV file
    grep 'SG' b.csv | head -n 3 >> cdnIP.csv
    grep 'HK' b.csv | head -n 3 >> cdnIP.csv
    grep 'JP' b.csv | head -n 3 >> cdnIP.csv
    grep 'KR' b.csv | head -n 3 >> cdnIP.csv
    grep 'TW' b.csv | head -n 3 >> cdnIP.csv
    grep 'US' b.csv | head -n 3 >> cdnIP.csv
    grep 'GB' b.csv | head -n 3 >> cdnIP.csv
    grep 'DE' b.csv | head -n 3 >> cdnIP.csv
    grep 'NL' b.csv | head -n 3 >> cdnIP.csv
    grep 'FR' b.csv | head -n 3 >> cdnIP.csv

    echo
    echo "The top three preferred IPs by region are as follows:"
    cat cdnIP.csv
}

ipcdn2(){
    rm -rf cdnIP.csv
    {
        grep 'HKG' ip.csv | head -n 3
        echo
        grep 'NRT' ip.csv | head -n 3
        echo
        grep 'KIX' ip.csv | head -n 3
        echo
        grep 'SIN' ip.csv | head -n 3
        echo
        grep 'ICN' ip.csv | head -n 3
        echo
        grep 'FRA' ip.csv | head -n 3
        echo
        grep 'LHR' ip.csv | head -n 3
        echo
        grep 'SJC' ip.csv | head -n 3
        echo
    } >> cdnIP.csv
    echo
    echo "The top three preferred IPs by region are as follows:"
    cat cdnIP.csv
}

rmrf(){
    rm -rf txt txt.zip ip.txt ipv6.txt cfcdnip result.csv cdnIP.csv a.csv b.csv ip.csv
}

echo "------------------------------------------------------"
echo "	      Cloudflare IP Proxy Finder"
echo "------------------------------------------------------"
echo "Telegram  : t.me/November2k"
echo "Github    : github.com/SonzaiEkkusu"
echo "Github    : github.com/SonzaiX"
echo "------------------------------------------------------"
echo "NOTE : the results are not 100% accurate so it is important to check them directly from various places"
echo "Do you want to continue?"
echo
echo "Type Y to Continue"
echo "Type N to Exit"
read -p "Please choose:" menu
if [ "$menu" == "Y" ]; then
    rmrf && fdip
else 
    exit
fi
