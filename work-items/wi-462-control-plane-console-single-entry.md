---
status: pending
authors: [tn]
risk: medium
reversibility: irreversible
created_at: 2026-09-03
change_kind: feature
priority: p2
depends_on: []
affected_spec:
  - { path: docs/contexts/tenancy/scenarios.md, requirement: REQ-TENANCY-014 }
  - { path: docs/contexts/jobs/scenarios.md, requirement: REQ-JOBS-012 }
  - { path: docs/contexts/audit/scenarios.md, requirement: REQ-AUDIT-001 }
---

# テナント横断操作をシステムコンソールへ集約し、制御面の入口を一本化する

## Motivation

システムコンソール (`/system`) は、テナント横断の管理領域として意図的に分離されている。別ルート、別シェル、別配色、ロールでのゲート (`frontend/src/routes/-guards.ts:52`、`frontend/src/components/SystemShell.tsx`、`frontend/src/lib/systemNav.ts`) と、分離の意図はコードのコメントにも書かれている。

しかし持っている画面はテナント一覧、署名鍵ヘルス、DEK ヘルスの 3 つだけで、**テナント横断の操作がすべてそこにあるわけではない。**

| 横断操作 | いまの置き場所 | サーバー側の判定 |
| --- | --- | --- |
| テナント一覧・作成・更新・停止・上限 | `/system/tenants` | `admin_tenant_handler.go:258` |
| 署名鍵ヘルス | `/system/keys` | `admin_key_handler.go:199` |
| DEK ヘルス | `/system/data-keys` | `admin_data_key_handler.go:63` |
| **監査イベントの横断検索** | `/admin/audit_events` (`AdminAuditEventsPage.tsx:285`) | `admin_audit_event_handler.go:444` |
| **ジョブの横断一覧** | `/admin/jobs` (`AdminJobsPage.tsx:92`) | `admin_job_handler.go:161` |

後ろ 2 つは、テナント管理コンソールの画面が操作者のロールと所属を見て横断表示に切り替わる形になっている。同じ利用者が同じセッションで、テナント管理と制御面をひとつの画面の中で行き来する。**分離した意味が画面の側で失われている。**

さらに、監査イベントの横断検索には規範シナリオがない。`docs/contexts/audit/scenarios.md` は横断検索を書いておらず、条件は実装だけが持っている。`REQ-JOBS-012` はジョブについて「自テナントのジョブだけを一覧・参照できる」を書き、横断を許す条件はテストのコメントにある。制御面の到達範囲は認可の一部なので、規範のない横断能力を残さない。

`docs/structure.md` も実態とずれている。「制御面のテナント管理だけを `/realms/default/admin/tenants` に分離する」と書いているが、実装は制御面の経路を共有のテナントグループへ登録し、制御面テナントへの限定は認可層が担っている (`backend/shared/http/server_http/routes.go:174-178`)。経路としては `/realms/{任意}/api/admin/v1/tenants` が存在する。到達範囲を説明する文書がここで実態と食い違っている。

## Scope

- 監査イベントの横断検索とジョブの横断一覧を、システムコンソールの画面へ移す。テナント管理コンソール側の画面は自テナントに閉じる。
- 監査イベントの横断検索に規範シナリオを与える。条件は `system_admin` ロールと制御面テナント所属の両方とする。
- `REQ-JOBS-012` に、横断が制御面テナントの `system_admin` にだけ許されることを規範として書く。テストのコメントにしか無い条件を規範へ上げる。
- 「テナント横断の操作はシステムコンソールにだけ現れる」を不変条件として `docs/contexts/system/decisions.md` に記録する。
- `docs/structure.md` の制御面の経路に関する記述を実装に合わせる。

## Out of Scope

- サーバー側の認可の変更。監査とジョブの横断判定は現在正しい。移すのは画面である。
- テナント横断のヘルス参照の認可欠陥。[[wi-460-cross-tenant-health-control-plane-membership]] が持つ。
- 制御面操作に到達できる資格情報の種類。[[wi-461-control-plane-credential-boundary]] が持つ。
- 制御面を一般利用者向けの公開入口から到達不能にすること、管理コンソールを別ホストへ移すこと。[[wi-459-api-process-plane-separation-decision]] の D3 であり、同 work item の再検討条件 C5 が扱う。
- 経路の接頭辞と種別の対応を機械検査すること。[[wi-459-api-process-plane-separation-decision]] が Out of Scope として挙げた検査に含める。
- システムコンソールへ新しい運用機能 (容量、リリース、バックアップ、機能レジストリの状態) を足すこと。これらは runbook と配備側の道具が持つ領域で、製品の権限体系の中で表現する操作ではない。

## Design

### 不変条件

**テナント境界を越える操作は、システムコンソールにだけ現れる。テナント管理コンソールの画面は、操作者が誰であっても自テナントに閉じる。**

これを `docs/contexts/system/decisions.md` に置く理由は、システムコンソールの存在自体は既にコードのコメントが説明している一方、**どこまでがそこに属するか**を決めた記録がどこにもないためである。記録がないと、次にテナント横断の参照を足すとき、置き場所は実装者の判断になる。現に監査とジョブはテナント管理コンソール側に生えている。

