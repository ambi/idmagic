---
status: accepted
authors: [tn]
created_at: 2026-08-08
---

# ADR-156: 管理 API の版付けはパス版、外部契約の境界は API access token 到達可能性で判定する

## コンテキスト

[[wi-297-management-api-versioning-and-deprecation-policy]] の Motivation の通り、管理 API / SCIM API は
API access token ([[ADR-137-unify-api-access-token-wire-format]]) で外部から叩かれる外部契約だが、
版管理・非推奨化の方針が存在しない。既存 interface (SCL 全 context 合計 324 件) を棚卸しした結果、
`/api/admin/*` (201 件) のうち 116 件、`/api/account/*` (31 件) のうち 8 件は `TenantAdministrator` /
`SystemAdministrator` / `AuthenticatedSelf` など User principal のセッション認可のみで保護され、
`ManagementApiClient*` / `SelfApiClient*` / `Scim*` policy の対になる API access token 経路を持たない
(`principal: User`、`context.authenticated` によるセッション認可)。これらは実質的に管理コンソール /
account ポータル SPA と一体の画面であり、既存の browser API (`GetBrowserTransaction` 等) と同じ性質を持つ。
「何を外部契約として版管理・非推奨化の対象にするか」の境界を機械的な基準で決める必要がある。

## 決定

- **互換の定義**: 後方互換 = フィールド追加・任意パラメータ追加・新エンドポイント。破壊的変更 =
  フィールド削除・改名・型変更・任意から必須への変更・エラーコード変更・既定値変更。
  `interfaces.System.BackendErrorResponse` が返すエラーコード集合も契約に含める。
- **版の付け方**: パス版 (`/api/admin/v1/...`, `/api/account/v1/...`)。現行の無版パスは v1 の
  エイリアスとして維持する。SCIM は仕様準拠の `/scim/v2/` を維持し、IdMagic の版管理の対象外とする。
- **同時サポート版数と非推奨期間**: 同時 2 版、非推奨表明から削除まで最低 12 か月。
- **告知方法**: `Deprecation` / `Sunset` HTTP ヘッダ + OpenAPI `deprecated: true` + リリースノート。
- **外部契約の境界 (stability)**: interface に `stability: stable | beta | internal` を導入する。
  ある interface が `stable` となるのは次のいずれかを満たすときに限る。
  1. `access.policies` に `ManagementApiClient*` / `SelfApiClient*` / `Scim*` を含む
     (API access token で到達可能、ADR-137)。
  2. OAuth2/OIDC・SAML・WS-Federation・SCIM・SharedSignals (SSF) など標準仕様が定義する
     protocol endpoint である (版管理は discovery が正本、[[ADR-011-discovery-as-derived-artifact]])。
  3. 認証を要さない公開資産・運用 endpoint である (branding/icon 配信、liveness/readiness/startup
     probe、metrics scrape)。
  上記いずれにも該当せず、User principal のセッション認可のみで到達する interface
  (管理コンソール専用画面、pre-auth ブラウザフロー、no-binding の domain-internal 操作) は `internal` とし、
  外部契約から外す。棚卸しの結果は 324 件中 stable 158 件・internal 166 件。`beta` は現時点で該当なしだが
  将来の新規 endpoint 向けに予約する。
  `System` に「`stable`/`beta` の interface は `stability` を宣言しなければならない」制約を追加する。
- **非推奨情報**: `deprecated_since` / `sunset_at` / `successor` (後継 interface 参照) を任意 field として追加する。

## 却下した代替案

- **ヘッダ版管理 (`API-Version` request header)**: 却下。プロキシ / API gateway 越しでの可視性が低く、
  Okta・Entra ら競合が採るパス版の方が顧客の自動化にとって発見・診断しやすい。
- **`/api/admin/*` 全件を stable 扱いにする**: 却下。116 件は API access token 経路を持たず
  管理コンソール SPA と一体で変わる。ここを stable にすると Motivation が懸念する
  「外部契約にすると開発が固まる」を自ら招く。
- **同時サポート 3 版 / 無期限**: 却下。維持コストが線形に増える。チーム規模に対し 2 版・12 か月を
  初期値とし、実運用の負荷を見て見直す。
- **非推奨期間の下限を設けない**: 却下。顧客の移行計画が立たない。

## 影響

- `spec/scl.yaml`, `spec/contexts/*.yaml`: 全 interface に `stability` を付与 (324 件)。`System` に
  外部契約制約と `BackendErrorResponse` のコード安定性を追加。scenario 1 件追加。
- Go: 版付きルーティング (現行パスは v1 エイリアス)、`Deprecation`/`Sunset` ヘッダ付与ミドルウェア。
- tooling: `spec/idmagic.openapi.json` 生成に安定性・非推奨・版を反映。`just check-api-compat` を新設し
  CI (`idmagic-ci.yaml`) に組み込む。
- README: 安定性区分・版管理・非推奨スケジュール・互換の定義。
