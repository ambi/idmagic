---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-09-03
change_kind: feature
priority: p2
depends_on: []
affected_spec:
  - { path: docs/contexts/tenancy/scenarios.md, requirement: REQ-TENANCY-011 }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.SetTenantEndpointStyle }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.DisableTenant }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.EnableTenant }
---

# テナントの停止、再開、正規ロケーション切替を対話セッションに限定する

## Motivation

制御面の `SetTenantEndpointStyle`、`DisableTenant`、`EnableTenant` は、いずれも `tenants:write` を持つ API アクセストークンから呼び出せる。

`SetTenantEndpointStyle` は issuer、WebAuthn RP ID、Cookie の適用範囲を変更し、発行済みトークン、既存パスキー、進行中のセッション、依存するクライアント設定へ同時に影響する。

`DisableTenant` は対象テナントの認証とプロトコル経路を停止し、`EnableTenant` は停止中のテナントを再公開する。

これらは正規のシステム運用者だけが実行できるが、API アクセストークンは漏えいまたは誤った自動化によって、人間が対象テナントと影響を確認しないまま同じ操作を繰り返せる。

現在の `docs/authorization.md` は、既存スコープに対応しない操作と権限昇格を作る操作だけを対話セッション限定の理由として挙げているため、テナント全体の認証可用性または正規ロケーションを変更する操作を分類できない。

## Scope

- `docs/authorization.md` に、テナント全体の認証可用性または正規ロケーションを変更する制御面操作を対話セッション限定とする規則を追加する。
- `SetTenantEndpointStyle`、`DisableTenant`、`EnableTenant` の `x-api-token-scopes` を `interactive_session` に変更する。
- `REQ-TENANCY-011` に正規ロケーション切替の資格情報境界を追加する。
- テナントの停止と再開について、通常テナントの状態遷移と資格情報境界を表す新しい規範シナリオを追加する。
- `docs/contexts/tenancy/decisions.md` を、`tenants:*` が届く操作と対話セッションに限定される操作を区別できる説明へ更新する。
- 管理コンソールから3操作を実行できることと、API アクセストークンでは拒否されることを同じ受け入れ境界で確認する。
- 公開契約の互換性変更としてアップグレードノートを作成する。

## Out of Scope

- 制御面操作への再認証またはステップアップ。
  本作業項目は資格情報の種類を対話セッションへ限定するが、セッションの認証時刻や認証強度は変更しない。
- `CreateTenant`、`UpdateTenant`、`UpdateTenantQuota` を対話セッション限定にすること。
  これらはテナントの払い出し、属性更新、上限調整の自動化を支えるため、`tenants:write` から到達できる状態を維持する。
- `tenants:disable` などの新しいスコープを追加して、停止と再開の自動化を許可すること。
- API アクセストークンの有効期間、失効、発行時の承認を変更すること。
- テナント横断ヘルスの認可条件。
  [[wi-460-cross-tenant-health-control-plane-membership]] が扱う。
- テナント横断画面の配置。
  [[wi-462-control-plane-console-single-entry]] が扱う。

## Design

### 対話セッション限定の判定基準

**テナント全体の認証可用性、または issuer、WebAuthn RP ID、Cookie の適用範囲を決める正規ロケーションを変更する制御面操作は、対話セッション限定とする。**

この基準は「破壊的」という評価語ではなく、変更する製品状態を列挙するため、新しい制御面操作にも同じ分類を適用できる。

現在該当する操作は次の3つである。

| 操作 | 変更する状態 | 対話セッションに限定する理由 |
| --- | --- | --- |
| `SetTenantEndpointStyle` | issuer、WebAuthn RP ID、Cookie の適用範囲 | 既存の資格情報と依存クライアントの前提を一度に変更するため |
| `DisableTenant` | テナント全体の認証可用性 | 対象テナントの利用者を認証経路から締め出すため |
| `EnableTenant` | テナント全体の認証可用性 | 停止理由が解消したことを人間が確認してから再公開する必要があるため |

停止だけを限定して再開を自動化可能なままにすると、障害対応やセキュリティ対応で停止したテナントを、残存するトークンまたは誤った自動化が再公開できる。

そのため `DisableTenant` と `EnableTenant` は同じ資格情報境界に置く。

### 規範シナリオの配置

`REQ-TENANCY-003` は `default` テナントの起動時作成と無効化禁止を扱うため、通常テナントの停止、再開、資格情報境界を追加しない。

`SetTenantEndpointStyle` は既存の `REQ-TENANCY-011` に API アクセストークン拒否の `ALT` を追加する。

