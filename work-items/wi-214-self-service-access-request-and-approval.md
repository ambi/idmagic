---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-15
---

# セルフサービスの access 要求ワークフローを導入する

## Motivation
現状、Group や Application への割当は管理者 (または scoped admin) が直接行うしかなく、
エンドユーザーが必要な access を自ら要求し、承認者が承認したら自動的に付与される導線が
ない。Microsoft Entra entitlement management の access packages や Okta Access
Requests は、カタログ化された requestable resource (group / application) をユーザーが
要求し、resource owner や指定承認者が承認すると自動付与、却下されると何もしない、という
self-service request を提供する。これがないと、access 付与のすべてが管理者への個別依頼
(チケットやチャット) に頼ることになり、対応の遅延や、誰が何を承認したかの証跡不足に
つながる。

本 WI は、Group / Application を requestable catalog として公開し、ユーザーが要求し、
承認者 (resource owner または明示的承認者) が承認 / 却下した結果に応じて自動的に
group membership / application assignment を反映する access request workflow を
導入する。

## Scope
- **scl**:
  - `IdentityManagement` に AccessRequestCatalogItem (対象: Group または Application) /
    AccessRequest / RequestApproval (approver, decision) / RequestStatus を追加する。
  - 要求作成、承認、却下、自動失効 (期限付き付与時) を events / scenarios として
    追加する。
  - `IdentityManagement` の GroupRef、`Application` の ApplicationAssignmentRef を
    要求対象として参照する。
- **go**:
  - catalog 管理、request submission usecase、approver 解決 (明示的承認者または
    [[wi-94-delegated-administration]] の resource owner)、承認時の自動付与 /
    却下時の no-op executor、期限付き付与の失効を実装する。
  - `wi-126-async-job-runner` を使い期限失効を job として実行する。
- **http**:
  - catalog CRUD、request 送信、承認者向けの承認 / 却下 API、request 履歴・状態確認
    API を追加する。
- **ui**:
  - end user の account portal に「access を要求」導線と request 履歴を、承認者向けに
    approval queue を追加する。
- **documentation**:
  - README に catalog 登録方法、承認フロー、期限付き付与の扱いを追記する。

## Out of Scope
- [[wi-154-entitlement-catalog-and-separation-of-duties]] の application entitlement
  を requestable catalog に含めることは初期スコープ外とし、catalog item 種別を
  拡張可能にする。
- 多段階承認チェーン (複数承認者の直列 / 並列合意) は初期スコープ外とし、承認者は
  1 段階 (単一 approver またはいずれか 1 名) に限定する。
- SoD 競合する request の判定は [[wi-154-entitlement-catalog-and-separation-of-duties]]
  の保存時検証に委ね、本 WI では重複 request の防止のみ行う。
- 外部 ticketing システムとの連携。

## Plan
- catalog は管理者が明示的に「requestable」と印を付けた Group / Application に限定し、
  tenant 内の全リソースを既定で requestable にはしない (fail-closed)。
- approval は 1 段階に限定し、approver は catalog item 登録時に明示指定するか、
  [[wi-94-delegated-administration]] の resource owner scoped admin を既定 approver
  として解決する。
- 承認による group membership / application assignment の反映は冪等に実装し、
  二重承認や再処理で重複割当が起きないようにする。
- 期限付き付与 (temporary access) を任意機能として持たせ、失効は
  [[wi-153-identity-lifecycle-workflows]] と同様に job runner 経由の期限監視で行う。

## Tasks
- [ ] T001 [SCL] Catalog / AccessRequest / Approval model、状態遷移、events、scenarios を追加する。
- [ ] T002 [Decision] catalog 登録方針、approver 解決規則、期限付き付与の失効方針を ADR に記録する。
- [ ] T003 [App] catalog 管理 / request submission / approval / 自動付与・失効 executor を実装する。
- [ ] T004 [HTTP] catalog、request、approval API を追加する。
- [ ] T005 [UI] account portal の request 導線と approval queue UI を追加する。
- [ ] T006 [Verify] SCL、Go、UI、手動シナリオを検証する。

## Verification
- `just yaml-check`
- `just check-ids`
- `just test-go`
- `just verify-ui`
- 手動: requestable catalog からユーザーが group access を要求し、approver が承認する
  と group membership が自動的に付与されることを確認する。
- 手動: 却下された request では group membership が付与されず、requester に却下が
  表示されることを確認する。
- 手動: 期限付きで承認された access が、期限経過後に自動的に失効することを確認する。

## Risk Notes
承認による自動付与は、approver 解決を誤るとエンドユーザーが未承認の access を得る
リスクになる。approver 未設定の catalog item は request できない fail-closed な既定と
し、承認・却下・自動付与・自動失効はすべて監査イベントに残す。二重送信・二重承認による
assignment 重複は DB 制約または idempotency key で防ぐ。
