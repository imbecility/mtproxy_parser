package utils

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/imbecility/mtproxy_parser/types"
)

// LoadFromFile читает файл построчно и возвращает список прокси,
// если файла нет - возвращает пустой список без ошибки.
func LoadFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // первый запуск
		}
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("ошибка при закрытии файла %s: %v\n", filename, err)
		}
	}(file)

	var proxies []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			proxies = append(proxies, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return proxies, nil
}

// saveToFile сохраняет срез строк в файл с сохранением порядка
func saveToFile(proxies []string, filename string) error {
	if len(proxies) < 1 {
		return fmt.Errorf("список прокси пуст, нечего сохранять в файл %s", filename)
	}
	// создание всех промежуточных директорий если их нет
	if dir := filepath.Dir(filename); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("не удалось создать директорию %q: %w", dir, err)
		}
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("ошибка при закрытии файла %s: %v\n", filename, err)
		}
	}()

	for _, proxy := range proxies {
		if _, err := file.WriteString(proxy + "\n"); err != nil {
			return err
		}
	}
	return nil
}

// SaveResults выполняет итоговое сохранение проверенных прокси в соответствующие файлы,
// логирует статистику обработки и опционально записывает список неактивных адресов в зависимости от флага saveDead.
func SaveResults(results types.CheckResults, alivePath, deadPath string, saveDead bool) error {
	if err := saveToFile(results.Alive, alivePath); err != nil {
		return fmt.Errorf("ошибка сохранения живых прокси в %s: %v\n", alivePath, err)
	}
	log.Printf("обработка завершена, всего рабочих прокси: %d. файл: %s\n", len(results.Alive), alivePath)
	if saveDead && len(results.Dead) > 0 {
		if err := saveToFile(results.Dead, deadPath); err != nil {
			log.Printf("[система] ошибка сохранения мёртвых прокси в %s: %v\n", deadPath, err)
		} else {
			log.Printf("[система] мёртвые прокси (%d) сохранены в: %s\n", len(results.Dead), deadPath)
		}
	}
	return nil
}
