---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# HTTP ハンドラをコンテキスト別パッケージへ分割し共有 core を抽出する

http アダプタを per-context パッケージへ分割した。共有基盤 (依存集約 Deps・
HealthInfo・テナント解決 middleware・横断ヘルパ) を `internal/platform/http/core`
に集約し、各コンテキストは `internal/<context>/adapters/http` で自身のハンドラを
所有する。`internal/platform/http` は各コンテキストの RegisterRoutes を束ねる
router (+ /health) に縮約した。依存方向は core ← 各 context http ← router の一方向。
ハンドラは承認方針どおり「core.Deps を埋め込む薄いラッパ (type Deps struct{ *core.Deps })」
のメソッドのまま移し、ルーティング登録とハンドラ本体はほぼ無改変とした
(受け渡し方式の判断記録)。挙動・パス・メソッド・middleware は不変。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
