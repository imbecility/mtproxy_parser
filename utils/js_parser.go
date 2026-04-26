package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/imbecility/mtproxy_parser/types"
	"log"
	"regexp"
)

// ExtractProxiesFromJS находит Base64 в atob(), декодирует его, достает JSON и формирует ссылки
func ExtractProxiesFromJS(content string, providerName string) []string {
	reAtob := regexp.MustCompile(`atob\(['"]([^'"]+)['"]\)`)
	matches := reAtob.FindStringSubmatch(content)

	if len(matches) < 2 {
		log.Printf(" [%s] Base64 строка не найдена (atob не найден).\n", providerName)
		return nil
	}

	b64String := matches[1]
	decodedBytes, err := base64.StdEncoding.DecodeString(b64String)
	if err != nil {
		log.Printf(" [%s] ошибка декодирования Base64: %v\n", providerName, err)
		return nil
	}
	decodedJS := string(decodedBytes)

	reJSON := regexp.MustCompile(`(?s)\[\s*\{.*?}\s*]`)
	jsonStr := reJSON.FindString(decodedJS)

	if jsonStr == "" {
		log.Printf(" [%s] JSON массив внутри расшифрованного кода не найден.\n", providerName)
		return nil
	}

	var proxyList []types.ProxyData
	if err := json.Unmarshal([]byte(jsonStr), &proxyList); err != nil {
		log.Printf(" [%s] ошибка парсинга JSON: %v\n", providerName, err)
		return nil
	}

	var results []string
	for _, p := range proxyList {
		link := fmt.Sprintf("tg://proxy?server=%s&port=%d&secret=%s", p.Host, p.Port, p.Secret)
		results = append(results, link)
	}

	return results
}
