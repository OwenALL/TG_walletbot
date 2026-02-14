#!/usr/bin/env bash
# ==============================================================================
# 自动代码审查 + 提交推送脚本
# 功能: 分析代码变更 → 安全检查 → 生成提交消息 → 提交推送
# 用法:
#   bash scripts/auto-review-commit.sh review              # 仅审查
#   bash scripts/auto-review-commit.sh commit              # 自动提交
#   bash scripts/auto-review-commit.sh commit "feat: xxx"  # 指定消息提交
# 环境: Windows Git Bash (MINGW64) / Linux / macOS
# ==============================================================================

set -euo pipefail

# ---------- 颜色与日志 ----------

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m'

info()    { echo -e "${BLUE}[信息]${NC} $*"; }
ok()      { echo -e "${GREEN}[成功]${NC} $*"; }
warn()    { echo -e "${YELLOW}[警告]${NC} $*"; }
fail()    { echo -e "${RED}[错误]${NC} $*"; exit 1; }
header()  { echo -e "\n${CYAN}=== $* ===${NC}"; }

# ---------- 项目根目录 ----------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# ---------- 参数解析 ----------

ACTION="${1:-review}"
CUSTOM_MSG="${2:-}"

# ---------- 全局变量 ----------

ALL_CHANGED_FILES=""
STAGED=0
UNSTAGED=0
UNTRACKED=0
TOTAL=0
GO_COUNT=0
TS_COUNT=0
SQL_COUNT=0
CONFIG_COUNT=0
DOC_COUNT=0
SCRIPT_COUNT=0
OTHER_COUNT=0
COMMIT_MSG=""

# ---------- 敏感文件模式 ----------

SENSITIVE_PATTERNS=(
    ".env"
    ".env.*"
    "configs/config.yaml"
    "*.pem"
    "*.key"
    "*.p12"
    "*.pfx"
    "id_rsa"
    "id_ed25519"
)

# ==================== 变更检测 ====================

detect_changes() {
    header "代码变更检测"

    # 统计各类变更
    STAGED=$(git diff --cached --name-only 2>/dev/null | wc -l | tr -d ' ')
    UNSTAGED=$(git diff --name-only 2>/dev/null | wc -l | tr -d ' ')
    UNTRACKED=$(git ls-files --others --exclude-standard 2>/dev/null | wc -l | tr -d ' ')
    TOTAL=$((STAGED + UNSTAGED + UNTRACKED))

    if [ "$TOTAL" -eq 0 ]; then
        ok "没有检测到任何变更，无需操作"
        exit 0
    fi

    info "变更统计:"
    [ "$STAGED" -gt 0 ]    && echo -e "  ${GREEN}已暂存:${NC}   $STAGED 个文件"
    [ "$UNSTAGED" -gt 0 ]  && echo -e "  ${YELLOW}未暂存:${NC}   $UNSTAGED 个文件"
    [ "$UNTRACKED" -gt 0 ] && echo -e "  ${MAGENTA}未跟踪:${NC}   $UNTRACKED 个文件"
    echo -e "  合计:     $TOTAL 个文件"

    # 收集所有变更文件（去重）
    ALL_CHANGED_FILES=$(
        {
            git diff --cached --name-only 2>/dev/null
            git diff --name-only 2>/dev/null
            git ls-files --others --exclude-standard 2>/dev/null
        } | sort -u
    )
}

# ==================== 分类统计 ====================

