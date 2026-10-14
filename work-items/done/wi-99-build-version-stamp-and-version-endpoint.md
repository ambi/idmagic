---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# ビルド版数（git SHA / version / build date）を成果物に刻み /version と telemetry へ露出する

Go API のビルド時に ldflags を用いて動的にバージョン（SemVer）、Git コミットハッシュ、ビルド日付を注入する仕組みを実装した。
ldflags が渡されないローカル実行などの場合は、`runtime/debug.BuildInfo` を解析する VCS フォールバック機能を実装している。
なお、外部への詳細なバージョン情報漏洩（既知の脆弱性特定の起点となるリスク）を防ぐため、 unauthenticated な `/version` エンドポイントの公開は避け、非公開の起動ログおよび内部 OTel service.version のみにこれら情報を適用するようにした。
`justfile` と `Dockerfile` のビルドプロセスを、これら ldflags を渡すように更新し、`README.md` に仕様とビルド・検証手順を記載した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
