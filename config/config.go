package config

import (
	"context"
	"time"
)

var (
	// TgChannels содержит список tg-каналов где постят прокси
	TgChannels = []string{
		"ProxyMTProto",
		"TProxyRU",
		"TelMTProto",
		"mtprotoF",
		"mtproto6",
		"mtp4tg",
		"memtproxy",
		"MTProxy4free",
	}

	// CustomRawUrls путь до файла с прямыми ссылками на списки прокси
	CustomRawUrls = ""

	// CustomTgChannels путь до файла со списком TG-каналов где публикуют MTPROTO-прокси
	CustomTgChannels = ""

	// BgCtx, BgCancel устанавливают глобальный контекст для управления фоновыми задачами
	BgCtx, BgCancel = context.WithCancel(context.Background())

	// TimeZone таймзона для отображения последнего обновления файла с прокси в репо
	TimeZone = time.FixedZone("MSK", 3*60*60)
)

const (
	// OutputFile имя файла с результатами
	OutputFile = "tg_proxies.txt"

	// DaysToParse устанавливает глубину сканирования tg-каналов
	DaysToParse = 14

	// MaxCheckers устанавливает во сколько одновременных потоков выполнять проверку проксей.
	// больше 200 ставить не стоит, чтобы не словить таймауты от ОС,
	// если соединение нестабильное и слабое - лучше понизить до 10
	MaxCheckers = 100

	// таймауты чекера
	// реальный MTProto-хендшейк занимает 200–2000ms, всё что дольше практически гарантированно мертво

	// CheckTimeout общий лимит на проверку одного прокси
	CheckTimeout = 10 * time.Second
	// DialTimeout таймаут TCP-соединения
	DialTimeout = 5 * time.Second
	// ExchangeTimeout таймаут на MTProto хендшейк (ключи, авторизация)
	ExchangeTimeout = 7 * time.Second

	// UserAgent для всех сетевых запросов парсера
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36"
)
