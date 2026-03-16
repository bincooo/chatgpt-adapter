#!/usr/bin/env bash
set -euo pipefail

# ── 颜色定义 ─────────────────────────────────────────────────────────
readonly NC='\033[0m'          # 无颜色
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[0;33m'
readonly BLUE='\033[0;34m'

# ── 带颜色的日志辅助函数 ───────────────────────────────────────────────
log()   { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*" >&2; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }

# ── 全局变量 ─────────────────────────────────────────────────────────
GO_BIN=$(command -v go || true)
PROJECT_ROOT=$(pwd)
PATCH_DIR=""
GOMODCACHE=""
VERBOSE=false
REVERT=false          # 是否为还原模式

# ── 帮助信息 ─────────────────────────────────────────────────────────
show_help() {
    cat <<EOF
Usage: $0 [options] <patch-dir>

Apply all .patch files from <patch-dir> to their corresponding Go modules.
Each patch file must contain a header line: '# module: <module@version>'

Options:
  -h, --help     显示此帮助信息
  -v, --verbose  输出更多执行细节
  -r, --revert   还原已应用的补丁（等效于 git apply -R）

Example:
  $0 ./patches          # 应用补丁
  $0 -r ./patches       # 还原补丁
EOF
}

# ── 启动横幅 ──────────────────────────────────────────────────────────
print_banner() {
    echo -e "${BLUE}"
    echo "=============================================="
    echo "         Go Module Patch Applicator           "
    echo "=============================================="
    echo -e "${NC}"
}

# ── 环境检查 ──────────────────────────────────────────────────────────
check_env() {
    if [[ -z "$GO_BIN" ]]; then
        error "未找到 go 命令"
        exit 1
    fi
    if ! command -v git &>/dev/null; then
        error "未找到 git 命令（应用补丁需要）"
        exit 1
    fi
    GOMODCACHE=$($GO_BIN env GOMODCACHE)
}

# ── 从补丁头解析模块信息 ────────────────────────────────────────────────
parse_patch_module() {
    local patch_file="$1"
    grep -m1 '^# module:' "$patch_file" | sed 's/# module:[[:space:]]*//'
}

# ── 提取模块路径（不含版本） ─────────────────────────────────────────────
module_path() {
    local module="$1"
    echo "$module" | cut -d '@' -f1
}

# ── 解析模块目录（优先 vendor，再使用 go list）───────────────────────────
resolve_module_dir() {
    local module="$1"
    local path
    local vendor_dir

    path=$(module_path "$module")
    vendor_dir="$PROJECT_ROOT/vendor/$path"

    if [[ -d "$vendor_dir" ]]; then
        echo "$vendor_dir"
        return
    fi

    $GO_BIN list -m -f "{{.Dir}}" "$module" 2>/dev/null || true
}

# ── 安全应用或还原单个补丁 ───────────────────────────────────────────────
apply_patch() {
    local patch_file="$PROJECT_ROOT/$1"          # 直接使用传入的路径（已为绝对或正确相对路径）
    local module
    local module_dir
    local action
    local action_past

    if $REVERT; then
        action="还原"
        action_past="还原"
    else
        action="应用"
        action_past="应用"
    fi

    log "正在${action}补丁: $(basename "$patch_file")"
    module=$(parse_patch_module "$patch_file")

    if [[ -z "$module" ]]; then
        warn "未找到模块头，跳过"
        return
    fi

    if $VERBOSE; then
        log "模块: $module"
    fi

    module_dir=$(resolve_module_dir "$module")

    if [[ -z "$module_dir" || ! -d "$module_dir" ]]; then
        warn "未找到模块目录: $module"
        return
    fi

    if $VERBOSE; then
        log "目录: $module_dir"
    fi

    # 确保有写权限
    chmod -R u+w "$module_dir"
    pushd "$module_dir" >/dev/null

    if $REVERT; then
        # 还原模式：检查是否可以反向应用
        if git apply --check -R "$patch_file" 2>/dev/null; then
            git apply -R "$patch_file"
            log "补丁还原成功"
        else
            warn "补丁还原跳过（可能尚未应用或冲突）"
        fi
    else
        # 应用模式
        if git apply --check "$patch_file" 2>/dev/null; then
            git apply "$patch_file"
            log "补丁应用成功"
        else
            warn "补丁跳过（可能已应用或存在冲突）"
        fi
    fi

    popd >/dev/null
}

# ── 遍历所有补丁文件 ────────────────────────────────────────────────────
apply_all_patches() {
    local count=0
    local patch

    for patch in "$PATCH_DIR"/*.patch; do
        [[ -e "$patch" ]] || continue
        apply_patch "$patch"
        count=$((count+1))
        echo
    done

    if $REVERT; then
        log "总共还原的补丁数: $count"
    else
        log "总共处理的补丁数: $count"
    fi
}

# ── 主函数 ────────────────────────────────────────────────────────────
main() {
    # 解析命令行选项
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -h|--help)
                show_help
                exit 0 ;;
            -v|--verbose)
                VERBOSE=true
                shift ;;
            -r|--revert)
                REVERT=true
                shift ;;
            --)
                shift
                break ;;
            -*)
                error "未知选项: $1"
                show_help
                exit 1 ;;
            *)
                break ;;
        esac
        # 已删除多余的 shift
    done

    PATCH_DIR="${1:-}"
    if [[ -z "$PATCH_DIR" ]]; then
        error "缺少补丁目录参数"
        show_help
        exit 1
    fi

    if [[ ! -d "$PATCH_DIR" ]]; then
        error "补丁目录不存在: $PATCH_DIR"
        exit 1
    fi

    print_banner
    check_env

    log "补丁目录: $PATCH_DIR"
    log "Go 模块缓存: $GOMODCACHE"
    echo

    apply_all_patches
    log "完成。"
    return 0
}

main "$@"
exit 0