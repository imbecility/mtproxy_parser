[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8

$ErrorActionPreference = "Stop"

# ---------------------------------------------------------------------------
function Write-Info  { param([string]$Message); Write-Host "[INFO] $Message"  -ForegroundColor Green  }
function Write-Warn  { param([string]$Message); Write-Host "[WARN] $Message"  -ForegroundColor Yellow }
function Write-Err   { param([string]$Message); Write-Host "[ERROR] $Message" -ForegroundColor Red    }

# ---------------------------------------------------------------------------

function Test-Dependencies {
    $missing = $false

    foreach ($cmd in @("go", "tar")) {
        if (-not (Get-Command $cmd -ErrorAction SilentlyContinue)) {
            Write-Err "не найден: $cmd"
            $missing = $true
        }
    }

    if ($missing) {
        Write-Err "сначала нужно установить недостающие зависимости!"
        exit 1
    }
}

# ---------------------------------------------------------------------------

function Get-AppName {
    try {
        $module = go list -m 2>$null
        if ($module) {
            return [System.IO.Path]::GetFileName($module)
        }
    } catch {}

    return [System.IO.Path]::GetFileName((Get-Location).Path)
}

# ---------------------------------------------------------------------------

Test-Dependencies

$APP_NAME  = Get-AppName
$BUILD_DIR = "build"
$DIST_DIR  = "dist"

# ---------------------------------------------------------------------------

Write-Info "подготовка директорий..."

foreach ($dir in @($BUILD_DIR, $DIST_DIR)) {
    if (Test-Path $dir) {
        Write-Warn "очистка папки '$dir'..."
        Remove-Item -Recurse -Force $dir
    }
}

$subDirs = @(
    "$BUILD_DIR/linux/arm",
    "$BUILD_DIR/linux/x64",
    "$BUILD_DIR/macos/arm",
    "$BUILD_DIR/macos/intel",
    "$BUILD_DIR/windows/arm",
    "$BUILD_DIR/windows/x64",
    $DIST_DIR
)

foreach ($dir in $subDirs) {
    New-Item -ItemType Directory -Path $dir -Force | Out-Null
}

Write-Info "структура директорий './$BUILD_DIR/' и './$DIST_DIR/' создана"

# ---------------------------------------------------------------------------
# матрица сборки
# ---------------------------------------------------------------------------
#   ключи хэш-таблицы:
#     Goos   - GOOS
#     Goarch - GOARCH
#     SubDir - путь внутри build/
#     Ext    - суффикс бинарника (пустой для unix, .exe для windows)
# ---------------------------------------------------------------------------
$targets = @(
    @{ Goos = "linux";   Goarch = "arm";   SubDir = "linux/arm";    Ext = ""     },
    @{ Goos = "linux";   Goarch = "amd64"; SubDir = "linux/x64";    Ext = ""     },
    @{ Goos = "darwin";  Goarch = "arm64"; SubDir = "macos/arm";    Ext = ""     },
    @{ Goos = "darwin";  Goarch = "amd64"; SubDir = "macos/intel";  Ext = ""     },
    @{ Goos = "windows"; Goarch = "arm64"; SubDir = "windows/arm";  Ext = ".exe" },
    @{ Goos = "windows"; Goarch = "amd64"; SubDir = "windows/x64";  Ext = ".exe" }
)

# ---------------------------------------------------------------------------

Write-Info "компиляция приложения: $APP_NAME"
Write-Host "-------------------------------------------"
go mod tidy
foreach ($target in $targets) {
    $goos       = $target.Goos
    $goarch     = $target.Goarch
    $subDir     = $target.SubDir
    $ext        = $target.Ext

    $binaryName  = "$APP_NAME$ext"
    $outputPath  = "$BUILD_DIR/$subDir/$binaryName"

    # метки для имени архива
    $archLabel   = Split-Path $subDir -Leaf          # arm / x64 / intel
    $osLabel     = ($subDir -split "/")[0]           # linux / macos / windows

    $archiveName = "$APP_NAME-$osLabel-$archLabel"

    Write-Info "сборка: GOOS=$goos GOARCH=$goarch  ->  $outputPath"

    $env:GOOS   = $goos
    $env:GOARCH = $goarch

    go build -ldflags="-s -w" -trimpath -o $outputPath .

    # упаковка в архивы
    if ($goos -eq "windows") {
        # Windows: zip-архив
        $archiveFile = "$DIST_DIR/$archiveName.zip"
        Write-Info "архивирование -> $archiveFile"

        Compress-Archive -Path $outputPath -DestinationPath $archiveFile -Force
    } else {
        # Linux / macOS: tar.gz
        $archiveFile = "$DIST_DIR/$archiveName.tar.gz"
        Write-Info "архивирование -> $archiveFile"

        tar -czf $archiveFile -C "$BUILD_DIR/$subDir" $binaryName
    }

    Write-Host "-------------------------------------------"
}

# сброс переменных окружения
Remove-Item Env:\GOOS   -ErrorAction SilentlyContinue
Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue

# ---------------------------------------------------------------------------

Write-Info "сборка завершена!"
Write-Info "архивы находятся в '$DIST_DIR':"
Get-ChildItem $DIST_DIR | Format-Table Name, Length, LastWriteTime
Write-Info "бинарники находятся в '$BUILD_DIR'"