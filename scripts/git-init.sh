#!/usr/bin/env bash
# ==============================================================================
# 项目 Git 初始化脚本
# 功能: 初始化 Git 仓库 → 创建 GitHub 远程仓库 → 首次提交推送
# 用法: bash scripts/git-init.sh
# 环境: Windows Git Bash (MINGW64) / Linux / macOS
# ==============================================================================

set -euo pipefail

# ---------- 颜色与日志 ----------

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

info()    { echo -e "${BLUE}[信息]${NC} $*"; }
ok()      { echo -e "${GREEN}[成功]${NC} $*"; }
warn()    { echo -e "${YELLOW}[警告]${NC} $*"; }
fail()    { echo -e "${RED}[错误]${NC} $*"; exit 1; }
header()  { echo -e "\n${CYAN}=== $* ===${NC}"; }

# ---------- 项目根目录定位 ----------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# 仓库名变量 (全局)
REPO_NAME=""
VISIBILITY=""

# ==================== 1. 环境检查 ====================

check_git() {
    header "检查 Git 环境"
    command -v git >/dev/null 2>&1 || fail "未找到 git，请先安装 Git: https://git-scm.com/"
    info "Git 版本: $(git --version)"
}

check_gh() {
    if command -v gh >/dev/null 2>&1; then
        info "GitHub CLI 版本: $(gh --version | head -1)"
        return
    fi

    warn "未找到 GitHub CLI (gh)，正在自动安装..."
    install_gh
}

install_gh() {
    local installed=false

    # Windows: 优先 winget，其次 scoop
    if command -v winget >/dev/null 2>&1; then
        info "使用 winget 安装 gh..."
        if winget install --id GitHub.cli --accept-source-agreements --accept-package-agreements 2>/dev/null; then
            installed=true
        fi
    fi

    if ! $installed && command -v scoop >/dev/null 2>&1; then
        info "使用 scoop 安装 gh..."
        if scoop install gh 2>/dev/null; then
            installed=true
        fi
    fi

    # Linux: apt / yum
    if ! $installed && command -v apt-get >/dev/null 2>&1; then
        info "使用 apt 安装 gh..."
        if sudo apt-get update && sudo apt-get install -y gh 2>/dev/null; then
            installed=true
        fi
    fi

    if ! $installed && command -v yum >/dev/null 2>&1; then
        info "使用 yum 安装 gh..."
        if sudo yum install -y gh 2>/dev/null; then
            installed=true
        fi
    fi

    # macOS: brew
    if ! $installed && command -v brew >/dev/null 2>&1; then
        info "使用 brew 安装 gh..."
        if brew install gh 2>/dev/null; then
            installed=true
        fi
    fi

    if ! $installed; then
        fail "无法自动安装 gh CLI。请手动安装: https://cli.github.com/"
    fi

    # 刷新 PATH（Windows 安装后可能需要）
    hash -r 2>/dev/null || true

    # 验证安装
    if ! command -v gh >/dev/null 2>&1; then
        warn "gh 已安装但不在当前 PATH 中"
        info "请关闭并重新打开终端，然后重新运行此脚本"
        exit 1
    fi

    ok "gh CLI 安装成功: $(gh --version | head -1)"
}

check_gh_auth() {
    header "检查 GitHub 认证"
    if gh auth status >/dev/null 2>&1; then
        ok "GitHub CLI 已认证"
        gh auth status 2>&1 | head -3
    else
        warn "GitHub CLI 尚未认证"
        echo ""
        info "请运行以下命令完成认证:"
        echo "  gh auth login"
        echo ""
        fail "认证完成后请重新运行此脚本"
    fi
}

# ==================== 2. Git 仓库初始化 ====================

init_git_repo() {
    header "初始化 Git 仓库"

    if [ -d ".git" ]; then
        warn "当前目录已是 Git 仓库，跳过初始化"
        info "当前分支: $(git branch --show-current 2>/dev/null || echo 'unknown')"
        return
    fi

    git init
    git branch -M main

    # Windows CRLF 防护
    git config core.autocrlf input
    info "已设置 core.autocrlf=input (防止 Windows CRLF 问题)"

    ok "Git 仓库已初始化 (主分支: main)"
}

# ==================== 3. GitHub 远程仓库 ====================

extract_repo_name() {
    # 从 go.mod 中提取模块路径
    if [ -f "go.mod" ]; then
        local module_path
        module_path=$(head -1 go.mod | awk '{print $2}')
        # 去掉 github.com/ 前缀
        local extracted="${module_path#github.com/}"
        if [ -n "$extracted" ] && [ "$extracted" != "$module_path" ]; then
            REPO_NAME="$extracted"
            info "从 go.mod 检测到仓库名: $REPO_NAME"
            echo ""
            read -rp "使用此仓库名? (y/n, 默认y): " use_detected
            if [ "${use_detected:-y}" = "y" ] || [ "${use_detected:-y}" = "Y" ]; then
                return
            fi
        fi
    fi

    # 回退: 使用目录名或让用户输入
    local default_name
    default_name=$(basename "$PROJECT_ROOT")
    echo ""
    read -rp "请输入 GitHub 仓库名 (格式: owner/repo, 默认: $default_name): " input_name
    REPO_NAME="${input_name:-$default_name}"

    [ -z "$REPO_NAME" ] && fail "仓库名不能为空"
}

