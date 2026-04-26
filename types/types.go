package types

import (
	"errors"
	"time"
)

// CheckResults содержит результаты проверки прокси, разделённые на живые и мёртвые.
type CheckResults struct {
	Alive []string
	Dead  []string
}

// ProxyData структура для парсинга JSON из раскодированных JS скриптов
type ProxyData struct {
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Secret string `json:"secret"`
}

// Provider структура, описывающая провайдера данных
type Provider struct {
	Name  string
	Fetch func() []string
}

// VpnAPIResponse структуры для парсинга JSON api.telegramvpn.org
type VpnAPIResponse struct {
	Pages   int        `json:"pages"`
	Proxies []vpnProxy `json:"proxies"`
}

type vpnProxy struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// CheckResult содержит результат проверки прокси.
type CheckResult struct {
	URL  string
	Ping time.Duration
}

// Result содержит результат проверки tg-прокси.
type Result struct {
	Ping time.Duration
}

// ErrProxyDead возвращается когда прокси недоступен или заблокирован.
var ErrProxyDead = errors.New("proxy is dead")
