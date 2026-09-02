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
  - { path: docs/contexts/signing-keys/scenarios.md, requirement: REQ-SIGNINGKEYS-009 }
  - { path: docs/contexts/data-keys/scenarios.md, requirement: REQ-DATAKEYS-006 }
  - { path: docs/contexts/identity-management/scenarios.md, requirement: REQ-IDMANAGEMENT-014 }
  - { path: spec/contexts/signing-keys/main.tsp, symbol: IdMagic.SigningKeys.Operations.ListTenantKeyHealth }
  - { path: spec/contexts/data-keys/main.tsp, symbol: IdMagic.DataKeys.Operations.ListTenantDataKeyHealth }
---

# テナント横断のヘルス参照に制御面テナント所属を要求し、`system_admin` を予約ロールにする

## Motivation

`docs/authorization.md` のテナント境界は、テナントを越える操作について「`system_admin` ロールと**デフォルト (制御面) テナントへの所属**を合わせて要求する」と定めている。この規則を実際に守っているのはテナント CRUD (`backend/tenancy/handlers_http/admin_tenant_handler.go:258`)、横断監査 (`backend/audit/handlers_http/admin_audit_event_handler.go:444`)、横断ジョブ (`backend/jobs/handlers_http/admin_job_handler.go:161`) の 3 か所である。

テナント横断のヘルス参照 2 本だけが、ロールしか見ていない。

| 欠陥 | 位置 | 内容 |
| --- | --- | --- |
| A | `backend/signingkeys/handlers_http/admin_key_handler.go:199` | `slices.Contains(actor.Roles, "system_admin")` だけを条件にし、所属テナントを見ない |
| A | `backend/datakeys/handlers_http/admin_data_key_handler.go:63` | 同上 |
| B | `backend/idmanagement/usecases/helpers.go:28` | `NormalizeRoles` は空白除去と重複排除だけで値を検証せず、`backend/idmanagement/user/handlers_http/admin_user_handler.go:33` はリクエスト本文の `roles` をそのまま渡す |

A だけなら、`system_admin` が制御面テナントにしか存在しない限り実害は出ない。B があるため出る。**任意のテナントの `admin` が、自テナントの User または Group に `system_admin` ロールを割り当てられる。** 経路はユーザー作成・更新 (`backend/idmanagement/user/usecases/admin_users.go:145,276`) とグループ作成・更新 (`backend/idmanagement/group/usecases/admin_groups.go:210,312`) の両方にあり、後者はグループ由来の有効ロールとして効く。

2 つを合わせると、テナント "acme" の管理者が自テナントに `system_admin` を作り、`/realms/acme/api/admin/v1/keys/health` を呼ぶだけで**全テナントの `tenant_id`、プロバイダー、`active_kid`、鍵数、到達性**が返る。DEK ヘルスも同様である。経路はホスト形式とパス形式の両方に登録されている (`backend/shared/http/server_http/routes.go:156-158`、`backend/signingkeys/handlers_http/routes.go:27`)。フロントエンドのガード (`frontend/src/routes/-guards.ts:52`) もロールしか見ないため、この利用者はシステムコンソールの画面自体を開ける。

これは `docs/authorization.md` の「応答が決して含まないもの — 要求者が権限を持たないテナントの識別子、名前、件数」に反する。鍵素材は返らないが、共有基盤に同居する他テナントの存在と鍵運用の状態が漏れる。

**仕様の側にも欠落がある。** `REQ-SIGNINGKEYS-009` と `REQ-DATAKEYS-006` はどちらも条件を「`system_admin` ロールを持つか」だけで書いており、制御面テナント所属に触れていない。実装は自分のシナリオどおりに動いており、シナリオが `docs/authorization.md` の規範より弱い。したがって修正はシナリオから入る。

## Scope

- `ListTenantKeyHealth` と `ListTenantDataKeyHealth` の認可を、テナント CRUD・横断監査・横断ジョブと同じ条件に揃える。
- テナント横断を許すかどうかの判定を 1 つの述語に集約し、4 か所の手書きを置き換える。
- `REQ-SIGNINGKEYS-009` と `REQ-DATAKEYS-006` に、制御面テナントの外にいる `system_admin` が拒否されることを規範として書く。
- `system_admin` を予約ロールとし、制御面テナント以外の User と Group への割り当てを拒否する。ユーザー作成・更新とグループ作成・更新の両方を塞ぐ。
- `REQ-IDMANAGEMENT-014` に、予約ロールの割り当てが拒否されることを書く。

