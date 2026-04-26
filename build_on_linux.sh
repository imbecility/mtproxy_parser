#!/bin/bash

set -e

# имя программы из имени директории или go.mod
APP_NAME=$(basename "$(go list -m 2>/dev/null || basename "$(pwd)")")

# папки для бинарников и архивов
BUILD_DIR="build"
DIST_DIR="dist"

# цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()    { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $1"; }

check_dependencies() {
    local missing=0
    for cmd in go zip tar; do
        if ! command -v "$cmd" &>/dev/null; then
            log_error "не найден: ${cmd}"
            missing=1
        fi
    done

    if [ "$missing" -eq 1 ]; then
        log_error "сначала нужно установить недостающие зависимости!"
        exit 1
    fi
}

# ---------------------------------------------------------------------------

check_dependencies

log_info "подготовка директорий..."

if [ -d "$BUILD_DIR" ]; then
    log_warn "очистка папки '$BUILD_DIR'..."
    rm -rf "$BUILD_DIR"
fi

if [ -d "$DIST_DIR" ]; then
    log_warn "очистка папки '$DIST_DIR'..."
    rm -rf "$DIST_DIR"
fi

mkdir -p \
    "$BUILD_DIR/linux/arm" \
    "$BUILD_DIR/linux/x64" \
    "$BUILD_DIR/macos/arm" \
    "$BUILD_DIR/macos/intel" \
    "$BUILD_DIR/windows/arm" \
    "$BUILD_DIR/windows/x64" \
    "$DIST_DIR"

log_info "структура директорий ./'$BUILD_DIR'/ и ./'$DIST_DIR'/ создана"

# ---------------------------------------------------------------------------
# матрица сборки: "GOOS GOARCH подпапка суффикс_бинарника"
# ---------------------------------------------------------------------------
#   колонки:
#     1) GOOS
#     2) GOARCH
#     3) путь внутри build/
#     4) суффикс имени файла (пустой для Unix, .exe для Windows)
# ---------------------------------------------------------------------------
declare -a TARGETS=(
    "linux   arm   linux/arm    "
    "linux   amd64 linux/x64    "
    "darwin  arm64 macos/arm    "
    "darwin  amd64 macos/intel    "
    "windows arm64  windows/arm  .exe"
    "windows amd64 windows/x64  .exe"
)

# ---------------------------------------------------------------------------

log_info "компиляция приложения: ${APP_NAME}"
echo "-------------------------------------------"
go mod tidy
for target in "${TARGETS[@]}"; do
    # разбор строки
    read -r GOOS GOARCH SUB_DIR EXT <<< "$target"

    BINARY_NAME="${APP_NAME}${EXT}"
    OUTPUT_PATH="${BUILD_DIR}/${SUB_DIR}/${BINARY_NAME}"

    # метки для имени архива: linux-arm, macos-arm, windows-x64 ...
    ARCH_LABEL=$(basename "$SUB_DIR")          # последний сегмент arm / x64 / arm64
    OS_LABEL=$(echo "$SUB_DIR" | cut -d'/' -f1) # linux / macos / windows

    ARCHIVE_NAME="${APP_NAME}-${OS_LABEL}-${ARCH_LABEL}"

    log_info "сборка: GOOS=${GOOS} GOARCH=${GOARCH}  ->  ${OUTPUT_PATH}"

    GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags="-s -w -extldflags '-static'" -trimpath -o "$OUTPUT_PATH" .

    # упаковка в архивы
    if [ "$GOOS" = "windows" ]; then
        # windows: zip-архив
        ARCHIVE_FILE="${DIST_DIR}/${ARCHIVE_NAME}.zip"
        log_info "архивирование -> ${ARCHIVE_FILE}"
        zip -j "$ARCHIVE_FILE" "$OUTPUT_PATH"
    else
        # Linux / macOS: tar.gz
        ARCHIVE_FILE="${DIST_DIR}/${ARCHIVE_NAME}.tar.gz"
        log_info "архивирование -> ${ARCHIVE_FILE}"
        tar -czf "$ARCHIVE_FILE" -C "${BUILD_DIR}/${SUB_DIR}" "$BINARY_NAME"
    fi

    echo "-------------------------------------------"
done

# ---------------------------------------------------------------------------

log_info "сборка завершена!"
log_info "архивы находятся в '${DIST_DIR}':"
ls -lh "$DIST_DIR"
log_info "бинарники находятся в '${BUILD_DIR}'"