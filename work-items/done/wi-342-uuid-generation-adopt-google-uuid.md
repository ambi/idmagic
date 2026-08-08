---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-08
depends_on: []
---

# 自前 UUIDv4 生成 (`spec.NewUUIDv4`) を `google/uuid` に置き換える

## Motivation

`backend/shared/spec/uuid.go` の `NewUUIDv4` は `crypto/rand` から16バイトを読み、
version/variant ビットを設定する自前実装。`crypto/rand`(CSPRNG)を使っており、
`math/rand` や時刻シードのような典型的な落とし穴は踏んでおらず、現状のコードに
セキュリティ上のバグはない。

一方で `github.com/google/uuid` はすでに `go.sum` 上の間接依存として存在しており
(直接依存への格上げに追加コストはほぼない)、以下の点で自前実装より優位がある。

- 多くの目に晒され継続的に保守されているコードであること自体の安心感。
- `Parse` / `Validate` など、外部入力の UUID を検証する箇所で車輪の再発明を避けられる。
- 将来 `NewV7`(時系列ソート可能な UUID)へ移行する際、同一ライブラリ内で完結する。

本 wi は `NewUUIDv4` の実装を `google/uuid` ベースに差し替えるのみで、生成される
UUID のフォーマット(v4, RFC 4122)や `spec.NewUUIDv4() (string, error)` という
シグネチャは変えない。外部から観測可能な振る舞いの変更はないため、SCL の変更は
不要。

## Scope

- `backend/shared/spec/uuid.go`: `NewUUIDv4` の実装を `google/uuid.NewRandom()` ベースに置き換える。
- `backend/go.mod` / `go.sum`: `github.com/google/uuid` を直接依存に格上げする。
- 呼び出し側(`spec.NewUUIDv4()` を使う約40箇所、oauth2/tenancy/provisioning/idmanagement/sharedsignals 等の各 bounded context)は、シグネチャを変えない限り変更不要。念のため全呼び出し箇所のビルド・テストを通す。

## Out of Scope

- `spec/scl.yaml` の変更(観測可能な振る舞いの変更なし)。
- UUIDv7 など他バージョンへの移行(将来の別 wi)。
- 既存データ(すでに発行済みの UUID 文字列)のマイグレーション(フォーマット互換のため不要)。

## Design

- `google/uuid.NewRandom()` は内部で `crypto/rand` を使用し、`NewUUIDv4` と同じ
  RFC 4122 version 4 / variant ビットの UUID を返す。文字列表現も同じ
  `8-4-4-4-12` 小文字16進形式。
- `NewUUIDv4` のシグネチャ `(string, error)` は維持し、内部実装のみ
  `uuid.NewRandom()` の戻り値を `.String()` して返す形にする。呼び出し側は無改修。
- 代替案として検討し不採用: 呼び出し側を `uuid.UUID` 型に置き換える全面改修は
  約40箇所への影響が大きく、本 wi の動機(ライブラリ置き換え)に対して過剰。
  シグネチャを維持した最小差分を採用する。

## Plan

1. `backend/go.mod` に `github.com/google/uuid` を直接依存として追加(`go mod tidy`)。
2. `backend/shared/spec/uuid.go` の実装を `uuid.NewRandom()` ベースに置き換え。
3. 既存の `NewUUIDv4` 単体テスト(v4 フォーマット・variant ビットの検証)がそのまま
   通ることを確認。
4. `just verify` で全体のビルド・テストを実行し、約40箇所の呼び出し元に回帰がないことを確認。

## Tasks

- [x] T001 [App] `github.com/google/uuid` を直接依存に追加する。→ 既に go.mod の直接依存(`shared/storage/db_postgres`, `authentication/federation/handlers_http`, `saml/handlers_http` で使用済み)だったため追加作業不要。
- [x] T002 [App] `backend/shared/spec/uuid.go` の `NewUUIDv4` を `google/uuid` ベースに置き換える。
- [x] T003 [Verify] 既存の UUID 関連テストと `just verify` を通す。既存の `TestNewUUIDv4Format`(`wi129_coverage_test.go`)がフォーマット(v4 version/variant nibble, 一意性)を検証しており、新たな RED テストの追加は不要と判断(振る舞い変更なし、既存回帰テストで十分)。

## Verification

- `just verify`
- `go test ./backend/shared/spec/...`

## Risk Notes

シグネチャ非互換な変更ではなく、内部実装のみの置き換えのため影響範囲は限定的。
リスクは low。生成される UUID のフォーマット(v4, RFC 4122)が変わらないことを
テストで確認する。

## Completion

- **Completed At**: 2026-08-08
- **Summary**:
  `backend/shared/spec/uuid.go` の `NewUUIDv4` を、`crypto/rand` 直叩きの自前実装から
  `github.com/google/uuid` の `uuid.NewRandom().String()` ベースに置き換えた。
  `github.com/google/uuid` はすでに他箇所(`shared/storage/db_postgres`,
  `authentication/federation/handlers_http`, `saml/handlers_http`)で使われている
  go.mod 直接依存だったため、依存追加作業は不要だった。シグネチャ
  `(string, error)` および生成される UUID のフォーマット(v4, RFC 4122)は不変。
  `spec.NewUUIDv4()` を呼ぶ約40箇所は無改修で動作する。
- **Verification Results**:
  - `just test-go-package ./backend/shared/spec/...` - passed
  - `just verify-go`(golangci-lint + `go test -race ./...`) - passed(0 issues, 全パッケージ green)
