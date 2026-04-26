package providers

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/imbecility/mtproxy_parser/config"
	"github.com/imbecility/mtproxy_parser/utils"

	"github.com/PuerkitoBio/goquery"
)

func init() {
	Register("MtproXyz", withRetry("MtproXyz", fetchMtproXyz, 3))
}

// fetchMtproXyz извлекает список прокси-серверов с ресурса mtpro.xyz:
// парсит главную страницу, ищет JavaScript-файл и декодирует из него списки прокси.
func fetchMtproXyz() []string {
	utils.CheckMtproXyzAvailability()
	if config.MtproXyzBaseUrl == "" {
		return nil
	}

	log.Println(" [MtproXyz] загрузка главной...")

	req, err := http.NewRequest("GET", config.MtproXyzBaseUrl+"/mtproto", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", config.UserAgent)

	resp, err := config.HTTPClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		log.Printf(" [MtproXyz] ошибка загрузки страницы: %v\n", err)
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

	var jsURL string
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if exists && strings.Contains(src, "autoptimize/js/") {
			jsURL = src
			if strings.HasPrefix(jsURL, "/") {
				jsURL = config.MtproXyzBaseUrl + jsURL
			}
		}
	})

	if jsURL == "" {
		log.Println(" [MtproXyz] ошибка: JS файл с данными не найден на странице.")
		return nil
	}

	log.Printf(" [MtproXyz] загрузка и парсинг JS-файла\n")

	jsReq, _ := http.NewRequest("GET", jsURL, nil)
	jsReq.Header.Set("User-Agent", config.UserAgent)

	jsResp, err := config.HTTPClient.Do(jsReq)
	if err != nil || jsResp.StatusCode != 200 {
		log.Printf(" [MtproXyz] ошибка загрузки JS файла: %v\n", err)
		if jsResp != nil {
			_ = jsResp.Body.Close()
		}
		return nil
	}

	jsBytes, err := io.ReadAll(jsResp.Body)
	_ = jsResp.Body.Close()
	if err != nil {
		return nil
	}

	return utils.ExtractProxiesFromJS(string(jsBytes), "MtproXyz")
}
