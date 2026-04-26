package providers

import (
	"bufio"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/imbecility/mtproxy_parser/config"
	"github.com/imbecility/mtproxy_parser/utils"

	"github.com/PuerkitoBio/goquery"
)

func init() {
	Register("TelegramChannels", fetchTelegramChannels)
}

// fetchTelegramChannels парсит прокси в публичных Telegram-каналах, из сообщений не старше config.DaysToParse,
// поддерживает загрузку дополнительных каналов из файла.
func fetchTelegramChannels() []string {
	utils.CheckTelegramWebAvailability()
	if config.TelegramBaseUrl == "" {
		return nil
	}
	cutoffDate := time.Now().AddDate(0, 0, -config.DaysToParse)

	if config.CustomTgChannels != "" && utils.IsFile(config.CustomTgChannels) {
		err := LoadTgChannelsFromFile(config.CustomTgChannels)
		if err != nil {
			log.Printf(" [Telegram] не удалось добавить кастомные TG-каналы из файла '%s': %v", config.CustomTgChannels, err)
		}
	}
	var allProxies []string
	for _, channel := range config.TgChannels {
		log.Printf(" [Telegram] парсинг канала: %s\n", channel)
		proxies := parseTelegramChannel(channel, cutoffDate)
		allProxies = append(allProxies, proxies...)
		time.Sleep(1000 * time.Millisecond)
	}
	return allProxies
}

// parseTelegramChannel извлекает ссылки на прокси из публичного веб-превью Telegram-канала,
// обходя сообщения в обратном порядке до даты отсечки cutoffDate, ссылки нормализуются к единому формату tg://proxy?.
func parseTelegramChannel(channel string, cutoffDate time.Time) []string {
	var proxies []string
	baseURL := fmt.Sprintf("%s/s/%s", config.TelegramBaseUrl, channel)
	currentURL := baseURL

	for {
		req, err := http.NewRequest("GET", currentURL, nil)
		if err != nil {
			break
		}
		req.Header.Set("User-Agent", config.UserAgent)

		resp, err := config.HTTPClient.Do(req)
		if err != nil {
			break
		}

		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			break
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		_ = resp.Body.Close()

		if err != nil {
			break
		}

		messages := doc.Find(".js-widget_message")
		if messages.Length() == 0 {
			break
		}

		var newestDateOnPage time.Time

		messages.Each(func(i int, s *goquery.Selection) {
			timeNode := s.Find("time.time").First()
			datetimeStr, exists := timeNode.Attr("datetime")
			if !exists {
				return
			}

			postDate, err := time.Parse(time.RFC3339, datetimeStr)
			if err != nil {
				return
			}

			if postDate.After(newestDateOnPage) {
				newestDateOnPage = postDate
			}
			if postDate.Before(cutoffDate) {
				return
			}

			s.Find("a").Each(func(i int, link *goquery.Selection) {
				href, exists := link.Attr("href")
				if exists {
					href = strings.TrimSpace(html.UnescapeString(href))
					if strings.HasPrefix(href, "tg://proxy?") || strings.HasPrefix(href, "https://t.me/proxy?") {
						normalizedLink := strings.Replace(href, "https://t.me/proxy?", "tg://proxy?", 1)
						proxies = append(proxies, normalizedLink)
					}
				}
			})
		})

		if !newestDateOnPage.IsZero() && newestDateOnPage.Before(cutoffDate) {
			break
		}

		firstMessage := messages.First()
		dataPost, exists := firstMessage.Attr("data-post")
		if !exists {
			break
		}

		parts := strings.Split(dataPost, "/")
		if len(parts) != 2 {
			break
		}

		currentURL = fmt.Sprintf("%s?before=%s", baseURL, parts[1])
		time.Sleep(1 * time.Second)
	}
	return proxies
}

// LoadTgChannelsFromFile читает файл, парсит имена Telegram-каналов и добавляет их в config.TgChannels
func LoadTgChannelsFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("не удалось открыть файл %q: %w", path, err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Printf("не удалось корректно закрыть файловый дескриптор для %s", path)
		}
	}(f)

	existing := make(map[string]struct{}, len(config.TgChannels))
	for _, ch := range config.TgChannels {
		existing[strings.ToLower(ch)] = struct{}{}
	}

	var (
		added   int
		skipped int
		total   int
	)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		if total < 1 && !utils.IsTextLine(line) {
			return fmt.Errorf("передан не текстовый файл '%s'", path)
		}

		total++

		name, warn := parseTgChannelName(line)

		if warn != "" {
			log.Printf("[WARN] %s", warn)
			skipped++
			continue
		}

		if name == "" {
			continue
		}

		key := strings.ToLower(name)
		if _, exists := existing[key]; exists {
			continue
		}

		existing[key] = struct{}{}
		config.TgChannels = append(config.TgChannels, name)
		added++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка чтения файла %q: %w", path, err)
	}

	if total == 0 {
		log.Printf("[WARN] файл %q пуст", path)
		return nil
	}

	if added == 0 {
		log.Printf("[WARN] файл %q не содержит ничего полезного (строк: %d, пропущено: %d)", path, total, skipped)
		return nil
	}

	log.Printf("[INFO] из файла %q добавлено каналов: %d (пропущено: %d)", path, added, skipped)
	return nil
}

// parseTgChannelName парсит строку и возвращает чистое имя канала
func parseTgChannelName(line string) (name string, warn string) {
	s := strings.TrimSpace(line)
	if s == "" {
		return "", ""
	}

	if strings.HasPrefix(s, "tg://") {
		return "", fmt.Sprintf("пропущена ссылка tg:// (id каналов не поддерживаются): %q", s)
	}

	// https://t.me/s/ChannelName или http://t.me/s/ChannelName
	for _, prefix := range []string{"https://t.me/s/", "http://t.me/s/"} {
		if strings.HasPrefix(s, prefix) {
			name = strings.TrimPrefix(s, prefix)
			name = cleanChannelName(name)
			return name, ""
		}
	}

	// https://t.me/ChannelName или http://t.me/ChannelName
	for _, prefix := range []string{"https://t.me/", "http://t.me/"} {
		if strings.HasPrefix(s, prefix) {
			name = strings.TrimPrefix(s, prefix)
			name = cleanChannelName(name)
			return name, ""
		}
	}

	// t.me/ChannelName
	if strings.HasPrefix(s, "t.me/") {
		name = strings.TrimPrefix(s, "t.me/")
		name = cleanChannelName(name)
		return name, ""
	}

	// @ChannelName
	if strings.HasPrefix(s, "@") {
		name = strings.TrimPrefix(s, "@")
		name = cleanChannelName(name)
		return name, ""
	}

	// ChannelName
	if isValidChannelName(s) {
		return s, ""
	}

	return "", ""
}

// cleanChannelName убирает возможные query-параметры, слэши и пробелы
func cleanChannelName(s string) string {
	if idx := strings.IndexAny(s, "/?#"); idx != -1 {
		s = s[:idx]
	}
	s = strings.TrimSpace(s)
	return s
}

// isValidChannelName проверяет, что строка похожа на имя Telegram-канала:
// только буквы, цифры и подчёркивания, длина 4–32 символа.
func isValidChannelName(s string) bool {
	if len(s) < 4 || len(s) > 32 {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_') {
			return false
		}
	}
	return true
}