categorize_changes() {
    header "变更分类"

    GO_COUNT=0; TS_COUNT=0; SQL_COUNT=0; CONFIG_COUNT=0; DOC_COUNT=0; SCRIPT_COUNT=0; OTHER_COUNT=0

    while IFS= read -r file; do
        [ -z "$file" ] && continue
        case "$file" in
            *.go)
                GO_COUNT=$((GO_COUNT + 1))
                ;;
            *.ts|*.tsx|*.js|*.jsx|*.css|*.scss|*.less)
                TS_COUNT=$((TS_COUNT + 1))
                ;;
            *.sql)
                SQL_COUNT=$((SQL_COUNT + 1))
                ;;
            *.yaml|*.yml|*.toml|*.json|Dockerfile*|docker-compose*|nginx*|Makefile|*.conf)
                CONFIG_COUNT=$((CONFIG_COUNT + 1))
                ;;
            *.md|*.txt|*.doc|*.rst)
                DOC_COUNT=$((DOC_COUNT + 1))
                ;;
            *.sh|*.bat|*.cmd|*.ps1)
                SCRIPT_COUNT=$((SCRIPT_COUNT + 1))
                ;;
            *)
                OTHER_COUNT=$((OTHER_COUNT + 1))
                ;;
        esac
    done <<< "$ALL_CHANGED_FILES"

    [ $GO_COUNT -gt 0 ]     && echo -e "  ${GREEN}Go 后端:${NC}     $GO_COUNT 个文件"
    [ $TS_COUNT -gt 0 ]     && echo -e "  ${BLUE}前端代码:${NC}    $TS_COUNT 个文件"
    [ $SQL_COUNT -gt 0 ]    && echo -e "  ${CYAN}SQL 迁移:${NC}    $SQL_COUNT 个文件"
    [ $CONFIG_COUNT -gt 0 ] && echo -e "  ${YELLOW}配置文件:${NC}    $CONFIG_COUNT 个文件"
    [ $DOC_COUNT -gt 0 ]    && echo -e "  ${MAGENTA}文档:${NC}        $DOC_COUNT 个文件"
    [ $SCRIPT_COUNT -gt 0 ] && echo -e "  脚本:        $SCRIPT_COUNT 个文件"
    [ $OTHER_COUNT -gt 0 ]  && echo -e "  其他:        $OTHER_COUNT 个文件"
}

# ==================== 安全检查 ====================

security_check() {
    header "安全检查"

    local has_sensitive=false
    local sensitive_files=()

    while IFS= read -r file; do
        [ -z "$file" ] && continue
        local basename_file
        basename_file=$(basename "$file")

        for pattern in "${SENSITIVE_PATTERNS[@]}"; do
            # 检查完整路径匹配和文件名匹配
            if [[ "$file" == $pattern ]] || [[ "$basename_file" == $pattern ]]; then
                has_sensitive=true
                sensitive_files+=("$file")
                break
            fi
        done
    done <<< "$ALL_CHANGED_FILES"

    if $has_sensitive; then
        warn "检测到可能包含敏感信息的文件:"
        for sf in "${sensitive_files[@]}"; do
            echo -e "  ${RED}!!${NC} $sf"
        done
        echo ""
        if [ "$ACTION" = "commit" ]; then
            warn "以上文件将被自动排除在提交之外"
        fi
    else
        ok "未发现敏感文件"
    fi
}

# ==================== 变更摘要 ====================

show_diff_summary() {
    header "变更内容摘要"

    # 行级统计（已有提交时显示与 HEAD 的差异）
    if git rev-parse HEAD >/dev/null 2>&1; then
        echo ""
        git diff --stat HEAD 2>/dev/null || true
    fi

    # 显示新增的文件列表（最多显示 20 个）
    if [ "$UNTRACKED" -gt 0 ]; then
        echo ""
        info "新增文件:"
        git ls-files --others --exclude-standard 2>/dev/null | head -20 | while read -r f; do
            echo -e "  ${GREEN}+${NC} $f"
        done
        if [ "$UNTRACKED" -gt 20 ]; then
            echo "  ... 还有 $((UNTRACKED - 20)) 个文件"
        fi
    fi

    # 建议的 commit 类型
    echo ""
    suggest_commit_type
}

# ==================== Commit 类型建议 ====================