不変条件の帰結として、`system_admin` がテナント管理コンソールを開いたときの見え方も決まる。自テナント (制御面テナント) の管理者として振る舞い、横断の入口は持たない。横断が要るときはシステムコンソールへ移る。

### 画面をどう移すか

`AdminAuditEventsPage` と `AdminJobsPage` は、いま `canCrossTenant` で表示と問い合わせを切り替えている。この分岐を消し、テナント用のページは常に自テナントに閉じる。システムコンソール側には横断用のページを置き、`all_tenants` を常に付けて呼ぶ。

分岐を残したまま `/system` にも同じページを置く案は採らない。同じコンポーネントが 2 つの文脈で違う権限前提を持つ状態が残り、片方の変更が他方へ漏れる。表示ロジックの分離はテナント側と制御面側で別のページに分ける形で行う (`work-items/done/wi-166-admin-resource-page-presentation-logic-separation.md` が置いた形に従う)。

### 規範をどう足すか

監査の横断検索には新しい `REQ-AUDIT-NNN` を割り当てる。既存のシナリオはいずれも自テナントの管理者を主体にしており、横断は主体も条件も違うため、`ALT` では表せない。番号の割り当ては取り消せないので、`reversibility` は irreversible とする。

ジョブは `REQ-JOBS-012` に `ALT` を足す。同じシナリオが「自テナントのジョブだけを一覧・参照できる」を主題にしており、横断が許される条件はその例外として読めるためである。

### 却下した選択肢

- **画面はそのままにして、不変条件だけを文書に書く。** 文書と実装が最初から食い違う。守られていない不変条件は約束ですらない。
- **システムコンソールを廃止し、すべてをテナント管理コンソールのロール分岐に統一する。** 横断能力が画面の分岐として散らばり、どの画面が制御面かを読み取る手段がなくなる。分離の意図はすでにコードにあり、それを捨てる理由がない。
- **横断監査と横断ジョブを、テナント管理コンソールとシステムコンソールの両方に置く。** 上記のとおり、同じコンポーネントが 2 つの権限前提を持つ。
- **`docs/structure.md` の記述に合わせて実装を変え、制御面の経路を `/realms/default` 配下だけに登録する。** 経路の登録先を変えると到達範囲は狭まるが、認可層の判定はそれでも要る (`/realms/default` は制御面テナント以外の利用者も通る)。二重にする利得はあるが、[[wi-459-api-process-plane-separation-decision]] が D2 として扱う経路登録の選択と重なるため、ここでは文書を実態に合わせるだけにする。

## Plan

1. 監査の横断検索に規範シナリオを与え、`REQ-JOBS-012` に `ALT` を足す。
2. 不変条件を `docs/contexts/system/decisions.md` に記録し、`docs/structure.md` を実態に合わせる。
3. システムコンソール側に横断監査と横断ジョブのページを追加し、`systemNav` に項目を足す。
4. テナント管理コンソール側の `canCrossTenant` 分岐を消す。
5. 経路のガードを確認する。新しい 2 ページも `requireSystemAccount` を通す。
6. 検査を通す。

## Tasks

- [ ] T001 [Spec] 監査の横断検索に規範シナリオを追加し、`REQ-JOBS-012` に `ALT` を足す。
- [ ] T002 [Spec] `docs/contexts/system/decisions.md` に不変条件を記録し、`docs/structure.md` を修正する。
- [ ] T003 [Acceptance] テナント管理コンソールの画面が横断表示に切り替わることを観測し、RED を確認する。
- [ ] T004 [App] システムコンソールに横断監査と横断ジョブのページを追加する。
- [ ] T005 [App] テナント側ページの横断分岐を削除する。
- [ ] T006 [App] `systemNav` とシェルの案内を更新し、i18n 辞書へ文言を足す。
- [ ] T007 [Verify] 検査を通す。

## Verification

- `mise run test-ui-unit`
- `mise run test-ui-e2e`
- `mise run test-go-race`
- `mise run check-spec`
- `mise run check-ids`
- `mise run verify`

## Risk Notes

リスクは medium。サーバー側の認可は変えないため、誤っても権限が広がる方向には壊れない。壊れるとすれば、制御面テナントの運用者が横断の目を失う方向である。

**移設で機能を落とす危険がある。** 監査の横断検索は絞り込み、ページング、エクスポートを持つ。カーソルはテナントと絞り込み条件を束縛する (`docs/authorization.md` のテナント境界) ため、横断のカーソルをテナント用の画面と共有しない。移設後に絞り込みとページングが横断の文脈で成立することを確かめる。

**不変条件は機械検査を持たない。** 「テナント横断の操作はシステムコンソールにだけ現れる」を守るものは、いまのところ規律だけである。[[wi-459-api-process-plane-separation-decision]] が同じ性質のリスク (経路接頭辞と種別の対応に検査がない) を残しており、検査の追加はそちらの Out of Scope に連なる。ここでも約束だけで守られていることを記録に残す。

`reversibility` は irreversible。新しい `REQ-AUDIT-NNN` を割り当てるためである。画面の移設そのものは取り消せる。
