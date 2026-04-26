package utils

import (
	"log"
	"net/http"
	"time"

	"github.com/imbecility/mtproxy_parser/config"
)

// connectivityClient клиент для быстрой проверки доступности веб-ресурса с коротким тайм-аутом
var connectivityClient = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:    1,
		IdleConnTimeout: 10 * time.Second,
	},
}

// IsWebAvailable проверяет доступность веб-версии Telegram
func IsWebAvailable(url string) bool {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", config.UserAgent)

	resp, err := connectivityClient.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()

	return resp.StatusCode < 500
}

// CheckTelegramWebAvailability проверяет доступность основного домена Telegram или его зеркала,
// устанавливает рабочий базовый URL в конфигурации и выводит уведомление в лог при обнаружении сетевых ограничений.
func CheckTelegramWebAvailability() {
	switch {
	case IsWebAvailable("https://t.me"):
		config.TelegramBaseUrl = "https://t.me"
		return
	case IsWebAvailable(config.TmeMirror):
		config.TelegramBaseUrl = config.TmeMirror
		log.Printf("  основной домен t.me не доступен из-за ограничений текущего интернет-подключения, используется зеркало!")
		return
	}
	log.Printf("  домен t.me блокируется на текущем интенрент-подключении, сбор в tg-каналах невозможен!")
}

// CheckMtproXyzAvailability проверяет доступность основного домена mtpro.xyz или его зеркала,
// устанавливает рабочий базовый URL в конфигурации и выводит уведомление в лог при обнаружении сетевых ограничений.
func CheckMtproXyzAvailability() {
	switch {
	case IsWebAvailable("https://mtpro.xyz"):
		config.MtproXyzBaseUrl = "https://mtpro.xyz"
		return
	case IsWebAvailable(config.MtproXyzMirror):
		config.MtproXyzBaseUrl = config.MtproXyzMirror
		log.Printf("  основной домен mtpro.xyz не доступен из-за ограничений текущего интернет-подключения, используется зеркало!")
		return
	}
	log.Printf("  домен mtpro.xyz блокируется на текущем интернет-подключении, сбор прокси невозможен!")
}

// CheckApiTelegramVpnOrgAvailability проверяет доступность основного домена api.telegramvpn.org или его зеркала,
// устанавливает рабочий базовый URL в конфигурации и выводит уведомление в лог при обнаружении сетевых ограничений.
func CheckApiTelegramVpnOrgAvailability() {
	switch {
	case IsWebAvailable("https://api.telegramvpn.org/proxies"):
		config.ApiTelegramVpnOrgBaseUrl = "https://api.telegramvpn.org"
		return
	case IsWebAvailable(config.ApiTelegramVpnOrgMirror + "/api.telegramvpn.org/proxies"):
		config.ApiTelegramVpnOrgBaseUrl = config.ApiTelegramVpnOrgMirror
		log.Printf("  основной домен api.telegramvpn.org не доступен из-за ограничений текущего интернет-подключения, используется зеркало!")
		return
	}
	log.Printf("  домен api.telegramvpn.org блокируется на текущем интернет-подключении, сбор прокси невозможен!")
}

// CheckMtprotoRuAvailability проверяет доступность основного домена mtproto.ru или его зеркала,
// устанавливает рабочий базовый URL в конфигурации и выводит уведомление в лог при обнаружении сетевых ограничений.
func CheckMtprotoRuAvailability() {
	switch {
	case IsWebAvailable("https://mtproto.ru"):
		config.MtprotoRuBaseUrl = "https://mtproto.ru"
		return
	case IsWebAvailable(config.MtprotoRuMirror):
		config.MtprotoRuBaseUrl = config.MtprotoRuMirror
		log.Printf("  основной домен mtproto.ru не доступен из-за ограничений текущего интернет-подключения, используется зеркало!")
		return
	}
	log.Printf("  домен mtproto.ru блокируется на текущем интернет-подключении, сбор прокси невозможен!")
}
