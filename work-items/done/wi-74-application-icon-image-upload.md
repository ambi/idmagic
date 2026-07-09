---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-06-27
---

# アプリケーションアイコンに画像アップロード機能を追加する

## Motivation
[[wi-69-application-catalog-aggregate-and-assignment]] では Application のアイコンを
icon_url の自由入力で持たせた。しかし Okta / Entra ID の管理画面では、管理者が
ロゴ画像を直接アップロードでき、URL を用意する必要がない。自由入力 URL は外部依存・
リンク切れ・任意 URL 埋め込み (SSRF / トラッキング) のリスクもある。

本 WI は Application アイコンを画像アップロードに置き換える。管理者は管理画面から
画像ファイルを選んでアップロードし、IdP がテナント境界内に保存して配信する。
icon_url は内部生成の配信 URL を指す。

## Scope
- **decision**:
  - 新規 ADR: アイコン画像の保存形態 (tenant-scoped object store / DB blob / ファイルシステム)、 受理する形式 (png / jpeg / svg / webp) と最大サイズ、配信経路 (キャッシュ・content-type 固定・実行可能コンテンツ拒否)、icon_url との関係を確定する。
- **scl**:
  - Application に保存済みアイコンの参照 (object key 等) を表現し、icon_url を内部配信 URL に解決する。
  - interface: アイコンのアップロード (multipart) と削除、配信 (GET)。
- **go**:
  - アイコンの保存 adapter (tenant scope) と検証 (magic byte / サイズ / 形式)。
  - 配信ハンドラ (content-type 固定、no-sniff、実行可能扱いの防止)。
- **http**:
  - POST /api/admin/applications/{id}/icon (multipart)、DELETE、GET 配信。
- **ui**:
  - admin: Application 編集でアイコンのドラッグ&ドロップ / ファイル選択アップロードとプレビュー。
  - icon_url 自由入力フィールドはアップロードウィジェットに置き換える。

## Out of Scope
- 画像のリサイズ / 最適化パイプライン。初期は受理した画像をそのまま保存・配信する。
- CDN 配信や署名付き URL。初期は IdP 直配信。
- アプリケーション本体・割当 ([[wi-69-application-catalog-aggregate-and-assignment]])。

## Verification
- `GOCACHE=/tmp/idmagic-cache go test ./...` (in: idmagic)
- `golangci-lint run ./...` (in: idmagic)
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui lint`
- `bun --cwd idmagic/ui build`
- `bun run yaml-check:work-items` (in: tools)
- 手動: 画像をアップロード → ポータル / 管理一覧にアイコンが表示される。非画像・過大サイズが 拒否される。配信レスポンスが content-type 固定・no-sniff であることを確認する。

## Risk Notes
ファイルアップロードは検証不備があると保存型 XSS / コンテンツ偽装の温床になる。
magic byte 検証・content-type 固定・X-Content-Type-Options: nosniff・実行権限の無い保存先を
徹底する。svg を受理する場合はスクリプト除去 (sanitize) するか svg を受理対象から外す。

## Completion
- **Completed At**: 2026-07-02
- **Summary**:
  Application アイコンを URL 自由入力から tenant-scoped な画像アップロードへ置き換えた。
  ADR-073 で PostgreSQL blob 保存、PNG/JPEG/WebP/GIF の magic byte 検証、256 KiB 上限、
  SVG 非対応、nosniff 配信を決定し、SCL に icon_object_key、upload/delete/serve interface、
  安全配信と tenant isolation の invariant を追加した。

  Go 実装では ApplicationIconStore port、memory/PostgreSQL adapter、schema、upload/delete/serve
  handler、ApplicationIconUpdated event を追加した。管理 UI は Application 編集画面の
  アイコン URL 入力をファイル選択 + プレビュー + 削除に置き換え、作成 API から icon_url 入力を
  外した。
- **Verification Results**:
  - `just yaml-check` - passed
    - environment: local
    - result: work item / SCL YAML すべて成功。
  - `just scl-render` - passed
    - environment: local
    - result: idmagic HTML / JSON Schema / OpenAPI 派生物を再生成。
  - `GOCACHE=/tmp/idmagic-cache go test ./...` - passed
    - environment: idmagic
    - result: 全 Go package 成功。
  - `GOCACHE=/tmp/idmagic-cache go test -race ./...` - passed
    - environment: idmagic
    - result: 全 Go package race test 成功。
  - `just build-go` - passed
    - environment: local
    - result: go build ./... 成功。
  - `just verify-ui` - passed
    - environment: local
    - result: UI format check / lint / typecheck / build 成功。
  - `just lint-go` - passed
    - environment: local, outside Codex filesystem sandbox
    - result: golangci-lint run ./... 成功。0 issues。
- **Affected Guarantees State**:
  - guarantee: safe serving
  - state: passed
  - guarantee: input validation
  - state: passed
  - guarantee: tenant isolation
  - state: passed
  - guarantee: no external fetch
  - state: passed
