package providers

import (
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/imbecility/mtproxy_parser/config"
	"github.com/imbecility/mtproxy_parser/utils"

	"github.com/PuerkitoBio/goquery"
)

func init() {
	Register("MtprotoRu_Background", fetchMtprotoRu)
}

// fetchMtprotoRu осуществляет непрерывный сбор уникальных прокси-ссылок по одной с ресурса mtproto.ru:
// запускается в фоне до сигнала отмены через контекст и возвращает список накопленных адресов.
func fetchMtprotoRu() []string {
	utils.CheckMtprotoRuAvailability()
	if config.MtprotoRuBaseUrl == "" {
		return nil
	}

	url := config.MtprotoRuBaseUrl + "/personal.php"
	uniqueProxies := make(map[string]struct{})
	iterations := 0

	log.Println(" [MtprotoRu] запущен в фоне для итеративного получения прокси по одной...")

	for {
		select {
		case <-config.BgCtx.Done():
			var results []string
			for p := range uniqueProxies {
				results = append(results, p)
			}
			log.Printf(" [MtprotoRu] фоновый сбор остановлен, итераций пройдено: %d, добыто прокси: %d\n", iterations, len(results))
			return results

		default:
			// контекстно-завимимый запрос для мгновенной отмены
			req, err := http.NewRequestWithContext(config.BgCtx, "GET", url, nil)
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
			req.Header.Set("User-Agent", config.UserAgent)

			resp, err := config.HTTPClient.Do(req)
			if err != nil || resp.StatusCode != 200 {
				if resp != nil {
					_ = resp.Body.Close()
				}
				time.Sleep(1 * time.Second)
				continue
			}

			doc, err := goquery.NewDocumentFromReader(resp.Body)
			_ = resp.Body.Close()
			if err == nil {
				doc.Find("a").Each(func(_ int, s *goquery.Selection) {
					href, exists := s.Attr("href")
					if exists {
						href = strings.TrimSpace(href)
						if strings.HasPrefix(href, "https://t.me/proxy?") {
							href = strings.Replace(href, "https://t.me/proxy?", "tg://proxy?", 1)
						}
						if strings.HasPrefix(href, "tg://proxy?") {
							uniqueProxies[href] = struct{}{}
						}
					}
				})
			}

			iterations++
			// рандомная пауза от 1.5 до 2.01 секунд чтобы не дудосить сайт
			sleep := rand.Intn(522) + 1500
			time.Sleep(time.Duration(sleep) * time.Millisecond)
		}
	}
}
