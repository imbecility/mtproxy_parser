package utils

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// NormalizeProxyURL нормализует ссылку tg://proxy?, убирая мусорные параметры и приводя к единому виду,
// возвращает ошибку если ссылка не содержит обязательных полей server/port/secret.
func NormalizeProxyURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("невалидный URL: %w", err)
	}

	q := u.Query()

	server := strings.ToLower(strings.TrimSpace(q.Get("server")))
	port := strings.TrimSpace(q.Get("port"))
	secret := strings.ToLower(strings.TrimSpace(q.Get("secret")))

	if server == "" || port == "" || secret == "" {
		return "", errors.New("отсутствуют обязательные поля (server/port/secret)")
	}

	return fmt.Sprintf("tg://proxy?server=%s&port=%s&secret=%s", server, port, secret), nil
}
