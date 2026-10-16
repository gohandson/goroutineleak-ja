#!/bin/bash
# ハンズオン教材の検証スクリプト。
# CI(GitHub Actions)から実行されるが、リポジトリのルートで
# ./.github/scripts/verify.sh として手元でも実行できる。
set -euo pipefail

root=$(cd "$(dirname "$0")/../.." && pwd)
steps="step01 step02 step03 step04 step05 step06"
module=github.com/gohandson/goroutineleak-ja

# 問題(skeleton)に上書きして解答例(solution)と一致させるファイル
answers() {
	case "$1" in
	step01) echo "search/search_test.go" ;;
	step02) echo "search/search_test.go go.mod go.sum" ;;
	step03) echo "cmd/server/main.go" ;;
	step04) echo "search/search.go" ;;
	step05) echo "search/search.go" ;;
	step06) echo "leaks/leaks_test.go" ;;
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
# リークを直す前のstep02とstep03は、goleakがリークを検出してテストが失敗するのが正しい。
for s in step01 step04 step05 step06; do
	go -C "$root/solution/$s" test -race -count=1 ./...
done
for s in step02 step03; do
	out=$(go -C "$root/solution/$s" test -race -count=1 ./search 2>&1) && {
		echo "$s: リークを検出してテストが失敗するはずが成功しました" >&2
		exit 1
	}
	if ! grep -q "found unexpected goroutines" <<<"$out"; then
		echo "$s: goleakによる失敗ではありません" >&2
		echo "$out" >&2
		exit 1
	fi
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
# リークを直す前(step03)はリークが報告される
n=$(leak_total step03 ./cmd/server goroutineleak)
echo "step03 cmd/server: $n"
[ "$n" -gt 0 ]
# 直したあと(step04, step05)は報告されない
for s in step04 step05; do
	n=$(leak_total $s ./cmd/server goroutineleak)
	echo "$s cmd/server: $n"
	[ "$n" -eq 0 ]
done
# 検出できないリーク(step06)は、goroutineleakでは0だが、goroutineは増えていく
for p in ./cmd/global ./cmd/io; do
	n=$(leak_total step06 $p goroutineleak)
	g=$(leak_total step06 $p goroutine)
	echo "step06 $p: goroutineleak=$n goroutine=$g"
	[ "$n" -eq 0 ]
	[ "$g" -gt 10 ]
done

echo "==> README: skeletonとsolutionの本文が一致するか(フッターのリンクを除く)"
body() { awk '{ lines[NR] = $0 } /^---$/ { last = NR } END { for (i = 1; i < last; i++) print lines[i] }' "$1"; }
for s in $steps step07; do
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