## Out of Scope

- 制御面操作を API アクセストークンから到達不能にすること。[[wi-461-control-plane-credential-boundary]] が持つ。
- テナント横断操作の UI をシステムコンソールへ集約すること。[[wi-462-control-plane-console-single-entry]] が持つ。
- ロール語彙を閉じた集合にすること。`docs/contexts/tenancy/decisions.md` はロール名を `User.roles` に直接持つ設計を選んでおり、語彙の閉包はその判断のやり直しになる。本 work item は `system_admin` 1 語を予約するだけで、他のロール名は自由なままにする。
- 制御面の判定を持つ経路がすべて共通述語を通っていることの機械検査。同じ欠陥を再発させないための検査であり、経路の分類を伴うため別に扱う。
- 制御面操作への step-up と再認証。
- 監査イベントの横断検索に規範シナリオがないこと。`docs/contexts/audit/scenarios.md` は横断検索を持たず、実装だけが `admin_audit_event_handler.go:444` で条件を持つ。欠落の補いは本 work item の対象外とし、[[wi-462-control-plane-console-single-entry]] で扱う。

## Design

### 直し方を「条件を書き直す」ではなく「述語に寄せる」にする

同じ規則が 4 か所に手書きされ、そのうち 2 か所が食い違ったことが今回の原因である。2 か所を書き直すだけでは、次に横断参照が生えたときに同じ欠陥が再生産される。制御面の主体かどうかを返す述語を `backend/shared/http/support_http` に 1 つ置き、既存の 4 か所すべてがそれを通るようにする。

述語が満たすべき条件は、現在いちばん厳しい `requireSystemAdmin` (`admin_tenant_handler.go:245`) に揃える。

| 条件 | 現在の実装差 |
| --- | --- |
| 認証済みで、認証が保留状態でない | tenancy のみが `AuthenticationPending` を見る |
| 主体が有効である (`user.IsActive()`) | tenancy のみが見る |
| `tenant_id` が制御面テナントである | tenancy、audit、jobs が見る。鍵と DEK のヘルスは見ない |
| 有効ロールに `system_admin` を含む | tenancy は `EffectiveRoles` (グループ由来を含む)、他は `actor.Roles` (直接付与のみ) |

有効ロールは `EffectiveRoles` に揃える。グループ由来のロールを制御面判定で無視すると、`REQ-IDMANAGEMENT-015` が定める有効ロールの意味と、認可の判定が食い違うためである。予約ロールの制限 (下記) を同時に入れるので、`EffectiveRoles` に揃えても制御面テナントの外から `system_admin` は湧かない。

### 予約ロールの規則

**`system_admin` は制御面テナントの User と Group にだけ割り当てられる。** それ以外のテナントの主体へ割り当てようとしたリクエストは拒否する。判定は所属テナントで行い、操作者のロールでは行わない。

却下した案。

- **鍵と DEK のヘルスだけを直し、ロールの割り当ては自由なままにする。** 今日の到達先は 2 本だが、`system_admin` を条件にする判定が今後増えるたびに、テナント管理者が自分で作れる主体がその条件を満たしてしまう。二重防御として両方を直す。
- **「操作者が持つロールしか付与できない」規則にする。** 昇格の一般則としては正しいが、テナント管理者が `admin` を付与する現在の運用まで制約が及び、変更の範囲がこの欠陥に対して広すぎる。予約する語を 1 つに絞れば、同じ昇格経路は塞げる。
- **ロール名の語彙を閉じた集合にする。** Out of Scope に挙げた理由による。
- **文書だけを直し、`REQ-SIGNINGKEYS-009` の条件を実装に合わせて弱める。** `docs/authorization.md` のテナント境界が正であり、シナリオがそれに従う。実装を通すために規範を弱めない。

### シナリオへの書き方

新しい `REQ` 番号は割り当てず、既存のシナリオへ `ALT` を足す。番号は取り消せないので、既存のシナリオが同じ主題を持つ間は増やさない。

| シナリオ | 追加する `ALT` |
| --- | --- |
| `REQ-SIGNINGKEYS-009` | 制御面テナント以外に所属する `system_admin` が署名鍵ヘルスを呼ぶ → `AccessDeniedError` |
| `REQ-DATAKEYS-006` | 同上を DEK ヘルスについて |
| `REQ-IDMANAGEMENT-014` | `admin` が自テナントの User または Group に `system_admin` を割り当てる → 拒否され、ロールは変わらない |

