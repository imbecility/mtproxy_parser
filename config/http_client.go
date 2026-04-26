package config

import (
	"net/http"
	"time"
)

// HTTPClient общий клиент для сетевых запросов
var HTTPClient *http.Client

func init() {
	HTTPClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}
