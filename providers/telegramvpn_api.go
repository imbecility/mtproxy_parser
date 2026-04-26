package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/imbecility/mtproxy_parser/config"
	"github.com/imbecility/mtproxy_parser/types"
	"github.com/imbecility/mtproxy_parser/utils"
)

func init() {
	Register("TelegramVPN_API", withRetry("TelegramVPN_API", fetchTelegramVPN, 3))
}

// fetchTelegramVPN извлекает список URL-адресов MTProto-прокси из API telegramvpn.org
func fetchTelegramVPN() []string {
	utils.CheckApiTelegramVpnOrgAvailability()
	if config.ApiTelegramVpnOrgBaseUrl == "" {
		return nil
	}
	var results []string
	page := 1
	for {
		url := fmt.Sprintf("%s/proxies?page=%d&per=50&sort=ping&status=all", config.ApiTelegramVpnOrgBaseUrl, page)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			break
		}
		req.Header.Set("User-Agent", config.UserAgent)

		resp, err := config.HTTPClient.Do(req)
		if err != nil {
			log.Printf(" [TelegramVPN_API] ошибка запроса: %v\n", err)
			break
		}

		if resp.StatusCode != 200 {
			_ = resp.Body.Close()
			break
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			break
		}

		var apiData types.VpnAPIResponse
		if err := json.Unmarshal(body, &apiData); err != nil {
			log.Printf(" [TelegramVPN_API] ошибка парсинга JSON: %v\n", err)
			break
		}
		fmt.Printf("\r [TelegramVPN_API] загрузка страницы %d/%d", page, apiData.Pages)
		for _, p := range apiData.Proxies {
			// только mtproto
			if p.Type == "mtproto" {
				results = append(results, p.URL)
			}
		}

		if page >= apiData.Pages {
			fmt.Printf("\r [TelegramVPN_API] загрузка страницы %d/%d", apiData.Pages, apiData.Pages)
			fmt.Printf("\n [TelegramVPN_API] обход всех страниц завершен\n")
			break
		}

		page++
		time.Sleep(500 * time.Millisecond)
	}

	return results
}
