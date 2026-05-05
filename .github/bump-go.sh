#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 honeok <i@honeok.com>

set -eEo pipefail

die() {
    printf >&2 "Error: %s\n" "$*"
    exit 1
}

curl() {
    local rc

    # 添加 --fail 不然404退出码也为0
    # 32位cygwin已停止更新, 证书可能有问题, 添加 --insecure
    # centos7 curl 不支持 --retry-connrefused --retry-all-errors 因此手动 retry
    for ((i = 1; i <= 5; i++)); do
        command curl --connect-timeout 10 --fail --insecure "$@"
        rc="$?"
        if [ "$rc" -eq 0 ]; then
            return
        else
            # 403 404 错误或达到重试次数
            if [ "$rc" -eq 22 ] || [ "$i" -eq 5 ]; then
                return "$rc"
            fi
            sleep 0.5
        fi
    done
}

is_in_china() {
    if [ -z "$COUNTRY" ]; then
        # www.prologis.cn
        # www.autodesk.com.cn
        # www.keysight.com.cn
        if ! COUNTRY="$(curl -L -4 http://www.qualcomm.cn/cdn-cgi/trace | grep '^loc=' | cut -d= -f2 | grep .)"; then
            die "Can not get location."
        fi
        echo >&2 "Location: $COUNTRY"
    fi
    [ "$COUNTRY" = CN ]
}

main() {
    local go_mirror official_var work_dir local_ver

    if is_in_china; then
        go_mirror="golang.google.cn"
    else
        go_mirror="go.dev"
    fi

    # 官方版本
    official_var="$(curl -L "https://$go_mirror/dl/?mode=json" | awk '/"version"/ && !p { sub(/.*"go/, ""); sub(/".*/, ""); print; p=1 }')"

    find "$PWD" -type f -name "go.mod" -not -path '*/.*' | while read -r f; do
        work_dir="$(dirname "$f")"

        cd "$work_dir" || exit 1
        local_ver="$(awk '/^[[:space:]]*go[[:space:]]+[0-9]+\.[0-9]+(\.[0-9]+)?$/ {print $2; exit}' go.mod || true)" # 本地版本
        if [ "$local_ver" != "$official_var" ]; then
            sed -E -i "s#^[[:space:]]*go[[:space:]]+[0-9]+\.[0-9]+(\.[0-9]+)?#go $official_var#" go.mod
            go mod tidy
        fi
    done
}

main
