package utils

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/dcs"
	"github.com/gotd/td/tg"
	"github.com/imbecility/mtproxy_parser/types"
)

// CheckAndFilterProxies многопоточно проверяет список прокси, отбрасывает мертвые и сортирует по пингу.
func CheckAndFilterProxies(proxies []string, maxWorkers int, localAddr string, timeout time.Duration) types.CheckResults {
	log.Printf("[Checker] проверка %d прокси в %d потоков...\n", len(proxies), maxWorkers)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var alive []types.CheckResult
	var dead []string
	var checked atomic.Int64

	sem := make(chan struct{}, maxWorkers)
	total := len(proxies)

	for _, p := range proxies {
		wg.Add(1)
		go func(rawURL string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res, err := CheckProxy(rawURL, localAddr, timeout)

			mu.Lock()
			if err == nil {
				alive = append(alive, types.CheckResult{URL: rawURL, Ping: res.Ping})
			} else {
				dead = append(dead, rawURL)
			}
			mu.Unlock()

			current := checked.Add(1)
			fmt.Printf("\r[Checker] проверено %d/%d", current, total)
		}(p)
	}

	wg.Wait()
	fmt.Println()

	sort.Slice(alive, func(i, j int) bool {
		return alive[i].Ping < alive[j].Ping
	})

	log.Printf("[Checker] проверка завершена: живых - %d, мертвых - %d\n", len(alive), len(dead))

	var aliveURLs []string
	for _, r := range alive {
		aliveURLs = append(aliveURLs, r.URL)
	}
	return types.CheckResults{
		Alive: aliveURLs,
		Dead:  dead,
	}
}

// CheckProxy проверяет один MTProto-прокси через gotd, выполняя реальный MTProto-хендшейк
// и запрос HelpGetNearestDC для подтверждения работоспособности.
func CheckProxy(rawURL string, localAddr string, timeout time.Duration) (types.Result, error) {
	server, port, secret, err := parseProxyURL(rawURL)
	if err != nil {
		return types.Result{}, fmt.Errorf("parse error: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", server, port)

	resolver, err := dcs.MTProxy(addr, secret, dcs.MTProxyOptions{Dial: buildDialer(localAddr)})
	if err != nil {
		return types.Result{}, fmt.Errorf("invalid proxy config: %w", err)
	}

	client := telegram.NewClient(telegram.TestAppID, telegram.TestAppHash, telegram.Options{
		Resolver:        resolver,
		SessionStorage:  &session.StorageMemory{},
		NoUpdates:       true,
		DialTimeout:     timeout / 2,     // 50% на TCP
		ExchangeTimeout: timeout * 3 / 4, // 75% на хендшейк
	})

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()
	var latency time.Duration
	var runErr error

	// проверка в дочерней горутине
	done := make(chan struct{})
	go func() {
		defer close(done)
		runErr = client.Run(ctx, func(runCtx context.Context) error {
			_, err := tg.NewClient(client).HelpGetNearestDC(runCtx)
			latency = time.Since(start)
			return err
		})
	}()

	// ожидание реального завершения, либо по таймеру
	select {
	case <-done:
		// результат до истечения `timeout`
		if runErr != nil {
			return types.Result{}, types.ErrProxyDead
		}
		return types.Result{Ping: latency}, nil
	case <-ctx.Done():
		// таймаут, но gotd почему-то завис, придется просто бросить горутину на произвол OS
		return types.Result{}, types.ErrProxyDead
	}
}

// parseProxyURL парсит tg://proxy?... или https://t.me/proxy?...
// возвращает server, port, secret в байтах.
func parseProxyURL(raw string) (server string, port int, secret []byte, err error) {
	var queryStr string

	switch {
	case strings.HasPrefix(raw, "tg://proxy?"):
		queryStr = strings.TrimPrefix(raw, "tg://proxy?")
	case strings.HasPrefix(raw, "https://t.me/proxy?"):
		queryStr = strings.TrimPrefix(raw, "https://t.me/proxy?")
	case strings.HasPrefix(raw, "http://t.me/proxy?"):
		queryStr = strings.TrimPrefix(raw, "http://t.me/proxy?")
	default:
		return "", 0, nil, fmt.Errorf("unsupported URL format")
	}

	params, err := url.ParseQuery(queryStr)
	if err != nil {
		return "", 0, nil, fmt.Errorf("malformed query string: %w", err)
	}

	server = params.Get("server")
	if server == "" {
		return "", 0, nil, fmt.Errorf("missing parameter: server")
	}

	portStr := params.Get("port")
	if portStr == "" {
		return "", 0, nil, fmt.Errorf("missing parameter: port")
	}
	if _, scanErr := fmt.Sscanf(portStr, "%d", &port); scanErr != nil {
		return "", 0, nil, fmt.Errorf("invalid port %q: %w", portStr, scanErr)
	}
	if port < 1 || port > 65535 {
		return "", 0, nil, fmt.Errorf("port %d out of range [1, 65535]", port)
	}

	secretStr := params.Get("secret")
	if secretStr == "" {
		return "", 0, nil, fmt.Errorf("missing parameter: secret")
	}

	secret, err = decodeSecret(secretStr)
	if err != nil {
		return "", 0, nil, err
	}

	return server, port, secret, nil
}

// decodeSecret декодирует секрет из hex или base64url в байты.
func decodeSecret(s string) ([]byte, error) {
	isHex := len(s) > 0
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			isHex = false
			break
		}
	}

	if isHex {
		if len(s)%2 != 0 {
			return nil, fmt.Errorf("invalid secret: hex string has odd length")
		}
		b, err := hex.DecodeString(s)
		if err != nil {
			return nil, fmt.Errorf("invalid secret: %w", err)
		}
		return b, nil
	}

	normalized := strings.NewReplacer("-", "+", "_", "/").Replace(s)
	switch len(normalized) % 4 {
	case 2:
		normalized += "=="
	case 3:
		normalized += "="
	case 1:
		return nil, fmt.Errorf("invalid secret: invalid base64 length")
	}

	b, err := base64.StdEncoding.DecodeString(normalized)
	if err != nil {
		return nil, fmt.Errorf("invalid secret: cannot decode as hex or base64: %w", err)
	}
	return b, nil
}

// buildDialer возвращает dial-функцию привязанную к указанному локальному адресу.
// Если localAddr пуст — возвращает nil и gotd использует системный диалер по умолчанию.
func buildDialer(localAddr string) func(ctx context.Context, network, addr string) (net.Conn, error) {
	if localAddr == "" {
		return nil
	}
	dialer := &net.Dialer{
		LocalAddr: &net.TCPAddr{
			IP: net.ParseIP(localAddr),
		},
	}
	return dialer.DialContext
}