select_visibility() {
    echo ""
    info "请选择仓库可见性:"
    echo "  [1] 私有仓库 (private) -- 默认"
    echo "  [2] 公开仓库 (public)"
    echo ""
    read -rp "选择 (1/2, 默认1): " visibility_choice
    case "${visibility_choice:-1}" in
        2) VISIBILITY="public" ;;
        *) VISIBILITY="private" ;;
    esac
    info "仓库可见性: $VISIBILITY"
}

create_github_repo() {
    header "创建 GitHub 远程仓库"

    extract_repo_name
    select_visibility

    # 检查远程仓库是否已存在
    if gh repo view "$REPO_NAME" >/dev/null 2>&1; then
        warn "远程仓库 $REPO_NAME 已存在"

        # 检查 origin 是否已配置
        if git remote get-url origin >/dev/null 2>&1; then
            info "origin 已配置: $(git remote get-url origin)"
        else
            git remote add origin "https://github.com/${REPO_NAME}.git"
            ok "已关联远程仓库: https://github.com/${REPO_NAME}"
        fi
        return
    fi

    # 创建远程仓库
    info "正在创建 $VISIBILITY 仓库: $REPO_NAME ..."

    # 判断是否为 owner/repo 格式 (包含组织)
    if [[ "$REPO_NAME" == *"/"* ]]; then
        local org_name="${REPO_NAME%%/*}"
        local repo_name="${REPO_NAME##*/}"
        gh repo create "$REPO_NAME" --"$VISIBILITY" --description "TG Bot 管理平台" 2>/dev/null || \
            fail "创建仓库失败。请检查是否有 $org_name 组织的权限"
    else
        gh repo create "$REPO_NAME" --"$VISIBILITY" --description "TG Bot 管理平台" 2>/dev/null || \
            fail "创建仓库失败"
    fi

    # 添加 remote
    if ! git remote get-url origin >/dev/null 2>&1; then
        git remote add origin "https://github.com/${REPO_NAME}.git"
    fi

    ok "GitHub 仓库已创建: https://github.com/$REPO_NAME"
}

# ==================== 4. 首次提交 + 推送 ====================

initial_commit_and_push() {
    header "首次提交与推送"

    # 检查是否有文件可提交
    if [ -z "$(git status --porcelain)" ]; then
        warn "没有文件变更，跳过提交"

        # 如果有 remote 且有本地提交，尝试推送
        if git rev-parse HEAD >/dev/null 2>&1 && git remote get-url origin >/dev/null 2>&1; then
            info "尝试推送已有提交..."
            git push -u origin main 2>/dev/null || warn "推送失败（可能已是最新）"
        fi
        return
    fi

    # 暂存所有文件
    git add -A

    # 统计信息
    local file_count
    file_count=$(git diff --cached --numstat | wc -l | tr -d ' ')
    info "暂存文件数: $file_count"

    # 首次提交
    git commit -m "$(cat <<'EOF'
feat: 初始化项目

- Go 后端 (Gin + GORM) 六边形架构
- React 前端 (Ant Design + TypeScript)
- MySQL + Redis 基础设施
- Docker Compose 部署配置
- AI 开发团队配置 (6 agents)
- Git 自动化工作流脚本
EOF
)"

    ok "首次提交完成"

    # 推送
    if git remote get-url origin >/dev/null 2>&1; then
        info "正在推送到 origin/main ..."
        git push -u origin main
        ok "已推送到远程仓库"
    else
        warn "未配置远程仓库，跳过推送"
    fi
}

# ==================== 5. 完成汇总 ====================

print_summary() {
    echo ""
    echo "========================================"
    ok "项目 Git 初始化完成!"
    echo "========================================"
    echo ""

    if [ -n "$REPO_NAME" ]; then
        info "仓库地址:  https://github.com/$REPO_NAME"
    fi
    info "当前分支:  $(git branch --show-current 2>/dev/null || echo 'main')"
    info "提交数量:  $(git rev-list --count HEAD 2>/dev/null || echo '0')"
    info "跟踪文件:  $(git ls-files 2>/dev/null | wc -l | tr -d ' ') 个"
    echo ""
    info "后续操作:"
    echo "  make review     -- 查看代码变更"
    echo "  make commit     -- AI 审查 + 自动提交"
    echo "  make push       -- 推送到远程"
    echo "  make git-status -- 查看 Git 状态"
    echo "  make git-log    -- 查看提交历史"
    echo "========================================"
}

# ==================== 主流程 ====================

main() {
    echo "========================================"
    info "TG Bot 管理平台 - Git 项目初始化"
    echo "========================================"

    check_git
    check_gh
    check_gh_auth

    init_git_repo
    create_github_repo
    initial_commit_and_push
    print_summary
}

main "$@"
