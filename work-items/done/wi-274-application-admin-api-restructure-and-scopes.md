---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-23
depends_on: [wi-273-unified-api-token-foundation]
---

# アプリケーション周り管理 API のパス構成を見直し、application/protocol 系の API scope を定義する

## Motivation

管理用 API を scope 付きで公開する（[wi-273] 統一 API トークン基盤）にあたり scope 語彙を洗い出したところ、アプリケーション周りだけ構成が入り組んでいて、そのままでは綺麗な scope 分割ができないことが判明した。

- `Application`（application context）は `kind: federated | weblink | service` の上位 aggregate で、OIDC / SAML / WS-Fed を `ProtocolBinding` として束ねる。binding 操作は `/api/admin/applications/:id/oidc|saml|wsfed`。
- 一方で低レベルの独立プロトコルエンドポイントが別に存在する: `/api/admin/clients`（OAuth2 client）、`/api/admin/saml/service-providers`、`/api/admin/wsfed/relying-parties`、`/api/admin/wsfed/entra-federation`。加えて `mcp-resource-servers`、`authorization-detail-types`、`consents`、outbound `provisioning`（`/api/admin/provisioning/connections` と `/api/admin/applications/:id/provisioning/*` に二分）。
- application aggregate と各 protocol 生リソースの責務境界・パス構成が一貫しておらず、公開 API として scope を切る前に整理（必要ならリファクタリング）した方がよい。

本 WI で application 周りのパス構成を評価し、必要ならリファクタリングした上で、application/protocol 系の API scope を確定し `ApiTokenScope` enum と各 context の authorization に反映する。[wi-273] は安定リソース（users/groups/agents/sessions/consents/lifecycle-workflows/tenants/settings/signing-keys/audit/scim）の scope のみ先行実装済みで、本 WI の scope はそこに追記する。

## Scope

- 調査・設計判断（コード + SCL）: application context と oauth2 / saml / wsfederation / provisioning の管理エンドポイントのパス構成評価。
- `spec/scl.yaml` の Application / OAuth2 / Saml / WsFederation / Provisioning から ApiTokens への依存。
- `spec/contexts/application.yaml`・`oauth2.yaml`・`saml.yaml`・`ws-federation.yaml`・`provisioning.yaml` の `models` / `authorization` / `interfaces.access` / `scenarios`。
- `spec/contexts/api-tokens.yaml` の `models.ApiTokenScope`（application/protocol 系 scope 値を追記）。
- 上記各 context の `authorization`（`ManagementApiClient` principal + scope 別 policy）と対象 interface の `access.policies`。
- `backend/apitoken/domain` と管理 UI の scope 正準リスト。

## Out of Scope

- 安定リソースの scope と管理 API 公開（[wi-273] および認証カーネル / IdManagement 公開 WI）。
- SCIM inbound（`scim` scope、[wi-273] で確定済み）。
- 管理 API の PAT 認証カーネル（PAT 解決、scope enforcement、CSRF 除外、監査 actor 帰属）。各 context の policy を実行時に結線する横断基盤として後続 WI で扱う。

## Plan

- **決定**: 現行パスは変更しない。Application aggregate とその protocol binding は `applications:*`、独立した protocol/resource API は `oauth-clients:*` / `authorization-detail-types:*` / `mcp-resource-servers:*` / `saml:*` / `wsfed:*`、outbound provisioning は `provisioning:*` とする。application 配下に見える provisioning endpoint も所有 context に従って `provisioning:*` を要求する。
- **read/write**: GET は `*:read`、作成・更新・削除・rotate/test/on-demand/full-resync/resume/retry は `*:write`。既存 `consents:*` は維持し、tenant default sign-in policy は既存 `settings:*` に帰属させる。

- まず「パス構成を変えるべきか」を判断する。候補となる scope 分割方針:
  - **Application 中心 + 独立 protocol は別 scope**: `applications:*` が app aggregate と `/applications/:id/` 配下の binding を、独立エンドポイントは `oauth-clients:*` / `saml:*` / `wsfed:*` が担う。
  - **完全 Application 中心**: 生 protocol エンドポイントも `applications:*` に集約。
  - **完全 protocol 分割**: binding 操作も対応 protocol scope を要求。
