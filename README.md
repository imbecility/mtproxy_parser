# mtproxy_parser

Парсер Telegram MTProxy — автоматически собирает прокси из десятков источников, проверяет их доступность и сохраняет рабочие адреса в единый файл, отсортированные по пингу.

## Возможности

- Сбор прокси из нескольких независимых источников параллельно:
  - публичные Telegram-каналы (через веб-превью)
  - GitHub-репозитории со списками прокси (200+ источников из коробки)
  - сайты в интернете
- Нормализация и дедупликация перед проверкой
- Многопоточная проверка доступности
- Сортировка рабочих прокси по пингу
- Инкрементальное слияние с результатами предыдущего запуска
- Повторная проверка прокси из прошлого «мёртвого» списка
- Опциональная выгрузка результата в GitHub-репозиторий
- Поддержка кастомных источников без перекомпиляции (через файлы)
- Автоматическое переключение на зеркало если основной домен t.me недоступен (заблокирован) при парсинге каналов

## Как работает валидация прокси

Каждый прокси проверяется TCP-подключением, создаёт полноценное TCP-соединение и выполняет MTProto-хендшейк + API-запрос.

По умолчанию используется 100 потоков, при нестабильном соединении рекомендуется снизить через `-workers 10`.

## Быстрый старт

### Скачать готовый бинарник