suggest_commit_type() {
    info "建议的提交类型:"

    if [ "$UNTRACKED" -gt "$((TOTAL / 2))" ]; then
        echo -e "  ${GREEN}feat${NC} -- 大量新文件，可能是新功能"
    fi
    if [ "$GO_COUNT" -gt 0 ] && [ "$TS_COUNT" -gt 0 ]; then
        echo -e "  ${BLUE}feat${NC} -- 前后端同时变更，可能是完整功能"
    fi
    if [ "$SQL_COUNT" -gt 0 ]; then
        echo -e "  ${CYAN}feat(db)${NC} -- 包含数据库迁移"
    fi
    if [ "$DOC_COUNT" -gt 0 ] && [ "$DOC_COUNT" -eq "$TOTAL" ]; then
        echo -e "  ${MAGENTA}docs${NC} -- 纯文档变更"
    fi
    if [ "$CONFIG_COUNT" -gt 0 ] && [ "$CONFIG_COUNT" -eq "$TOTAL" ]; then
        echo -e "  ${YELLOW}chore${NC} -- 纯配置变更"
    fi
    if [ "$SCRIPT_COUNT" -gt 0 ] && [ "$SCRIPT_COUNT" -eq "$TOTAL" ]; then
        echo -e "  chore(infra) -- 脚本/基础设施变更"
    fi
}

# ==================== Commit 消息生成 ====================

