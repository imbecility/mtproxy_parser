package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/imbecility/mtproxy_parser/config"
	"github.com/imbecility/mtproxy_parser/providers"
	"github.com/imbecility/mtproxy_parser/types"
	"github.com/imbecility/mtproxy_parser/utils"

	"github.com/imbecility/colorgo/clih"
)

func main() {
	log.SetFlags(0)
	cliName := os.Args[0]
	validateExists := &utils.OptionalString{}
	customRawUrls := &utils.OptionalString{}
	customTgChannels := &utils.OptionalString{}

	workers := flag.Int("workers", config.MaxCheckers, "количество одновременных потоков для чекера прокси")
	flag.Var(customRawUrls, "raw-urls", "путь до текстового файла со списком ПРЯМЫХ ссылок на текстовые файлы с источниками прокси")
	flag.Var(customTgChannels, "tg-channels", "путь до текстового файла со списком Telegram-каналов, публикующих MTProxy")
	outputFile := flag.String("output", config.OutputFile, "путь к итоговому файлу (по умолчанию сохраняется рядом с программой)")
	flag.Var(validateExists, "validate", "выполнить валидацию файла со списком прокси, оставив в нем только рабочие, "+
		"и завершить программу без поиска новых, пример: "+filepath.Base(cliName)+" -validate ~/Desktop/my_tg_proxies.txt")
	saveDead := flag.Bool("save-dead", true, "сохранять ли нерабочие в данный момент прокси в отдельный файл")
	checkTimeout := flag.Duration("check-timeout", config.CheckTimeout, "таймаут проверки одного прокси (реальный MTProto-хендшейк занимает до 2 секунд, всё что дольше == заблокировано/мертво): 15s, 30s, 100s, ...")
	bindAddr := flag.String("bind-addr", "", "локальный IP-адрес сетевого интерфейса для подключений (например 192.168.1.122), игнорирует VPN, а чтобы узнать нужный адрес сетевой карты нужно выполнить: ifconfig (на любой современной OS) и найти нужную, либо посмотреть в настройках сетевого интерфейса")
	githubRepo := flag.String("github-repo", "", "репо GitHub для выгрузки (если пусто - не выгружаем)")
	githubToken := flag.String("github-token", "", "GitHub Token (если пусто, ищет в env GITHUB_TOKEN)")
	githubBranch := flag.String("github-branch", "main", "ветка GitHub")
	githubPath := flag.String("github-path", "", "путь сохранения в репо (по умолчанию имя файла)")
	githubSquash := flag.Bool("github-squash", false, "сделать SuperSquash (оставить только 1 коммит в истории)")

	flag.Usage = func() {
		cliHelp := clih.GetFlagDescriptions(clih.FlagDescriptionsConfig{
			CliName: &cliName,
			// Syntax update in Go 1.26+: replace pointer to local variable 'description' with new().
			Usage: new(fmt.Sprintf("%s [options] <command> [arguments]\n\n", cliName)),
			Description: new("парсер Telegram MTProxy с сайтов, tg-каналов, других списков подписок в единый файл с валидацией и сортировкой по пингу.\n\n" +
				"флаги -github-* не обязательны, и нужны для выгрузки результатов в репо.\n\n" +
				"флаги -raw-urls и -tg-channels не обязательны и служат для указания дополнительных/новых источников без необходимости перекомпиляции программы, пример:\n\n" +
				filepath.Base(cliName) + " -raw-urls ./gh_links.txt -tg-channels ./new_tg_channels.txt\n\n" +
				"в большинстве случаев достаточно просто выполнить программу без дополнительных флагов\n\n" +
				"исходный код и обновления: https://github.com/imbecility/mtproxy_parser"),
		})

		fmt.Println(cliHelp)
	}
	flag.Parse()
	utils.ResolveOptionalFlags(validateExists, "validate")
	utils.ResolveOptionalFlags(customRawUrls, "raw-urls")
	utils.ResolveOptionalFlags(customTgChannels, "tg-channels")

	aliveFilePath, err := utils.ResolvePath(*outputFile)
	if err != nil {
		log.Printf("  внимание, не удалось разрешить путь %q: %v", *outputFile, err)
		aliveFilePath = *outputFile
	}

	deadFilePath := utils.WithStemSuffix(aliveFilePath, "_dead")

	switch {
	case !customRawUrls.IsSet:

	case customRawUrls.IsSet && !customRawUrls.HasValue:
		log.Printf("указан флаг -raw-urls, но после него не передан путь до текстового файла со списком прямых ссылок! игнор...")

	case customRawUrls.IsSet && customRawUrls.HasValue:
		if !utils.IsFile(customRawUrls.Value) {
			log.Printf("указан флаг -raw-urls, но переданный путь не является файлом или не существует! игнор...")
		}

		config.CustomRawUrls = customRawUrls.Value
	}

	switch {
	case !customTgChannels.IsSet:

	case customTgChannels.IsSet && !customTgChannels.HasValue:
		log.Printf("указан флаг -tg-channels, но после него не передан путь до текстового файла со списком Telegram-каналов, публикующих MTProxy! игнор...")

	case customTgChannels.IsSet && customTgChannels.HasValue:
		if !utils.IsFile(customTgChannels.Value) {
			log.Printf("указан флаг -tg-channels, но переданный путь не является файлом или не существует! игнор...")
		}

		config.CustomTgChannels = customTgChannels.Value
	}

	// только валидация
	switch {
	case !validateExists.IsSet:

	case validateExists.IsSet && !validateExists.HasValue:
		log.Fatal("указан флаг -validate, но после него не передан путь до текстового файла со списком прокси!")

	case validateExists.IsSet && validateExists.HasValue:
		if !utils.IsFile(validateExists.Value) {
			log.Fatal("указан флаг -validate, но переданный путь '" + validateExists.Value + "' не является файлом или не существует!")
		}

		if err := utils.CheckFile(validateExists.Value); err != nil {
			log.Fatalf("не удалось проверить файл %q: %v", validateExists.Value, err)
		}

		proxies, err := utils.LoadFromFile(validateExists.Value)
		if err != nil {
			log.Fatalf("не удалось загрузить прокси из %q: %v", validateExists.Value, err)
		}

		validatePath, err := utils.ResolvePath(validateExists.Value)
		if err != nil {
			validatePath = validateExists.Value
		}
		validateDeadPath := utils.WithStemSuffix(validatePath, "_dead")
		results := utils.CheckAndFilterProxies(proxies, *workers, *bindAddr, *checkTimeout)
		if err := utils.SaveResults(results, validatePath, validateDeadPath, *saveDead); err != nil {
			log.Fatalf("%v", err)
		}
		log.Printf("валидация файла %q завершена.", validatePath)
		os.Exit(0)
	}

	// запуск парсера

	log.Println("──── старт сбора прокси со всех провайдеров ────")

	allUniqueProxies := make(map[string]struct{})
	var mu sync.Mutex

	addLinks := func(links []string) int {
		added := 0
		mu.Lock()
		defer mu.Unlock()
		for _, link := range links {
			if _, exists := allUniqueProxies[link]; !exists {
				allUniqueProxies[link] = struct{}{}
				added++
			}
		}
		return added
	}

	var regularProviders, backgroundProviders []types.Provider

	for _, p := range providers.Registry {
		if strings.Contains(p.Name, "Background") {
			backgroundProviders = append(backgroundProviders, p)
		} else {
			regularProviders = append(regularProviders, p)
		}
	}

	var bgWg sync.WaitGroup

	// фоновые провайдеры
	for _, p := range backgroundProviders {
		bgWg.Add(1)
		go func(prov types.Provider) {
			defer bgWg.Done()
			links := prov.Fetch()
			added := addLinks(links)
			log.Printf("──── фоновый провайдер %s завершен, добавлено: %d ────\n", prov.Name, added)
		}(p)
	}

	// последовательные провайдеры
	for i, provider := range regularProviders {
		log.Printf("\n──── [%d/%d] запуск провайдера: %s ────\n", i+1, len(regularProviders), provider.Name)
		links := provider.Fetch()
		added := addLinks(links)
		log.Printf("──── провайдер %s завершен, добавлено новых: %d ────\n", provider.Name, added)
	}

	log.Println("\n[система] основные провайдеры отработали, отправлен сигнал остановки фоновым...")
	config.BgCancel()
	bgWg.Wait()

	newScrapedCount := len(allUniqueProxies)
	log.Printf("\n[система] парсинг завершен: собрано %d уникальных прокси из сети.\n", newScrapedCount)

	if utils.IsFile(aliveFilePath) {
		log.Printf("[система] инкрементальное слияние с прошлыми результатами из %s...\n", aliveFilePath)
	}
	oldAliveProxies, err := utils.LoadFromFile(aliveFilePath)
	if err != nil {
		log.Printf("[система] ошибка чтения старого файла: %v\n", err)
	} else if len(oldAliveProxies) > 0 {
		addedOld := addLinks(oldAliveProxies)
		log.Printf("[система] успешно загружено %d прокси из файла: %d уникальных (не найденных при текущем парсинге).\n", len(oldAliveProxies), addedOld)
	}
	if utils.IsFile(deadFilePath) {
		oldDeadProxies, err := utils.LoadFromFile(deadFilePath)
		if err != nil {
			log.Printf("[система] ошибка чтения файла мёртвых прокси: %v\n", err)
		} else if len(oldDeadProxies) > 0 {
			addedOldDead := addLinks(oldDeadProxies)
			log.Printf("[система] успешно загружено %d мёртвых прокси из прошлого запуска для повторной проверки: %d уникальных.\n", len(oldDeadProxies), addedOldDead)
		}
	}

	// нормализация и финальная дедупликация перед чекером
	normalizedUnique := make(map[string]struct{}, len(allUniqueProxies))
	for p := range allUniqueProxies {
		normalized, err := utils.NormalizeProxyURL(p)
		if err != nil {
			continue
		}
		normalizedUnique[normalized] = struct{}{}
	}

	var rawProxiesList []string
	for p := range normalizedUnique {
		rawProxiesList = append(rawProxiesList, p)
	}

	log.Printf("[система] после нормализации: %d уникальных прокси (было %d)\n", len(rawProxiesList), len(allUniqueProxies))

	// чекер
	log.Printf("\n──── запуск валидации (всего на проверку: %d, потоков: %d) ────\n", len(rawProxiesList), *workers)

	results := utils.CheckAndFilterProxies(rawProxiesList, *workers, *bindAddr, *checkTimeout)

	// сохранение результата в файл
	if err := utils.SaveResults(results, aliveFilePath, deadFilePath, *saveDead); err != nil {
		log.Fatalf("%v", err)
	}

	// выгрузка на GitHub
	if *githubRepo != "" {
		log.Printf("\n──── выгрузка в репо ────\n")
		log.Printf("[GitHub] выгрузка файла %s в репозиторий %s ...\n", aliveFilePath, *githubRepo)

		cfg := utils.UploadConfig{
			Token:      *githubToken,
			RepoUrl:    *githubRepo,
			Branch:     *githubBranch,
			FilePath:   aliveFilePath,
			RemotePath: *githubPath,
			Squash:     *githubSquash,
		}

		if err := utils.Upload(cfg); err != nil {
			log.Printf("[GitHub] ❌ ошибка выгрузки в репозиторий: %v\n", err)
		} else {
			log.Println("[GitHub] ✅ файл успешно выгружен в репозиторий!")
		}
	} else {
		log.Println("[GitHub] пропуск выгрузки (не задан флаг -github-repo).")
		log.Printf("\n──── готово ────\n")
	}
}
