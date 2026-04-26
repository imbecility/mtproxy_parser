package utils

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

// IsTextLine проверяет, состоит ли строка из печатных символов и стандартных управляющих кодов:
// возвращает false, если в строке обнаружены непечатные управляющие символы.
func IsTextLine(s string) bool {
	for _, b := range []byte(s) {
		if b < 9 || (b > 13 && b < 32) {
			return false
		}
	}
	return true
}

// CheckFile анализирует файл по указанному пути на соответствие формату списка Telegram-прокси
func CheckFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Printf("не удалось корректно закрыть файл '%s': %v", path, err)
		}
	}(f)

	scanner := bufio.NewScanner(f)

	foundLines := 0
	for scanner.Scan() {
		line := scanner.Text()

		if !IsTextLine(line) {
			return fmt.Errorf("передан не текстовый файл '%s'", path)
		}

		if strings.TrimSpace(line) == "" {
			continue
		}

		foundLines++

		if foundLines == 2 {
			if strings.Contains(line, "tg://proxy") || strings.Contains(line, "https://t.me/proxy") {
				return nil
			}

			return fmt.Errorf("передан текстовый файл '%s' который не содержит ссылок tg-прокси", path)
		}
	}

	return scanner.Err()
}
