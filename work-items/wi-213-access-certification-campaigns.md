---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-15
---

# 定期アクセスレビュー (access certification) を導入する

## Motivation
IdMagic は Group membership と Application assignment を持つが、付与された access が
今も業務上必要かを定期的に再確認し、不要なら失効させる仕組みを持たない。Microsoft Entra
Access Reviews や Okta Access Certifications は、reviewer (self / resource owner など) が
対象ユーザーの group membership や application assignment を周期的に承認・却下し、無回答
時は既定動作 (失効 / 維持) を適用する certification campaign を提供する。これがないと、
異動・退職後も放置された access や、監査人に「誰が何の access を持ち、それを定期的に
見直しているか」を説明する証跡を提示できない。

本 WI は、Group membership と Application assignment を対象にした access certification
campaign (one-time または recurring) を導入し、reviewer による decision の記録、
no-response 時の既定失効、campaign 結果のレポートを提供する。

## Scope
- **scl**:
  - `IdentityManagement` に AccessCertificationCampaign / CertificationScope /
    CertificationItem (対象: GroupMembership または ApplicationAssignment) /
    CertificationDecision (approve/revoke) / CampaignStatus を追加する。
  - campaign 開始 / 締切 / 完了、decision 記録、no-response 時の auto-revoke を
    events / scenarios として追加する。
  - `Application` の ApplicationAssignmentRef、`IdentityManagement` の GroupRef を
    certification 対象として参照する。
  - reviewer は [[wi-94-delegated-administration]] の scoped admin (resource owner)、
    または明示的に指定した User とする。
- **go**:
  - campaign scheduler (one-time / recurring)、対象抽出、reviewer 解決、decision 記録
    usecase、締切超過時の auto-revoke executor、memory / postgres リポジトリを実装する。
  - `wi-126-async-job-runner` を使い、大量対象の抽出と auto-revoke 処理を job として
    実行する。
- **http**:
  - campaign CRUD、対象プレビュー、reviewer 向け決定 API (自分がレビューすべき一覧を
    含む)、campaign 進捗・結果レポート API を追加する。
- **ui**:
  - admin に campaign 作成・進捗ダッシュボードを、reviewer 向けに決定画面 (自分の
    レビュー待ち一覧、承認 / 失効操作) を追加する。
- **documentation**:
  - README に campaign 種別、reviewer 解決規則、no-response 時の既定動作を追記する。

## Out of Scope
- [[wi-154-entitlement-catalog-and-separation-of-duties]] の entitlement assignment や
  [[wi-152-just-in-time-privileged-role-activation]] の eligibility をレビュー対象にする
  ことは初期スコープ外とし、CertificationItem の対象種別を拡張可能な設計に留める。
- manager によるreviewer 自動解決 (User.manager のような属性は現状存在しないため、明示的
  reviewer 指定または [[wi-94-delegated-administration]] の resource owner に限定する)。
- risk score に基づくレビュー優先順位付けや機械学習によるレコメンド。
- 外部 ticketing / SIEM 通知連携。

## Plan
- campaign は「対象範囲 (どの group / application のどの assignment を見直すか) ×
  reviewer 解決規則 × 締切 × no-response 時の既定動作 (失効/維持)」の組で表現し、
  汎用ワークフローエンジンは作らない。
- decision は冪等に記録し、締切超過後の自動失効は [[wi-153-identity-lifecycle-workflows]]
  の action executor と同様の冪等性・監査要件に従う。
- 初期は one-time campaign と recurring campaign の両方をサポートするが、recurring
  間隔は固定周期に限定する。
- campaign 結果は監査イベントと export 可能なレポートとして残し、
  [[wi-148-admin-resource-csv-export]] の CSV export 機構を再利用する。

## Tasks
- [ ] T001 [SCL] Campaign / CertificationItem / Decision model、events、scenarios を追加する。
- [ ] T002 [Decision] レビュー対象範囲の初期選定、reviewer 解決規則、no-response 既定動作を ADR に記録する。
- [ ] T003 [App] scheduler / 対象抽出 / decision usecase / auto-revoke executor を実装する。
- [ ] T004 [HTTP] campaign 管理 API と reviewer 向け決定 API を追加する。
- [ ] T005 [UI] campaign 管理画面と reviewer 決定画面を追加する。
- [ ] T006 [Verify] SCL、Go、UI、手動シナリオを検証する。

## Verification
- `just yaml-check`
- `just check-ids`
- `just test-go`
- `just verify-ui`
- 手動: campaign 作成後、対象の group membership / application assignment が reviewer
  向け一覧に現れ、承認 / 失効を記録できることを確認する。
- 手動: 締切を過ぎても decision が記録されなかった対象が、既定動作 (失効) に従って
  自動的に失効することを確認する。

## Risk Notes
certification の自動失効は現役ユーザーの access を誤って失効させるリスクがある。
auto-revoke は締切超過が明確な対象にのみ fail-closed に適用し、実行前に対象一覧を
管理者が確認できるプレビューを必須にする。decision と auto-revoke は必ず監査イベントに
残す。
