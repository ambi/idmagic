---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-07-04
---

# ビルド版数（git SHA / version / build date）を成果物に刻み /version と telemetry へ露出する

## Motivation
現状バージョンはコード中の文字列リテラルで、observability 初期化に
`"0.3.0"` をハードコードしているだけで、git SHA・ビルド日時・ダーティ状態を
持たない。`/version` 相当のエンドポイントも無い。本番で「今どの版が動いているか」を
Pod・トレース・監査から確定できないと、ロールアウト事故の切り分け、脆弱性の
影響版特定（WI-100 の SBOM と対応付け）、サポート応答ができない。

Kubernetes・Go の release engineering は ldflags でビルド時に version metadata を
埋め込み、`kubectl version` / `component --version` / `runtime/debug.BuildInfo` で
露出するのを標準とする。idmagic も単一の version 情報源を持ち、それを
`/version` エンドポイント、起動ログ、OpenTelemetry の resource attribute、
監査イベントの発行元識別へ一貫して流すべきである。

## Scope
- **decision**:
  - 新規 ADR: version 情報源（ldflags か runtime/debug.BuildInfo か）、SemVer 付与ルール、 `/version` の公開範囲（認証要否・露出フィールド）を定義する。イメージ tag / SBOM / cosign 署名との対応関係を記す。
- **scl**:
  - System context に BuildInfo / ReportVersion の objective を追加する。
  - version が持つフィールド（version / git_commit / build_date / go_version）を定義する。
- **go**:
  - ldflags で version / commit / date を注入する変数を 1 箇所に集約し、 未設定時は runtime/debug.BuildInfo（vcs.revision / vcs.time / vcs.modified）へフォールバックする。
  - observability 初期化のハードコード `"0.3.0"` を集約した version 情報に置き換え、 OTel resource の service.version にする。
  - `/version` エンドポイントを追加する。公開範囲は ADR に従い、機密（内部 path 等）を含めない。
  - 起動ログに version / commit / build date を出す。
- **documentation**:
  - Dockerfile / justfile のビルドで ldflags を渡すよう更新し、README にリリース時の版数付与手順を書く。

## Out of Scope
- リリース自動化・タグ発番ワークフロー（CI リリースパイプライン全体）。
- SBOM 生成・cosign 署名・SLSA provenance（WI-100 が扱う）。
- UI 側のビルド版数表示。

## Verification
- `go test -race ./...` (in: idmagic)
- `go build ./...` (in: idmagic)
- 手動: ldflags 付きでビルドしたバイナリの `/version` が git SHA と build date を返し、 起動ログ・OTel resource と一致することを確認する。
- 手動: ldflags 無し（go run 相当）でも BuildInfo フォールバックで commit / dirty が 取得できることを確認する。

## Risk Notes
version 露出は情報漏えいリスクが低いが、内部ホスト名や絶対 path を含めない。
未タグビルドでの表記（0.0.0-dev+<sha>）を決めておかないと監視で紛れる。

## Completion
- **Completed At**: 2026-07-04
- **Summary**:
  Go API のビルド時に ldflags を用いて動的にバージョン（SemVer）、Git コミットハッシュ、ビルド日付を注入する仕組みを実装した。
  ldflags が渡されないローカル実行などの場合は、`runtime/debug.BuildInfo` を解析する VCS フォールバック機能を実装している。
  なお、外部への詳細なバージョン情報漏洩（既知の脆弱性特定の起点となるリスク）を防ぐため、 unauthenticated な `/version` エンドポイントの公開は避け、非公開の起動ログおよび内部 OTel service.version のみにこれら情報を適用するようにした。
  `justfile` と `Dockerfile` のビルドプロセスを、これら ldflags を渡すように更新し、`README.md` に仕様とビルド・検証手順を記載した。
- **Verification Results**:
  - just verify
  - go test ./internal/shared/version