TypeSpec 側は `ListTenantKeyHealth` と `ListTenantDataKeyHealth` の doc が「A system_admin lists ...」とだけ書いているため、制御面テナントからの呼び出しであることを doc に加える。`x-api-token-scopes` は本 work item では変えない ([[wi-461-control-plane-credential-boundary]] が扱う)。

### 影響を確かめる範囲

- `seed/manifests/*.yaml` が置く `system_admin` の主体が制御面テナントに属していることを確認する。属していなければ seed は新しい規則で拒否され、開発環境が起動しなくなる。
- フロントエンドの `requireSystemAccount` (`frontend/src/routes/-guards.ts:52`) はロールだけを見ている。サーバー側が正であり UI の非表示は認可判定ではないが、画面を開いてから 403 が並ぶ状態にはしない。ガードの条件も揃える。

## Plan

1. `REQ-SIGNINGKEYS-009`、`REQ-DATAKEYS-006`、`REQ-IDMANAGEMENT-014` に `ALT` を足し、TypeSpec の doc を合わせる。
2. 受け入れ境界で RED を確認する。制御面テナント外の `system_admin` がヘルスを呼べること、テナント管理者が `system_admin` を割り当てられることの 2 つが、現在は成功してしまうことを観測する。
3. 制御面の主体を判定する述語を `support_http` に置き、tenancy・audit・jobs・signingkeys・datakeys がそれを通るようにする。
4. ロールの書き込み経路 (User 作成・更新、Group 作成・更新) に予約ロールの検査を入れる。検査はユースケース層に置き、HTTP ハンドラーには置かない。SCIM の経路は `Roles` を空で作るため対象外だが、そのことを確認する。
5. `seed` と開発環境の起動を確認する。
6. フロントエンドのガードを揃える。
7. 検査を通す。

## Tasks

- [ ] T001 [Spec] `REQ-SIGNINGKEYS-009`、`REQ-DATAKEYS-006`、`REQ-IDMANAGEMENT-014` に `ALT` を追加し、TypeSpec の doc を更新する。
- [ ] T002 [Acceptance] 現在の実装で 2 つの経路が通ってしまうことを HTTP 境界で観測し、RED を確認する。
- [ ] T003 [App] 制御面の主体を判定する述語を `support_http` に導入し、既存 4 か所を置き換える。
- [ ] T004 [App] 鍵ヘルスと DEK ヘルスの守りを述語へ切り替える。
- [ ] T005 [App] User と Group のロール書き込みに予約ロールの検査を入れる。
- [ ] T006 [App] `seed` マニフェストと開発環境の起動を確認する。
- [ ] T007 [App] `requireSystemAccount` の条件を揃える。
- [ ] T008 [Verify] 検査を通す。

## Verification

- `mise run test-go-race`
- `mise run test-ui-unit`
- `mise run check-spec`
- `mise run check-ids`
- `mise run check-work-items`
- `mise run verify`

## Risk Notes

リスクは high。テナント境界を跨ぐ情報開示を塞ぐ変更であり、直し方を誤ると穴が残るか、正規の運用者が締め出される。

**誤りの向きが 2 つある。** 条件を緩く書けば穴が残り、厳しく書きすぎれば制御面テナントの `system_admin` が鍵と DEK のヘルスを見られなくなる。ヘルス参照は鍵の回転漏れとフェイルクローズ条件を判断する経路なので、後者は運用の目を失うことを意味する。どちらも受け入れ境界のテストで観測してから実装する。

**述語への集約が新しい失敗の形を作りうる。** 5 か所が 1 つの判定を共有するため、述語を誤ると影響がテナント CRUD と横断監査にも及ぶ。置き換えの前に、既存 4 か所の現在の振る舞いを固定するテストを揃える。

**予約ロールの検査は既存データに遡らない。** 制御面テナント以外に `system_admin` を持つ主体がすでに存在する環境では、割り当てを塞いでも残る。移行時に既存データを調べる手順を実装時に決める。放置すると欠陥 A を直しても経路が残る。

`reversibility` は reversible。ロール 1 語の予約と認可条件の統一であり、取り消しても外部に保存された値の意味は変わらない。ただし予約ロールの導入は、`system_admin` を制御面テナント外に置いていた利用者にとっては破壊的なので、着手時に `documentation_impact` を判断する。
