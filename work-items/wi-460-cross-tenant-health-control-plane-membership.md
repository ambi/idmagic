---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-09-03
change_kind: bugfix
priority: p1
depends_on: []
affected_spec:
  - { path: docs/contexts/signing-keys/scenarios.md, requirement: REQ-SIGNINGKEYS-008 }
  - { path: docs/contexts/signing-keys/scenarios.md, requirement: REQ-SIGNINGKEYS-009 }
  - { path: docs/contexts/data-keys/scenarios.md, requirement: REQ-DATAKEYS-006 }
  - { path: docs/contexts/jobs/scenarios.md, requirement: REQ-JOBS-012 }
  - { path: docs/contexts/jobs/scenarios.md, requirement: REQ-JOBS-013 }
  - { path: spec/contexts/signing-keys/main.tsp, symbol: IdMagic.SigningKeys.Operations.ListTenantKeyHealth }
  - { path: spec/contexts/data-keys/main.tsp, symbol: IdMagic.DataKeys.Operations.ListTenantDataKeyHealth }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.ListJobs }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.GetJob }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.CancelJob }
---

# テナント横断ヘルスの認可を制御面主体の共通判定へ統一する

## Motivation

`docs/authorization.md` は、テナントを越える操作に `system_admin` の有効ロールと `default`（制御面）テナントへの所属を合わせて要求している。

しかし `ListTenantKeyHealth` と `ListTenantDataKeyHealth` は、認証済み User のロールに `system_admin` が含まれることしか確認していない。

そのため、`default` 以外のテナントに属する User が直接または Group 経由で `system_admin` を得ると、そのテナントの管理 API 経路から全テナントの識別子と鍵運用状態を取得できる。

`ListTenantKeyHealth` は `tenant_id`、提供元、`active_kid`、鍵数、到達性を返し、`ListTenantDataKeyHealth` は `tenant_id`、有効版、状態、提供元、到達性を返すため、この欠陥は `docs/authorization.md` の「要求者が権限を持たないテナントの識別子、名前、件数を応答に含めない」という規則に反する。

`REQ-SIGNINGKEYS-009` と `REQ-DATAKEYS-006` も拒否条件を `system_admin` の有無だけで表しており、実装だけでなくシナリオも製品全体のテナント境界より弱い。

同じ制御面主体の判定は Tenancy、Audit、Jobs、SigningKeys、DataKeys の5か所に別々の形で存在し、認証保留、有効状態、所属テナント、有効ロールの扱いが一致していない。

Jobs は横断範囲を作る前に `RequireAdmin` を通すため、TypeSpec と `REQ-JOBS-012` が要求する `system_admin` に加えて `admin` も持たなければ横断一覧、詳細参照、取消しへ到達できない。

## Scope

- `REQ-SIGNINGKEYS-008`、`REQ-SIGNINGKEYS-009`、`REQ-DATAKEYS-006` に、制御面主体の許可条件と `default` 以外のテナントに属する `system_admin` の拒否を記述する。
- `ListTenantKeyHealth` と `ListTenantDataKeyHealth` の TypeSpec 文書コメントを同じ条件へ揃える。
- 認証済みで保留状態ではなく、有効であり、要求先と所属先がともに `default` テナントで、有効ロールに `system_admin` を含む User を返す共通の `RequireControlPlaneUser` を `support_http.Authenticator` に追加する。
- Tenancy の制御面 CRUD、Audit の横断検索、Jobs の横断一覧、詳細参照、取消し、SigningKeys と DataKeys の横断ヘルスを共通判定へ移し、Jobs の余分な `admin` 条件を除く。
- `docs/contexts/signing-keys/decisions.md`、`docs/contexts/data-keys/decisions.md`、`docs/contexts/identity-management/glossary.md` に残る「`system_admin` だけでよい」という説明を、制御面テナント所属と有効ロールを含む定義へ修正する。
- フロントエンドの `requireSystemAccount` も、アカウント文脈の有効ロールと realm の両方を確認する。

## Out of Scope

- `system_admin` を予約ロールにして書き込みを制限すること。
  [[wi-463-reserve-system-admin-role-to-control-plane]] が扱う。
- 監査、ジョブなどのテナント横断画面をシステムコンソールへ移すこと。
  [[wi-462-control-plane-console-single-entry]] が扱う。
- 制御面操作に到達できる資格情報の種類を変更すること。
  [[wi-461-control-plane-credential-boundary]] が扱う。
- 認可を HTTP ハンドラーからユースケース層へ移す一般化。
  [[wi-393-guard-rules-reach-the-usecase-layer]] の検討対象であり、本作業項目では既存の強制境界を統一する。
- 制御面操作への再認証またはステップアップ。

## Design

### 制御面主体の条件

`RequireControlPlaneUser` は次の条件をすべて満たした User だけを返す。

| 条件 | 理由 |
| --- | --- |
| 認証コンテキストが存在し、`AuthenticationPending` ではない | 認証途中の主体を管理操作へ進めないため |
| User が存在し、`IsActive()` である | 無効化済みの主体を拒否するため |
| 解決済みの要求先テナントが `default` である | 別テナントの経路を制御面の入口として扱わないため |
| User の所属テナントが `default` である | ロール名だけでテナント境界を越えないため |
| `EffectiveRoles` に `system_admin` が含まれる | User への直接付与と Group 由来の付与を同じ意味で扱うため |