停止と再開は SystemAdministrator を主体とする新しいシナリオにし、通常テナントが `Active` と `Disabled` の間を遷移する成功経路、`default` テナントの拒否、API アクセストークンの拒否を一緒に記述する。

### 実行時の強制

`interactive_session` は API アクセストークンへ付与できるスコープではない。

`requireAdminApiTokenScope` は TypeSpec から生成した操作メタデータを参照し、`interactive_session` を宣言した操作への API アクセストークンを `insufficient_scope` で拒否する。

ブラウザーの管理コンソールが提示する通常の OAuth アクセストークンまたはログインセッションは API アクセストークンではないため、この判定を通過した後も既存の `system_admin` と制御面テナントの認可を受ける。

受け入れテストは宣言の差分だけを検査せず、同じ操作について API アクセストークンが拒否され、対話セッションが成功することを HTTP 境界で確認する。

### 却下した選択肢

- **`tenants:disable` などのスコープを追加する。** 停止と再開を自動化する利用事例が示されておらず、公開スコープ語彙を先に増やすと、後から削除する互換性費用が生じる。
- **`tenants:write` のままにする。** 既存の決定は到達可能であることを記録しているだけで、テナント全体の認証可用性と正規ロケーションを無人操作へ開く判断をしていない。
- **管理コンソールの送信元だけを許可する。** クライアント名や画面を認可境界にすると、別の正規クライアントから同じ対話セッションを使う経路を説明できないため、資格情報の種類で判定する。
- **すべての `tenants:*` を対話セッション限定にする。** テナントの在庫参照、払い出し、属性更新、上限調整には自動化の用途があり、今回定める状態分類にも該当しない。

## Plan

1. `docs/authorization.md` と `docs/contexts/tenancy/decisions.md` に分類規則と対象操作を記述する。
2. `REQ-TENANCY-011` を更新し、停止と再開の新しい規範シナリオを追加する。
3. API アクセストークンが3操作に到達でき、対話セッションでも成功する現在の挙動を HTTP 境界で観測し、資格情報境界の受け入れ RED を確認する。
4. 3操作の `x-api-token-scopes` を `interactive_session` に変更し、実行時契約を再生成する。
5. API アクセストークンの拒否と対話セッションの成功を確認する。
6. 既存の自動化が使えなくなることと、管理コンソールから実行する代替をアップグレードノートへ記載する。
7. 互換性検査と標準検査を通す。

## Tasks

- [ ] T001 [Spec] 対話セッション限定の分類規則を `docs/authorization.md` と Tenancy の決定へ追加する。
- [ ] T002 [Spec] `REQ-TENANCY-011` を更新し、停止と再開の新しい規範シナリオを追加する。
- [ ] T003 [Acceptance] API アクセストークンと対話セッションの現在の挙動を同じ3操作で観測し、RED を確認する。
- [ ] T004 [Spec] 3操作の `x-api-token-scopes` を `interactive_session` に変更し、TypeSpec 文書コメントを更新する。
- [ ] T005 [App] 実行時契約を再生成し、API アクセストークンが `insufficient_scope` で拒否されることを確認する。
- [ ] T006 [Acceptance] 管理コンソールの対話セッションでは3操作が成功し続けることを確認する。
- [ ] T007 [Docs] 互換性変更と代替操作をアップグレードノートへ記載する。
- [ ] T008 [Verify] 互換性検査と標準検査を通す。

## Verification

- `mise run check-admin-scopes`
- `mise run check-api-compat`
- `mise run test-go-race`
- `mise run test-ui-e2e`
- `mise run check-spec`
- `mise run check-ids`
- `mise run check-work-items`
- `mise run verify`

## Risk Notes

リスクは high とする。

公開契約から既存の到達能力を取り除くため、`tenants:write` の API アクセストークンで3操作を自動化している外部利用者がいれば、その自動化は動かなくなる。

リポジトリ内に該当する呼び出し元がないことは、外部利用者がいないことの証拠にならないため、アップグレードノートに拒否される操作と代替を明記する。

TypeSpec の宣言だけを変更して実行時の強制が変わらない誤りを防ぐため、API アクセストークンによる実呼出しの失敗を受け入れ証拠にする。

対話セッション側まで拒否すると復旧手段を失うため、同じテスト群で許可側も確認する。

`reversibility` は irreversible とする。

宣言と実装は従来の到達能力へ戻せるが、停止と再開のために割り当てた新しい規範 ID は再利用しない。