- リファクタリングが要る場合は、パス整理を本 WI 内の独立タスク（SCL bindings + ハンドラ + UI + e2e）として先に片付け、その後 scope を定義する。大きくなるならさらに WI 分割してよい。
- provisioning（app 配下 + connections）と `mcp-resource-servers` / `authorization-detail-types` / `consents` の帰属もここで確定する（consents は [wi-273] で単独 `consents:*` を先行確定済み。application 整理で移動が必要になったら再検討）。
- scope 追記は認証カーネル WI（`AuthorizeAdminOrScope` / PAT 解決）に依存する。カーネル未完なら policy 追記のみ先行し、実 enforcement は後続で結線する。

## Tasks

- [x] T001 [Investigate] application/oauth2/saml/wsfed/provisioning の管理パス構成を評価し、パス維持と resource 単位 scope 分割を決定（ADR に記録）。
- [x] T002 [SCL/App] パス構成は変更不要と判断。Application aggregate、raw protocol resource、Provisioning read model の現行所有境界を維持する。
- [x] T003 [SCL] `ApiTokenScope` enum に application/protocol 系 14 scope を追記し、`just yaml-check-scl` で検証。
- [x] T004 [SCL] 各 context に `ManagementApiClient` principal + scope 別 policy を追加し、対象 interface の `access.policies` と受け入れ scenario に付与。
- [x] T005 [Domain/UI] RED: `TestParseApplicationProtocolScopes` を未定義 scope 定数で fail、`ApiTokensTab` の scope 選択肢 test を `applications:read` 不在で fail 確認（model `ApiTokenScope` / application・protocol management scenarios）→ GREEN。
- [x] T006 [Verify] SCL/render、Go race/lint、UI unit/build/E2E、traceability/tooling を検証。実クライアントの scope enforcement は横断認証カーネル未実装のため Out of Scope として未実施。

## Verification

- `just yaml-check` / `just scl-render`（SCL 妥当性・派生物、application/protocol scope が OpenAPI に反映）。
- `just verify-go` / `just verify-ui` / `just test-ui-e2e`。
- `just traceability-strict` / `just test-tools` / `just typecheck-tools`。
- 実クライアントの application/protocol scope 許可・scope 外 403 は、横断認証カーネル結線後の後続 WI で検証する。

## Risk Notes

- **パス構成変更（中）**: 管理 UI と既存クライアントに影響。未リリースのため破壊的変更は許容だが、UI の追随を同一 WI で担保する。
- 分割方針の誤りは後から scope 体系の変更を招くため、T001 の判断を ADR で残す。
- **実 enforcement の保留（中）**: 管理 API の PAT 解決は監査 actor 帰属と CSRF 除外を含む横断認証カーネルを要する。本 WI では SCL policy と発行可能 scope を先行し、handler の `RequireAdmin` は変更しない。
- **fuzz/property test 判断**: 不採用。追加する入力は固定 enum の scope 値であり、再帰文法や組み合わせ爆発を持たない。domain の表駆動 parse test と UI 選択肢 test で正準リスト同期を検証する。

## Completion

- **Completed At**: 2026-07-23
- **Summary**:
  Application aggregate と独立 protocol/resource の所有境界に沿う 14 scope を定義し、
  5 context の認可 policy、Go domain enum、管理 UI、SCL 派生物を同期した。
  管理 API path は変更不要と判断した。
- **Affected Guarantees State**:
  - Application binding は `applications:*`、独立 resource は resource 別 scope、Provisioning は path にかかわらず `provisioning:*` に対応する。
  - read scope は参照操作、write scope は変更・action 操作にだけ対応し、tenant 不一致は SCL policy で拒否する。
  - 管理 API の runtime PAT scope enforcement は横断認証カーネルへ保留し、既存 session administrator の `RequireAdmin` 経路は変更しない。
- **Verification Results**:
  - `just yaml-check-scl` — passed
  - `just scl-render` — passed
  - `just verify-go` — passed
  - `just verify-ui` — passed（424 UI unit tests）
  - `just test-ui-e2e` — passed（UI action 15、smoke 3、recovery 1、golden path 2）
  - `just traceability-strict` — passed
  - `just test-tools` — passed
  - `just typecheck-tools` — passed
- **Evidence**:
  `TestParseApplicationProtocolScopes` と `ApiTokensTab` の scope 選択肢 test を RED から GREEN にし、
  14 scope の domain enum・許可集合・UI 正準リストの同期を確認した。
- **Known Repository Baseline**:
  当時の repository-wide gate は、既存 wi-216 の Completion 欠落と
  `AdminUserEditPage.tsx` の complexity debt により非ゼロ終了だった。
