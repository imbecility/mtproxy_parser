package providers

import (
	"fmt"
	"github.com/imbecility/mtproxy_parser/config"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func init() {
	Register("YandexStorage_qlinks", withRetry("YandexStorage_qlinks", qlinks, 3))
	Register("YandexStorage_whtprxy", withRetry("YandexStorage_whtprxy", whtprxy, 3))
	Register("YandexStorage_mossad", withRetry("YandexStorage_mossad", FetchYandexMossad, 3))
}

func qlinks() []string {
	return fetchYandexStorage("https://storage.yandexcloud.net/qlinks/proxy.html")
}

func whtprxy() []string {
	return fetchYandexStorage("https://storage.yandexcloud.net/whtprxy/QProxy.html")
}

// FetchYandexMossad парсит прокси из объекта в JS со страницы, формирует строки вида tg://proxy?
func FetchYandexMossad() []string {
	url := "https://storage.yandexcloud.net/ocean/mossad/tg.html"
	log.Printf(" [YandexStorage] загрузка страницы %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", config.UserAgent)

	resp, err := config.HTTPClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			_ = resp.Body.Close()
		}
		log.Printf(" [YandexStorage] ошибка загрузки страницы %s: %v\n", url, err)
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil
	}

	re := regexp.MustCompile(`server:\s*['"]([^'"]+)['"].*?port:\s*['"]?(\d+)['"]?.*?secret:\s*['"]([^'"]+)['"]`)
	matches := re.FindAllStringSubmatch(string(body), -1)

	var results []string
	for _, m := range matches {
		if len(m) >= 4 {
			server := m[1]
			port := m[2]
			secret := m[3]

			link := fmt.Sprintf("tg://proxy?server=%s&port=%s&secret=%s", server, port, secret)
			results = append(results, link)
		}
	}

	return results
}

// fetchYandexStorage извлекает список прокси-серверов a[href=*] тегов со страницы
func fetchYandexStorage(url string) []string {
	log.Printf(" [YandexStorage] загрузка страницы %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", config.UserAgent)

	resp, err := config.HTTPClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			_ = resp.Body.Close()
		}
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil
	}

	var results []string

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			href = strings.TrimSpace(href)
			if strings.HasPrefix(href, "https://t.me/proxy?") {
				href = strings.Replace(href, "https://t.me/proxy?", "tg://proxy?", 1)
			}
			if strings.HasPrefix(href, "tg://proxy?") {
				results = append(results, href)
			}
		}
	})

	return results
}