На странице [релизов](https://github.com/imbecility/mtproxy_parser/releases/latest) можно загрузить архив для нужной платформы:

| скачать                                                                                                                      |
|------------------------------------------------------------------------------------------------------------------------------|
| [Linux x64](https://github.com/imbecility/mtproxy_parser/releases/latest/download/mtproxy_parser-linux-x64.tar.gz)           |
| [Linux ARM  ](https://github.com/imbecility/mtproxy_parser/releases/latest/download/mtproxy_parser-linux-arm.tar.gz)         |
| [macOS Apple Silicon](https://github.com/imbecility/mtproxy_parser/releases/latest/download/mtproxy_parser-macos-arm.tar.gz) |
| [macOS Intel ](https://github.com/imbecility/mtproxy_parser/releases/latest/download/mtproxy_parser-macos-intel.tar.gz)      |
| [Windows x64 ](https://github.com/imbecility/mtproxy_parser/releases/latest/download/mtproxy_parser-windows-x64.zip)         |
| [Windows ARM](https://github.com/imbecility/mtproxy_parser/releases/latest/download/mtproxy_parser-windows-arm.zip)          |

### Собрать из исходников

<details>
  <summary>👈 развернуть и показать инструкции по сборке</summary>

Установить Go: https://go.dev/dl
Затем:

```bash
git clone --depth 1 https://github.com/imbecility/mtproxy_parser.git
cd mtproxy_parser
go mod tidy
go build -ldflags="-s -w -extldflags '-static'" -trimpath .
```

Или:

### Linux / macOS

```bash
chmod +x build_on_linux.sh
./build_on_linux.sh
```

### Windows (только в PowerShell)

```powershell
.\build_on_windows.ps1
```

Готовые бинарники появятся в папке `build/`, архивы для публикации — в `dist/`.

Поддерживаемые платформы: `linux/x64`, `linux/arm`, `macos/intel`, `macos/apple_silicon`, `windows/x64`, `windows/arm64`.

</details>

---

### Запуск

```bash
# простой запуск — соберёт всё из встроенных источников и сохранит в tg_proxies.txt
./mtproxy_parser

# указать свой выходной файл
./mtproxy_parser -output ~/proxies/result.txt

# с выгрузкой в GitHub
./mtproxy_parser -github-repo owner/repo -github-token ghp_xxxxxx

# только проверить существующий файл, не собирать новые прокси
./mtproxy_parser -validate ~/Desktop/my_proxies.txt
```

## Флаги

| Флаг             | По умолчанию     | Описание                                                                                |
|------------------|------------------|-----------------------------------------------------------------------------------------|
| `-output`        | `tg_proxies.txt` | Путь к итоговому файлу с рабочими прокси                                                |
| `-workers`       | `100`            | Количество потоков для одновременной проверки прокси                                    |
| `-validate`      | —                | Проверить существующий файл и оставить только рабочие прокси, затем завершить программу |
| `-save-dead`     | `true`           | Сохранять ли нерабочие прокси в отдельный файл (`*_dead.txt`)                           |
| `-bind-addr`     | —                | Локальный IP-адрес сетевого интерфейса для запроса напрямую через него, минуя VPN       |
| `-check-timeout` | `10s`            | Тайм-аут проверки прокси в секундах (15s, 30s, 100s, ...)                               |
| `-raw-urls`      | —                | Путь к текстовому файлу со списком прямых ссылок на дополнительные источники прокси     |
| `-tg-channels`   | —                | Путь к текстовому файлу со списком дополнительных Telegram-каналов                      |
| `-github-repo`   | —                | Репозиторий GitHub для выгрузки результата (`owner/repo`)                               |
| `-github-token`  | —                | GitHub Personal Access Token (или переменная окружения `GITHUB_TOKEN`)                  |
| `-github-branch` | `main`           | Ветка репозитория для выгрузки                                                          |
| `-github-path`   | —                | Путь к файлу в репозитории (по умолчанию — имя выходного файла)                         |
| `-github-squash` | `false`          | Оставить только один коммит в истории ветки (SuperSquash)                               |


<details>
  <summary>👈 развернуть и показать справочный вывод программы</summary>

```text
> ./mtproxy_parser -h                                                                      

----------------------------------------------------------------------------
использование:
mtproxy_parser.exe [options] <command> [arguments]                                                                                                        

парсер Telegram MTProxy с сайтов, tg-каналов, других списков подписок в единый
файл с валидацией и сортировкой по пингу.

флаги -github-* не обязательны, и нужны для выгрузки результатов в репо.

флаги -raw-urls и -tg-channels не обязательны и служат для указания
дополнительных/новых источников без необходимости перекомпиляции программы,
пример:

mtproxy_parser.exe -raw-urls ./gh_links.txt -tg-channels ./new_tg_channels.txt

в большинстве случаев достаточно просто выполнить программу без дополнительных
флагов

исходный код и обновления: https://github.com/imbecility/mtproxy_parser

----------------------------------------------------------------------------

флаги:                                                                                                                                                                                                                                              
  -bind-addr [string: (по умолчанию '')]                                                                                                                                                                                                            
        локальный IP-адрес сетевого интерфейса для подключений (например
        192.168.1.122), игнорирует VPN, а чтобы узнать нужный адрес сетевой
        карты нужно выполнить: ifconfig (на любой современной OS) и найти
        нужную, либо посмотреть в настройках сетевого интерфейса

  -check-timeout [duration: (по умолчанию '10s')]
        таймаут проверки одного прокси (реальный MTProto-хендшейк занимает до 2
        секунд, всё что дольше == заблокировано/мертво): 15s, 30s, 100s, ...

  -github-branch [string: (по умолчанию 'main')]
        ветка GitHub

  -github-path [string: (по умолчанию '')]
        путь сохранения в репо (по умолчанию имя файла)

  -github-repo [string: (по умолчанию '')]
        репо GitHub для выгрузки (если пусто - не выгружаем)

  -github-squash [bool: (по умолчанию 'false')]
        сделать SuperSquash (оставить только 1 коммит в истории)

  -github-token [string: (по умолчанию '')]
        GitHub Token (если пусто, ищет в env GITHUB_TOKEN)

  -output [string: (по умолчанию 'tg_proxies.txt')]
        путь к итоговому файлу (по умолчанию сохраняется рядом с программой)

  -raw-urls [string: (по умолчанию '')]
        путь до текстового файла со списком ПРЯМЫХ ссылок на текстовые файлы с
        источниками прокси

  -save-dead [bool: (по умолчанию 'true')]
        сохранять ли нерабочие в данный момент прокси в отдельный файл)

  -tg-channels [string: (по умолчанию '')]
        путь до текстового файла со списком Telegram-каналов, публикующих
        MTProxy

  -validate [string: (по умолчанию '')]
        выполнить валидацию файла со списком прокси, оставив в нем только
        рабочие, и завершить программу без поиска новых, пример:
        mtproxy_parser.exe -validate ~/Desktop/my_tg_proxies.txt

  -workers [int: (по умолчанию '100')]
        количество одновременных потоков для чекера прокси
```

</details>

## Дополнительные источники

Можно подключать новые источники без перекомпиляции через текстовые файлы.

### Прямые ссылки на списки прокси (`-raw-urls`)

Файл содержит URL на текстовые файлы, в которых построчно записаны прокси:

```
https://example.com/my_proxy_list.txt
https://raw.githubusercontent.com/someone/repo/main/proxies.txt
```

### Дополнительные Telegram-каналы (`-tg-channels`)

Поддерживаются любые форматы записи:

```
ProxyMTProto
@NewProxyChannel
https://t.me/anotherChannel
t.me/yetAnotherOne
```

Пример:

```bash
./mtproxy_parser -raw-urls ./my_sources.txt -tg-channels ./my_channels.txt
```

## Формат выходного файла

Файл содержит список рабочих прокси, по одной ссылке на строку, отсортированных по возрастанию пинга:

```
tg://proxy?server=1.2.3.4&port=443&secret=ee...
tg://proxy?server=5.6.7.8&port=8888&secret=dd...
```

Ссылки в формате `tg://proxy?` можно напрямую открыть в Telegram или вставить вручную.

При наличии флага `-save-dead` рядом с основным файлом создаётся файл `*_dead.txt` с нерабочими адресами — они будут повторно проверены при следующем запуске.

---

## Вариант установки периодической задачи в Windows

1. Открыть приложение Terminal и выбрать профиль "PowerShell" (устаревший CMD не годится)
2. Вставьте команду установки и нажмите Enter:
   ```powershell
   irm bun.sh/install.ps1 | iex
   ```
3. Дождаться завершения установки
4. Закрыть окно и открыть новый PowerShell, чтобы система увидела Bun.
5. Установить PM2:
   ```powershell
   bun install -g pm2
   ```
6. Создать папку любом удобном месте, например `D:\mtproxy`, распаковать в нее [mtproxy_parser.exe](https://github.com/imbecility/mtproxy_parser/releases/latest/download/mtproxy_parser-windows-x64.zip), создать в ней файл `task.json`, открыть блокнотом и вставить:
   ```json
   {
     "apps":[
       {
         "name": "parse-mtproxy",
         "script": "D:/mtproxy/mtproxy_parser.exe",
         "args": "-output \"D:/mtproxy/tg_proxies.txt\" -save-dead -workers 100 -bind-addr 192.168.1.122",
         "autorestart": false,
         "cron_restart": "0 */6 * * *"
       }
     ]
   }
   ```
7. Отредактировать пример в файле `task.json`:
    - Заменить пути к программе и файлу с результатами на свои реальные
    - IP 192.168.1.122 заменить на локальный адрес своего ПК чтобы чекер работал напрямую, если установлен VPN, или удалить вместе с флагом -bind-addr
    - Добавить по желанию другие флаги и аргументы (например для выгрузки на GitHub)
8. Перед сохранением проверить, что:
    - `script` — путь к `mtproxy_parser.exe` указан как абсолютный.
    - слеши в путях для Windows удвоены `\\` или использованы обычные `/`
    - `args` — аргументы запуска соответствуют справке по флагу `-h` или документации на данной странице.
    - `autorestart`: **false** — установлен запрет перезапускать программу сразу после того, как она завершила работу (чтобы она не крутилась бесконечно).
    - кавычки в путях экранированы: `\"`
    - `cron_restart` — установлен таймер, например: `"0 */6 * * *"` — задача будет автоматически запускаться каждые 6 часов (в 00:00, 06:00, 12:00 и 18:00 по часам компьютера).
9. Запуск задачи в PowerShell:
   ```powershell
   pm2 start "D:/mtproxy/task.json"
   ```
   Программа сразу запустится (отработает один раз), после чего уснет и будет ждать своего часа по таймеру.
10. Чтобы PM2 запомнил эту задачу на будущее, **обязательно ввести и выполнить:**
   ```powershell
   pm2 save
   ```
11. Кликнуть правой кнопкой мыши по иконке приложения терминала в меню "Пуск" и выбрать запуск от имени Администратора, вставить и выполнить:
   ```powershell
   $action = New-ScheduledTaskAction -Execute "bunx" -Argument "pm2 resurrect"; $trigger = New-ScheduledTaskTrigger -AtStartup; $settings = New-ScheduledTaskSettingsSet -RunOnlyIfNetworkAvailable $false; Register-ScheduledTask -TaskName "PM2_Startup" -Action $action -Trigger $trigger -Settings $settings -RunLevel Highest -User "SYSTEM" -Force
   ```
   Эта команда будет запускать `bunx pm2 resurrect` автоматически при старте ПК, а PM2 сам будет поднимать задачи добавленные в него.

12. Как управлять задачей и смотреть логи (команды выполняются в приложении Terminal):
    - логи лежат в: `%USERPROFILE%\.pm2\logs` (вставить прям так в **адресную строку** проводника и нажать Enter, либо в окно после нажатия `Win+R`)
    - `pm2 ls` — посмотреть список всех запущенных задач и их статус.
    - `pm2 logs parse-mtproxy` — смотреть логи задачи в реальном времени.
    - `pm2 logs parse-mtproxy --lines 500` — вывести последние 500 строк логов.
    - `pm2 restart parse-mtproxy` — **запустить задачу прямо сейчас вне очереди** (таймер при этом не собьется, следующий запуск будет по расписанию).
    - `pm2 delete parse-mtproxy` — полностью удалить задачу.


## Автоматическое обновление через GitHub Actions

Пример рабочего процесса для ежедневного обновления списка прокси прямо в своем репозитории:

```yaml
name: Update proxies

on:
  schedule:
    - cron: "0 */6 * * *"   # каждые 6 часов
  workflow_dispatch:

permissions:
  contents: write

jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      - name: Download latest release
        run: |
          curl -sL https://github.com/imbecility/mtproxy_parser/releases/latest/download/mtproxy_parser-linux-x64.tar.gz | tar -xz

      - name: Run parser
        run: ./mtproxy_parser -github-repo ${{ github.repository }} -github-token ${{ secrets.GITHUB_TOKEN }} -save-dead -workers 50
```

---

## Лицензия

MIT