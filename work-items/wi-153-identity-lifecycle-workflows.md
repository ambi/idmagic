---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-10
---

# identity lifecycle workflow を導入する

## Motivation
IdMagic は User lifecycle、Group、Application assignment を持つが、入社前、初日、
異動、休職、退職などの identity lifecycle event に応じて一連の操作を自動実行する
workflow を持たない。Microsoft Entra Lifecycle Workflows や Okta Workflows のような
自動化がないと、属性変更や退職処理に伴うアクセス変更、通知、required action 付与を
運用者が手作業で行う必要がある。

本 WI は、User lifecycle / attribute change を trigger にして、グループ割当、アプリ割当、
required action、通知、無効化などを実行する identity lifecycle workflow を導入する。

## Scope
- **scl**:
  - `IdentityManagement` に LifecycleWorkflow / WorkflowTrigger / WorkflowAction / WorkflowRun を追加する。
  - User created / updated / status changed / required action changed を workflow trigger として明示する。
  - `Application` assignment と `Authentication` required action への action を定義する。
  - workflow run events と失敗時の監査モデルを追加する。
- **go**:
  - workflow definition validator、trigger dispatcher、action executor、run repository を実装する。
  - `wi-126-async-job-runner` が利用可能なら workflow run を job として実行する境界を用意する。
- **http**:
  - workflow CRUD、dry-run、run history API を追加する。
- **ui**:
  - admin に workflow editor、dry-run、実行履歴、失敗詳細を追加する。
- **documentation**:
  - README に初期 trigger / action 一覧と運用例を追記する。

## Out of Scope
- 汎用 DAG / BPMN / 複雑な分岐ループ。
- 外部 SaaS connector や webhook action の本番実装。
- durable async job 基盤そのもの。これは `wi-126-async-job-runner` が扱う。
- HR system からの inbound provisioning 実装。

## Plan
- 初期は identity lifecycle に閉じた構造化 workflow とし、trigger / action を enum で制約する。
- action は冪等に実装し、同じ event の再処理で重複割当や重複 required action が起きないようにする。
- workflow は dry-run を必須機能にし、対象ユーザーと実行予定 action を管理者が確認できるようにする。
- async job 基盤が未完了でも同期実行で最小実装できるよう usecase 境界を分離し、基盤完了後に載せ替えられるようにする。

## Tasks
- [ ] T001 [SCL] Workflow model、trigger/action、run events、scenarios を追加する。
- [ ] T002 [Decision] workflow の表現力、冪等性、job 基盤との接続方針を ADR に記録する。
- [ ] T003 [App] workflow validator / dispatcher / executor / run store を実装する。
- [ ] T004 [HTTP] workflow CRUD / dry-run / history API を追加する。
- [ ] T005 [UI] workflow editor と run history UI を追加する。
- [ ] T006 [Verify] SCL、Go、UI、手動シナリオを検証する。

## Verification
- `just yaml-check`
- `just check-ids`
- `just test-go`
- `just verify-ui`
- 手動: employment_type が変更されたユーザーに required action と group assignment が自動付与されることを確認する。
- 手動: workflow dry-run が対象ユーザーと action を表示し、無効 workflow は実行されないことを確認する。

## Risk Notes
workflow は複数 context の状態を自動変更するため、誤設定や再試行で大きな影響が出る。初期は構造化 action に限定し、dry-run、監査、冪等性、テナント境界検証を必須にする。外部 connector や任意コード実行は扱わない。
