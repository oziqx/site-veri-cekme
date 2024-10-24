package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/proxy" // SOCKS5 proxy için

	"github.com/chromedp/chromedp"
)

func createHTTPClientWithTor() (*http.Client, error) {
	torProxy := "127.0.0.1:9150"

	dialer, err := proxy.SOCKS5("tcp", torProxy, nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("SOCKS5 proxy'ye bağlanılamadı: %v", err)
	}

	httpTransport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
	}

	httpClient := &http.Client{
		Transport: httpTransport,
	}

	return httpClient, nil
}

// html verisini çeker ve döner (proxy ile)
func fetchHTML(url string) (string, error) {
	client, err := createHTTPClientWithTor()
	if err != nil {
		return "", err
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("hata: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("hata: %v", err)
	}

	return string(body), nil
}

// HTML verisini dosyaya kaydeder
func saveHTMLToFile(htmlContent string) error {
	return ioutil.WriteFile("html.txt", []byte(htmlContent), 0644)
}

func extractLinks(htmlContent string) ([]string, error) {
	var links []string
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("hata: %v", err)
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					links = append(links, a.Val)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return links, nil
}

// Linkleri dosyaya kaydeder
func saveLinksToFile(links []string) error {
	return ioutil.WriteFile("links.txt", []byte(strings.Join(links, "\n")), 0644)
}

// Verilen URL'den ekran görüntüsü alır (sadece Tor proxy ile)
func takeScreenshotWithTor(url string) {
	torProxy := "socks5://127.0.0.1:9150"

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ProxyServer(torProxy),
		chromedp.Flag("ignore-certificate-errors", true),
	)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	var buf []byte
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.FullScreenshot(&buf, 90),
	); err != nil {
		fmt.Println("Ekran görüntüsü alınırken hata:", err)
		return
	}

	if err := os.WriteFile("screenshot.png", buf, 0644); err != nil {
		fmt.Println("Ekran görüntüsü kaydedilirken hata:", err)
	}
}

func showHelp() {
	fmt.Println(`
Kullanım:
  go run main.go [SEÇENEKLER] URL

Seçenekler:
  -html          Sayfanın HTML içeriğini çek ve kaydet
  -links         Sayfadan tüm bağlantıları çıkar ve kaydet
  -ss            Sayfanın ekran görüntüsünü al
  -h             Bu yardım mesajını göster
  `)
}

func main() {
	if len(os.Args) < 3 {
		showHelp()
		return
	}

	url := os.Args[2]

	switch os.Args[1] {
	case "-html":
		htmlContent, err := fetchHTML(url)
		if err != nil {
			fmt.Println("Hata:", err)
			return
		}
		err = saveHTMLToFile(htmlContent)
		if err != nil {
			fmt.Println("HTML dosyası kaydedilirken hata:", err)
			return
		}
		fmt.Println("HTML içeriği başarıyla çekildi ve html.txt dosyasına kaydedildi.")
	case "-links":
		htmlContent, err := fetchHTML(url)
		if err != nil {
			fmt.Println("Hata:", err)
			return
		}
		links, err := extractLinks(htmlContent)
		if err != nil {
			fmt.Println("Linkler çıkarılırken hata:", err)
			return
		}
		err = saveLinksToFile(links)
		if err != nil {
			fmt.Println("Linkler dosyası kaydedilirken hata:", err)
			return
		}
		fmt.Println("Bağlantılar başarıyla links.txt dosyasına kaydedildi.")
	case "-ss":
		takeScreenshotWithTor(url)
		fmt.Println("Ekran görüntüsü 'screenshot.png' olarak kaydedildi.")
	case "-h":
		showHelp()
	default:
		showHelp()
	}
}
