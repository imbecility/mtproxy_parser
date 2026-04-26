package providers

import (
	"io"
	"log"
	"net/http"

	"github.com/imbecility/mtproxy_parser/config"
	"github.com/imbecility/mtproxy_parser/utils"
)

func init() {
	Register("Hookzof", withRetry("Hookzof", fetchHookzof, 3))
}

// fetchHookzof загружает страницу со списком tg-прокси из Hookzof,
// извлекает их из JavaScript, возвращает список найденных прокси или nil в случае ошибок
func fetchHookzof() []string {
	url := "https://hookzof.github.io/mtpro.xyz/mtproto.html"
	log.Println(" [Hookzof] загрузка страницы с инлайн-скриптом...")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", config.UserAgent)

	resp, err := config.HTTPClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		log.Printf(" [Hookzof] ошибка загрузки страницы: %v\n", err)
		if resp != nil {
			_ = resp.Body.Close()
		}
		return nil
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil
	}

	return utils.ExtractProxiesFromJS(string(bodyBytes), "Hookzof")
}
