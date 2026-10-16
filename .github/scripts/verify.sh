#!/bin/bash
# ハンズオン教材の検証スクリプト。
# CI(GitHub Actions)から実行されるが、リポジトリのルートで
# ./.github/scripts/verify.sh として手元でも実行できる。
set -euo pipefail

root=$(cd "$(dirname "$0")/../.." && pwd)
steps="step01 step02 step03 step04 step05 step06 step07 step08"
module=github.com/gohandson/goroutineleak-ja

# 問題(skeleton)に上書きして解答例(solution)と一致させるファイル
answers() {
	case "$1" in
	step01) echo "search/search_test.go" ;;
	step02) echo "search/search_test.go cmd/monitor/main.go" ;;
	step03) echo "search/search_test.go go.mod go.sum" ;;
	step04) echo "internal/mygoleak/mygoleak.go" ;;
	step05) echo "cmd/server/main.go" ;;
	step06) echo "search/search.go" ;;
	step07) echo "search/search.go" ;;
	step08) echo "leaks/leaks_test.go" ;;
	esac
}

echo "==> solution: build / vet"
for s in $steps; do
	go -C "$root/solution/$s" build ./...
	go -C "$root/solution/$s" vet ./...
done

echo "==> solution: gofmt / goimports"
fmt=$(gofmt -l "$root/solution" "$root/skeleton")
if [ -n "$fmt" ]; then
	echo "gofmtされていないファイルがあります:" >&2
	echo "$fmt" >&2
	exit 1
fi
for s in $steps; do
	imp=$(go -C "$root/solution/$s" tool goimports -local "$module" -l .)
	if [ -n "$imp" ]; then
		echo "goimportsされていないファイルがあります($s):" >&2
		echo "$imp" >&2
		exit 1
	fi
done

echo "==> solution: go test -race"
# リークを直す前のstep02〜step05は、リークを検出してテストが失敗するのが正しい。
for s in step01 step06 step07 step08; do
	go -C "$root/solution/$s" test -race -count=1 ./...
done
go -C "$root/solution/step04" test -race -count=1 ./internal/...
check_fail() {
	local step=$1 msg=$2 out
	out=$(go -C "$root/solution/$step" test -race -count=1 ./search 2>&1) && {
		echo "$step: リークを検出してテストが失敗するはずが成功しました" >&2
		exit 1
	}
	if ! grep -q "$msg" <<<"$out"; then
		echo "$step: 想定した失敗ではありません" >&2
		echo "$out" >&2
		exit 1
	fi
}
check_fail step02 "ゴールーチンがリークしています"
for s in step03 step04 step05; do
	check_fail $s "found .*unexpected goroutines"
done

echo "==> skeleton: build / vet"
for s in $steps; do
	go -C "$root/skeleton/$s" vet ./...
done

echo "==> skeleton + 解答 = solution の検証"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
for s in $steps; do
	echo "--- $s"
	work="$tmp/$s"
	cp -r "$root/skeleton/$s" "$work"
	for f in $(answers "$s"); do
		cp "$root/solution/$s/$f" "$work/$f"
	done
	diff -r --exclude=README.md "$work" "$root/solution/$s"
done

echo "==> サーバ: goroutineleakプロファイルの結果"
# ポート6060のサーバを起動し、プロファイルの先頭行(total N)を取得する。
leak_total() {
	local step=$1 pkg=$2 profile=$3 bin pid out
	bin="$tmp/server-$step-$(basename "$pkg")"
	go -C "$root/solution/$step" build -o "$bin" "$pkg"
	"$bin" >/dev/null 2>&1 &
	pid=$!
	sleep 3
	out=$(curl -sf "http://localhost:6060/debug/pprof/$profile?debug=1" | head -n 1)
	kill "$pid"
	wait "$pid" 2>/dev/null || true
	sed -E 's/.*total ([0-9]+).*/\1/' <<<"$out"
}
# step02のmonitorのexpvarで公開したゴールーチン数を、2回取得して増えていることを確かめる
goroutines_var() {
	local bin pid a b
	bin="$tmp/server-monitor"
	go -C "$root/solution/step02" build -o "$bin" ./cmd/monitor
	"$bin" >/dev/null 2>&1 &
	pid=$!
	sleep 2
	a=$(curl -sf http://localhost:6060/debug/vars | sed -nE 's/.*"goroutines": ([0-9]+).*/\1/p')
	sleep 2
	b=$(curl -sf http://localhost:6060/debug/vars | sed -nE 's/.*"goroutines": ([0-9]+).*/\1/p')
	kill "$pid"
	wait "$pid" 2>/dev/null || true
	echo $((b - a))
}
# step02のmonitorは、ゴールーチン数が増え続ける
a=$(goroutines_var)
echo "step02 cmd/monitor: $a"
[ "$a" -gt 0 ]
# リークを直す前(step05)はリークが報告される
n=$(leak_total step05 ./cmd/server goroutineleak)
echo "step05 cmd/server: $n"
[ "$n" -gt 0 ]
# 直したあと(step06, step07)は報告されない
for s in step06 step07; do
	n=$(leak_total $s ./cmd/server goroutineleak)
	echo "$s cmd/server: $n"
	[ "$n" -eq 0 ]
done
# 検出できないリーク(step08)は、goroutineleakでは0だが、goroutineは増えていく
for p in ./cmd/global ./cmd/io; do
	n=$(leak_total step08 $p goroutineleak)
	g=$(leak_total step08 $p goroutine)
	echo "step08 $p: goroutineleak=$n goroutine=$g"
	[ "$n" -eq 0 ]
	[ "$g" -gt 10 ]
done

echo "==> README: skeletonとsolutionの本文が一致するか(フッターのリンクを除く)"
body() { awk '{ lines[NR] = $0 } /^---$/ { last = NR } END { for (i = 1; i < last; i++) print lines[i] }' "$1"; }
for s in $steps step09; do
	diff <(body "$root/skeleton/$s/README.md") <(body "$root/solution/$s/README.md")
done

echo "==> README: 文体規則(太字・「〜とは」見出しがないこと)"
if grep -rn '\*\*' --include='README.md' "$root"; then
	echo "太字(**)が使われています" >&2
	exit 1
fi
if grep -rnE '^#+ .*とは$' --include='README.md' "$root"; then
	echo "「〜とは」見出しが使われています" >&2
	exit 1
fi

echo "OK: すべての検証を通過しました"
