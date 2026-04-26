package utils

import (
	"log"
	"os"
	"path/filepath"
)

// ResolvePath преобразует путь в абсолютный вид с использованием разделителей в стиле POSIX,
// раскрывая домашнюю директорию пользователя и разрешая символические ссылки.
func ResolvePath(path string) (string, error) {
	if len(path) >= 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[2:])
	}

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolved, err = filepath.Abs(path)
		if err != nil {
			return "", err
		}
	}

	return filepath.ToSlash(resolved), nil
}

// WithStemSuffix добавляет указанный суффикс к базовому имени файла перед его расширением,
// сохраняя исходный путь к директории и расширение файла.
func WithStemSuffix(path, suffix string) string {
	resolved, err := ResolvePath(path)
	if err != nil {
		log.Printf("  внимание, не удалось разрешить путь %q: %v", path, err)
		resolved = path
	}
	dir := filepath.Dir(resolved)
	ext := filepath.Ext(resolved)
	base := filepath.Base(resolved)
	stem := base[:len(base)-len(ext)]

	newName := stem + suffix + ext
	return filepath.Join(dir, newName)
}
