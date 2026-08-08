package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// Is machine ka LAN IP dhundo, taki phone se bhi blog khul sake.
// Ye poore project ka sabse chhota par sabse "wah" wala feature hai -
// judge apna phone nikal ke QR scan karta hai aur blog khul jata hai.
func lanIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	var fallback string
	for _, iface := range ifaces {
		// band ya loopback interface se kaam nahi banega
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok || ipnet.IP.IsLoopback() {
				continue
			}
			ip := ipnet.IP.To4()
			if ip == nil {
				continue // IPv6 se QR bahut bada ban jata hai, rehne do
			}

			s := ip.String()
			// Docker aur VM ke virtual interfaces (172.17.x, 169.254.x) ko
			// aakhir me rakho - wo aam taur pe phone se reachable nahi hote.
			if strings.HasPrefix(s, "172.1") || strings.HasPrefix(s, "169.254") {
				if fallback == "" {
					fallback = s
				}
				continue
			}
			return s
		}
	}
	return fallback
}

func lanURL(port int) string {
	ip := lanIP()
	if ip == "" {
		return ""
	}
	return fmt.Sprintf("http://%s:%d", ip, port)
}

// QR image bana ke bhej do. Har request pe naya banate hain -
// ye 2ms ka kaam hai, cache karne ka jhanjhat lene ka matlab nahi.
func handleQR(w http.ResponseWriter, r *http.Request) {
	url := lanURL(currentPort)
	if url == "" {
		// wifi se connected nahi ho? Toh localhost ka hi QR de do,
		// kam se kam isi machine pe kaam karega.
		url = fmt.Sprintf("http://localhost:%d", currentPort)
	}

	png, err := qrcode.Encode(url, qrcode.Medium, 300)
	if err != nil {
		http.Error(w, "QR nahi bana: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store") // IP badal sakta hai wifi change pe
	w.Write(png)
}
