#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 honeok <i@honeok.com>

set -eExo pipefail

die() {
    printf >&2 "Error: %s\n" "$*"
    exit 1
}

curl() {
    local RET
    # 添加 --fail 不然404退出码也为0
    # 32位cygwin已停止更新, 证书可能有问题, 添加 --insecure
    # centos7 curl 不支持 --retry-connrefused --retry-all-errors 因此手动 retry
    for ((i = 1; i <= 5; i++)); do
        command curl --connect-timeout 10 --fail --insecure "$@"
        RET="$?"
        if [ "$RET" -eq 0 ]; then
            return
        else
            # 403 404 错误或达到重试次数
            if [ "$RET" -eq 22 ] || [ "$i" -eq 5 ]; then
                return "$RET"
            fi
            sleep 1
        fi
    done
}

is_china() {
    if [ -z "$COUNTRY" ]; then
        if ! COUNTRY="$(curl -Ls -4 http://www.qualcomm.cn/cdn-cgi/trace | grep '^loc=' | cut -d= -f2 | grep .)"; then
            die "Can not get location."
        fi
        echo >&1 "Location: $COUNTRY"
    fi
    [ "$COUNTRY" = CN ]
}

if is_china; then
    GO_MIRROR="golang.google.cn"
else
    GO_MIRROR="go.dev"
fi

# 官方版本
OFFICIAL_VER="$(curl -Ls "https://$GO_MIRROR/dl/?mode=json" | awk '/"version"/ && !p { sub(/.*"go/, ""); sub(/".*/, ""); print; p=1 }')"

find "$PWD" -type f -name "go.mod" -not -path '*/.*' | while read -r f; do
    WORK_DIR="$(dirname "$f")"

    cd "$WORK_DIR" || exit 1
    LOCAL_VER="$(awk '/^[[:space:]]*go[[:space:]]+[0-9]+\.[0-9]+(\.[0-9]+)?$/ {print $2; exit}' go.mod || true)" # 本地版本
    if [ "$LOCAL_VER" != "$OFFICIAL_VER" ]; then
        sed -i -E "s#^[[:space:]]*go[[:space:]]+[0-9]+\.[0-9]+(\.[0-9]+)?#go $OFFICIAL_VER#" go.mod
        go mod tidy
    fi
done
