package providers

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/imbecility/mtproxy_parser/config"
	"github.com/imbecility/mtproxy_parser/utils"
)

func init() {
	Register("GitHubRaw", fetchGitHubRaw)
}

// fetchGitHubRaw выполняет параллельные HTTP-запросы с ограничением конкурентности,
// извлекает прокси-ссылки из удаленных источников, указанных в конфигурации, парсит содержимое
// ответов и возвращает список уникальных прокси, нормализованных к формату tg://proxy?.
func fetchGitHubRaw() []string {
	var urls []string

	if config.CustomRawUrls != "" && utils.IsFile(config.CustomRawUrls) {
		rawUrls, err := filterURLsFromFile(config.CustomRawUrls)
		if err != nil {
			log.Printf(" [GitHubRaw] не удалось добавить кастомные ссылки из файла '%s': %v", config.CustomRawUrls, err)
		}
		if rawUrls != "" {
			for _, line := range strings.Split(rawUrls, "\n") {
				urls = append(urls, line)
			}
		}
	}

	rawLines := strings.Split(config.RawLinks, "\n")

	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if line != "" && !slices.Contains(urls, line) {
			urls = append(urls, line)
		}
	}

	log.Printf(" [GitHubRaw] найдено ссылок для парсинга: %d\n", len(urls))

	var wg sync.WaitGroup
	var mu sync.Mutex
	uniqueProxies := make(map[string]struct{})

	// семафор для ограничения параллельных горутин
	sem := make(chan struct{}, 10)

	for _, u := range urls {
		wg.Add(1)
		go func(targetURL string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			req, err := http.NewRequest("GET", targetURL, nil)
			if err != nil {
				return
			}
			req.Header.Set("User-Agent", config.UserAgent)

			resp, err := config.HTTPClient.Do(req)
			if err != nil || resp.StatusCode != 200 {
				if resp != nil {
					_ = resp.Body.Close()
				}
				return
			}

			body, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				return
			}

			lines := strings.Split(string(body), "\n")

			mu.Lock()
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "https://t.me/proxy?") {
					line = strings.Replace(line, "https://t.me/proxy?", "tg://proxy?", 1)
				}
				if strings.HasPrefix(line, "tg://proxy?") {
					uniqueProxies[line] = struct{}{}
				}
			}
			mu.Unlock()
		}(u)
	}

	wg.Wait()

	var results []string
	for p := range uniqueProxies {
		results = append(results, p)
	}

	return results
}

// filterURLsFromFile читает файл, фильтрует валидные URL и возвращает их в виде многострочной строки,
// возвращает ошибку если файл не содержит ни одной валидной ссылки.
func filterURLsFromFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("не удалось корректно закрыть файл %s", filePath)
		}
	}(file)

	var validURLs []string
	var total int
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if total < 1 && !utils.IsTextLine(line) {
			return "", fmt.Errorf("передан не текстовый файл '%s'", filePath)
		}

		total++

		if line == "" {
			continue
		}

		if isValidURL(line) {
			validURLs = append(validURLs, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	if len(validURLs) == 0 {
		return "", fmt.Errorf("файл %q не содержит валидных ссылок", filePath)
	}

	return strings.Join(validURLs, "\n"), nil
}

// isValidURL проверяет, является ли строка валидным URL.
func isValidURL(s string) bool {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return false
	}

	if u.Scheme == "" {
		return false
	}

	if u.Host == "" {
		return false
	}

	return true
}