generate_commit_message() {
    # 用户手动指定了消息
    if [ -n "$CUSTOM_MSG" ]; then
        COMMIT_MSG="$CUSTOM_MSG"
        return
    fi

    local commit_type="chore"
    local scope=""
    local description=""

    # ---------- 推断 scope ----------

    local has_backend=false has_frontend=false has_bot=false has_db=false
    local has_deploy=false has_config=false has_infra=false

    if echo "$ALL_CHANGED_FILES" | grep -qE "^(internal/|cmd/|pkg/)"; then
        has_backend=true
    fi
    if echo "$ALL_CHANGED_FILES" | grep -qE "^web/"; then
        has_frontend=true
    fi
    if echo "$ALL_CHANGED_FILES" | grep -qE "(adapter/telegram|cmd/bot)"; then
        has_bot=true
    fi
    if echo "$ALL_CHANGED_FILES" | grep -qE "^migrations/"; then
        has_db=true
    fi
    if echo "$ALL_CHANGED_FILES" | grep -qE "^(deployments/|Dockerfile)"; then
        has_deploy=true
    fi
    if echo "$ALL_CHANGED_FILES" | grep -qE "^configs/"; then
        has_config=true
    fi
    if echo "$ALL_CHANGED_FILES" | grep -qE "^(\.claude/|\.github/|scripts/|Makefile)"; then
        has_infra=true
    fi

    # 组装 scope
    local scopes=()
    $has_bot && scopes+=("bot")
    $has_backend && ! $has_bot && scopes+=("backend")
    $has_frontend && scopes+=("frontend")
    $has_db && scopes+=("db")
    $has_deploy && scopes+=("deploy")
    $has_config && scopes+=("config")
    $has_infra && scopes+=("infra")

    if [ ${#scopes[@]} -eq 1 ]; then
        scope="${scopes[0]}"
    elif [ ${#scopes[@]} -gt 1 ]; then
        # 多个 scope 时取前两个
        scope="${scopes[0]},${scopes[1]}"
    fi

    # ---------- 推断 commit type ----------

    # 测试文件
    if echo "$ALL_CHANGED_FILES" | grep -qE "(_test\.go|\.test\.(ts|tsx)|\.spec\.(ts|tsx))$"; then
        if [ $GO_COUNT -le 2 ] && [ $TS_COUNT -le 2 ]; then
            commit_type="test"
        fi
    fi

    # 纯文档
    if [ "$DOC_COUNT" -eq "$TOTAL" ] && [ "$TOTAL" -gt 0 ]; then
        commit_type="docs"
    # 大量新文件 = 新功能
    elif [ "$UNTRACKED" -gt "$((TOTAL / 2))" ] && [ "$TOTAL" -gt 0 ]; then
        commit_type="feat"
    # 纯配置/脚本
    elif [ "$CONFIG_COUNT" -eq "$TOTAL" ] || [ "$SCRIPT_COUNT" -eq "$TOTAL" ]; then
        commit_type="chore"
    # 有新文件
    elif [ "$UNTRACKED" -gt 0 ]; then
        commit_type="feat"
    # 纯修改
    else
        commit_type="refactor"
    fi

    # ---------- 生成描述 ----------

    if [ "$TOTAL" -eq 1 ]; then
        local single_file
        single_file=$(echo "$ALL_CHANGED_FILES" | head -1)
        description="更新 $(basename "$single_file")"
    else
        if $has_backend && $has_frontend; then
            description="更新前后端代码 (Go:$GO_COUNT, TS:$TS_COUNT)"
        elif $has_backend && ! $has_frontend; then
            if [ $GO_COUNT -gt 0 ]; then
                description="更新后端代码 ($GO_COUNT 个文件)"
            else
                description="更新后端配置"
            fi
        elif $has_frontend && ! $has_backend; then
            description="更新前端代码 ($TS_COUNT 个文件)"
        elif $has_db; then
            description="更新数据库迁移 ($SQL_COUNT 个文件)"
        elif $has_deploy; then
            description="更新部署配置"
        elif $has_infra; then
            description="更新项目基础设施"
        elif [ "$DOC_COUNT" -eq "$TOTAL" ]; then
            description="更新项目文档"
        else
            description="更新 $TOTAL 个文件"
        fi
    fi

    # ---------- 组装消息 ----------

    if [ -n "$scope" ]; then
        COMMIT_MSG="${commit_type}(${scope}): ${description}"
    else
        COMMIT_MSG="${commit_type}: ${description}"
    fi
}

# ==================== 执行提交 ====================

do_commit() {
    header "执行提交"

    generate_commit_message
    info "提交消息: $COMMIT_MSG"
    echo ""

    # 暂存所有变更
    git add -A

    # 排除敏感文件
    for pattern in "${SENSITIVE_PATTERNS[@]}"; do
        # 使用通配符排除
        git reset HEAD -- "$pattern" 2>/dev/null || true
    done

    # 检查暂存区是否有内容
    if [ -z "$(git diff --cached --name-only)" ]; then
        warn "暂存区为空（可能所有文件都被安全规则排除了）"
        exit 0
    fi

    # 显示将要提交的文件数
    local commit_count
    commit_count=$(git diff --cached --name-only | wc -l | tr -d ' ')
    info "即将提交 $commit_count 个文件"

    # 执行提交
    git commit -m "$COMMIT_MSG"
    ok "提交成功"
}

# ==================== 推送 ====================

do_push() {
    header "推送到远程"

    if ! git remote get-url origin >/dev/null 2>&1; then
        warn "未配置远程仓库 (origin)，跳过推送"
        info "请先运行: make git-init"
        return
    fi

    local branch
    branch=$(git branch --show-current)

    git push origin "$branch"
    ok "已推送到 origin/$branch"
}

# ==================== 结果汇总 ====================

print_result() {
    echo ""
    echo "========================================"
    ok "操作完成!"
    echo "========================================"
    info "最近提交: $(git log --oneline -1 2>/dev/null || echo '无')"
    info "当前分支: $(git branch --show-current)"

    # 远程状态
    if git remote get-url origin >/dev/null 2>&1; then
        info "远程状态: $(git status -sb 2>/dev/null | head -1)"
    fi
    echo "========================================"
}

# ==================== 主流程 ====================

main() {
    # 前置检查
    [ -d ".git" ] || fail "当前目录不是 Git 仓库。请先运行: make git-init"

    case "$ACTION" in
        review)
            detect_changes
            categorize_changes
            security_check
            show_diff_summary
            ;;
        commit)
            detect_changes
            categorize_changes
            security_check
            show_diff_summary
            echo ""
            do_commit
            do_push
            print_result
            ;;
        *)
            echo "用法: $0 {review|commit} [提交消息]"
            echo ""
            echo "  review  -- 仅审查变更，不提交"
            echo "  commit  -- 审查 + 提交 + 推送"
            echo ""
            echo "示例:"
            echo "  $0 review"
            echo "  $0 commit"
            echo "  $0 commit \"feat(bot): 实现 /start 命令\""
            exit 1
            ;;
    esac
}

main "$@"