`ResolveAuthentication` は現在も User と要求先テナントの一致を確認するが、共通判定でも要求先と所属先を明示的に検査する。

呼び出し側が事前条件へ暗黙に依存すると、認証解決の内部変更だけで制御面の境界が弱くなるためである。

### 真偽値ではなく検証済み User を返す

共通処理は `bool` ではなく、有効ロールを反映した検証済み User を返す。

監査イベントの記録や対象操作が主体 ID を必要とする場合に、認証と User 読み込みをもう一度行わずに済み、検証後に別の User 表現を参照する食い違いも避けられる。

未認証と認証保留は `ErrAdminAuthenticationRequired`、それ以外の条件不一致は `ErrAdminAccessDenied` とし、既存の管理 API の応答規則を変えない。

### 既存判定の移行

| 現在の判定 | 移行後 |
| --- | --- |
| Tenancy の `requireSystemAdmin` | `RequireControlPlaneUser` を使用する |
| Audit の `system_admin`、`default`、`all_tenants` の組合せ | `RequireControlPlaneUser` が成功した場合だけ横断範囲を作る |
| Jobs の `RequireAdmin`、`actorMayReadAllTenants`、`scopeFor` | 制御面主体には `RequireControlPlaneUser` だけを要求し、それ以外の `admin` にはテナント内範囲だけを作る |
| SigningKeys と DataKeys の `requireSystemKeyHealthReader` | `RequireControlPlaneUser` を使用する |

Audit と Jobs のテナント内経路は通常の `admin` に開いたままにし、横断範囲へ切り替える箇所だけを共通判定へ通す。

### 拒否の観測

受け入れテストは、`default` 以外のテナントに属する `system_admin` が両方のヘルス API で拒否され、応答に別テナントの識別子や状態が含まれず、ヘルス取得ユースケースも呼ばれないことを確認する。

許可側では、`default` テナントの有効な User が直接または Group 経由で `system_admin` を持つ場合に横断ヘルスを取得でき、`admin` を併有しなくても Jobs の横断一覧、詳細参照、取消しを行えることを確認する。

## Plan

1. 3つのシナリオ、TypeSpec 文書コメント、関連する決定と用語集を制御面主体の条件へ揃える。
2. HTTP 境界で、`default` 以外の `system_admin` が両方の横断ヘルスを取得できる現在の挙動を観測し、受け入れ RED を確認する。
3. `RequireControlPlaneUser` の Unit RED を確認し、認証状態、有効状態、要求先、所属先、有効ロールの条件を実装する。
4. 5か所の制御面判定を共通処理へ移し、Jobs から余分な `admin` 条件を除いた許可経路と拒否経路を回帰テストで固定する。
5. `requireSystemAccount` を同じ条件へ揃え、制御面テナント外の `system_admin` が `/system` を開けないことを確認する。
6. 仕様生成物を再生成し、検査を通す。

## Tasks

- [ ] T001 [Spec] 3つのシナリオ、TypeSpec 文書コメント、関連する決定と用語集を更新する。
- [ ] T002 [Acceptance] 制御面テナント外の `system_admin` が両方の横断ヘルスを取得できることを HTTP 境界で観測し、RED を確認する。
- [ ] T003 [App] `RequireControlPlaneUser` の Unit RED を確認してから共通判定を実装する。
- [ ] T004 [App] Tenancy、Audit、Jobs、SigningKeys、DataKeys の制御面判定を共通処理へ移す。
- [ ] T005 [App] `requireSystemAccount` を realm と有効ロールの組合せへ揃える。
- [ ] T006 [Acceptance] 拒否応答に別テナントの情報が含まれず、横断取得処理が実行されないことを確認する。
- [ ] T007 [Verify] 仕様生成物を再生成し、検査を通す。

## Verification

- `mise run test-go-race`
- `mise run test-ui-unit`
- `mise run test-ui-e2e`
- `mise run check-spec`
- `mise run check-ids`
- `mise run check-work-items`
- `mise run verify`

## Risk Notes

リスクは high とする。

この変更はテナント境界を越える情報開示を塞ぐため、条件を緩く実装すると漏えい経路が残り、厳しく実装すると正規のシステム運用者が鍵運用状態を観測できなくなる。

5か所を1つの判定へ集約するため、共通処理の誤りはテナント CRUD、監査、ジョブにも広がる。

各移行先では、制御面テナントの直接ロール、Group 由来ロール、別テナント、無効 User、認証保留を表にした回帰テストを先に置く。

制御面テナント外に既存の `system_admin` が保存されていても、この変更後は横断能力を得ない。

保存済みロールの扱いと新規割当ての禁止は [[wi-463-reserve-system-admin-role-to-control-plane]] が扱い、本作業項目は既存データの変更を行わない。

`reversibility` は reversible とする。

共通判定と認可条件は元へ戻せるが、戻すと同じ情報開示が再発する。
