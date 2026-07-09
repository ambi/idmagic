---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-06-22
---

# HTTP ハンドラをコンテキスト別パッケージへ分割し共有 core を抽出する

## Motivation
ADR-047 で Layer 4 アダプタをコンテキスト所有へ寄せ、横断インフラを
`internal/platform/` に集約し、永続化はリソース別ファイルへ carve した。
残るのは http アダプタである。現状 `internal/platform/http` は57ファイル・
約9,800行が**単一 `Deps` 構造体のメソッド**として同居する単一パッケージで、
1機能のハンドラを読むのに大きなパッケージ全体が文脈に乗る。

本 WI は RA §3.6 の格子を http まで完成させ、ハンドラを所有コンテキスト
(`tenancy` / `authentication` / `oauth2`) の `adapters/http` パッケージへ分割する。
ユーザ承認済みの方針は「共有 core への分離」:
`Deps` と真に横断のヘルパ (errors / tenant middleware / client auth /
認証 sub 抽出 / HealthInfo) を `internal/platform/http/core` に集約し、
各ハンドラは所有コンテキストのパッケージで `core.Deps` を受け取る関数へ変換、
`internal/platform/http`(router) が各コンテキストの `RegisterRoutes` を集約する。

## Scope
- **decision**:
  - 共有 core の責務境界を確定する。core に置くのは Deps・HealthInfo・ tenant 解決 middleware・汎用 OAuth エラー出力・client 認証・認証 sub 抽出 など複数コンテキストが使う最小限。context 固有ヘルパは各 context へ移す。
  - 循環回避方針: core(依存なし) ← <context>/adapters/http(core を import) ← platform/http(router, 両方を import)。router 以外から context http を import しない。
- **go**:
  - `internal/platform/http/core` を作り、Deps と横断ヘルパを移して必要な識別子を export する。
  - `internal/oauth2/adapters/http` を作り、token/introspect/revoke/authorize/ userinfo/par/device/discovery/register/client_auth/end_session/consent と admin_client/admin_consent/admin_key/admin_audit_event/admin_role_policy ハンドラを移し、`RegisterRoutes(g, *core.Deps)` を定義する。
  - `internal/authentication/adapters/http` を作り、account_*・authflow(login/ totp/change_password/forgot/reset/email_change/step_up)・admin_user/ admin_group・auth_event_bucket ハンドラを移す。
  - `internal/tenancy/adapters/http` を作り、admin_tenant/admin_settings/ admin_user_attribute_schema と tenant 経路を移す。
  - `internal/platform/http` の `Register` を各 context の `RegisterRoutes` を 集約する router へ縮約する。ルーティング (パス・メソッド・middleware) は不変。
- **test**:
  - 約25個の http テスト (現状 `Deps` を直接構築し未公開ヘルパを白box呼び出し) を 移行する。ルーティング経由の結合テストは router パッケージで `core.Deps` を 構築して維持し、未公開ヘルパ直叩きの白box テストは所有 context パッケージへ移す。 `routes_e2e_test.go` で全エンドポイント網羅が維持されることを担保する。
- **documentation**:
  - README のディレクトリ構成節を per-context http に更新し、ADR-047 の 「http は後続」記述を完了状態に改める。

## Out of Scope
- HTTP の挙動・ルーティング・認証/認可ロジックの変更。本 WI は純粋な構造分割で ある (ハンドラのメソッド→関数化とパッケージ移動のみ)。
- per-context Deps への分解 (コンテキスト別の依存 struct 化)。本 WI は共有 `core.Deps` を維持する。さらに細かく分けるかは将来判断。
- persistence の per-context パッケージ化 (ADR-047 で平坦集約と決定済み)。

## Verification
- `go build ./...` (in: idmagic)
- `go test ./...` (in: idmagic)
  - reason: ルーティング不変と各ハンドラの挙動が分割前後で一致することを担保する。
- `golangci-lint run ./...` (in: idmagic)
- `bun run yaml-check:all` (in: tools)
- 手動: `go run ./cmd/idmagic` (PERSISTENCE=memory) を起動し、`/health` が 200、 主要フロー (authorize → login → consent → token → userinfo) が通ることを確認。

## Risk Notes
(1) 回帰: 57ソース + 約25テストに及ぶ大規模移動。God-struct のメソッド→関数化と
    ヘルパの横断/固有の仕分けを誤るとコンパイルは通ってもルーティングや認可が
    ずれうる。`routes_e2e_test.go` を分割前の網羅状態で固定してから着手する。
(2) 循環 import: router 以外から context http を参照すると循環する。core に
    置く識別子を最小化し、context 固有ヘルパを core に巻き込まないこと。
(3) テスト移行: 未公開ヘルパ直叩きの白box テストは所有 context へ移すか、
    公開 API 経由へ書き換える。移行漏れでカバレッジが落ちないよう確認する。
(4) 段階性: context 単位 (oauth2 → tenancy → authentication、依存の浅い順) に
    小さくコミットし、各コミットを build/test グリーンに保つ。

## Completion
- **Completed At**: 2026-06-23
- **Summary**:
  http アダプタを per-context パッケージへ分割した。共有基盤 (依存集約 Deps・
  HealthInfo・テナント解決 middleware・横断ヘルパ) を `internal/platform/http/core`
  に集約し、各コンテキストは `internal/<context>/adapters/http` で自身のハンドラを
  所有する。`internal/platform/http` は各コンテキストの RegisterRoutes を束ねる
  router (+ /health) に縮約した。依存方向は core ← 各 context http ← router の一方向。
  ハンドラは承認方針どおり「core.Deps を埋め込む薄いラッパ (type Deps struct{ *core.Deps })」
  のメソッドのまま移し、ルーティング登録とハンドラ本体はほぼ無改変とした
  (受け渡し方式の判断記録)。挙動・パス・メソッド・middleware は不変。
- **Verification Results**:
  - `go build ./...` (in: idmagic)
    - result: passed
  - `go test ./...` (in: idmagic)
    - result: passed
  - `golangci-lint run ./internal/...` (in: idmagic)
    - result: passed
  - `bun run yaml-check:all` (in: tools)
    - result: passed
  - 手動: go run ./cmd/idmagic (PERSISTENCE=memory) 起動で /health 200・discovery の endpoint がテナント込みで正常。authorize → login → consent → token → userinfo の 全フローは routes_e2e_test.go (Register 経由・demo client/alice seed) が網羅し passed。
- **Affected Guarantees State**:
  - 挙動不変: 全エンドポイントのパス・メソッド・middleware は router の RegisterRoutes 委譲で同一。routes_e2e_test.go の全フロー網羅が分割後も passed。後退なし。
  - 非循環依存: core(依存なし) ← 各 context adapters/http ← platform/http(router) の 一方向を維持。context 間の http 依存は無し (go build で担保)。
  - テスト網羅: ハンドラ単位のテストは所有コンテキストまたは router に保持し、移行漏れ なし。go test ./... 全 green。
