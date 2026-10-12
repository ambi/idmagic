---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-13
priority: p2
depends_on:
  - wi-47518-align-shared-document-placement-and-design-records
  - wi-84348-separate-http-response-shapes-from-domain-models
change_kind: refactor
spec_impact:
  kind: none
  reason: "有効なドメイン状態の構築、復元、更新で既存の不変条件を強制する。HTTP の成功と拒否、認証と認可、既存データの保存形、状態遷移、ドメインイベントと外向きの通知は維持し、新しい業務上の制約を追加しない。"
---

# ドメイン状態の不変条件を構築、復元、更新の境界で強制する

## 動機

[システム全体のリスク](../../docs/design/architecture/risks.md)には、ドメインの構造体の一部が構築後の検証を必要とする負債が載っている。
Tenant などは公開フィールドを組み立てた後で `Validate` を呼ぶ必要があり、その呼び出しを忘れるか、検証後にフィールドを変更できる。
[型の所有と不変条件](../../docs/design/application/design-guidelines.md#型の所有と不変条件)が求める最も狭い場所での強制を、既存の有効なドメイン状態へ適用する是正項目は確認できなかった。

HTTP 表現の分離を先行することで、内部状態のカプセル化を外部 JSON の形へ直接波及させない。
Go の任意のゼロ値まで不正な状態が存在しないという全体保証は置かず、有効なドメイン状態として使う経路を制御する。

## 対象範囲

- `backend/` の各モジュールで、永続化する状態と業務判断に使う状態のうち、構築後の `Validate` または呼び出し側の条件チェックに依存する型を分類し、既存の不変条件を型の構築と状態遷移に集める。
- メモリと PostgreSQL の復元、Repository の保存、HTTP、ジョブ、バッチ、seed、テスト前提の組み立てを新しい経路へ揃える。
- 外部から変更できるフィールド、ポインター、スライス、マップを通じた検証後の状態変更を防ぐ。
- 現行リスクの該当行と、変更した公開契約およびモジュールの設計を検証結果へ合わせる。

## 対象外

- 新しい業務制約、永続化の制約、ID の形式、必須フィールドの追加。
- 外部 DTO、デコード途中の入力、部分更新の入力、永続化の行表現まで、有効な業務状態として扱うこと。
- 全識別子の一律の別名型化、汎用の Validated 型や新しい共有フレームワークの導入。
- モジュールの分割、テーブル所有者の変更、公開パッケージやモジュール間依存の追加。
- データ破損の修復や既存データの移行。既存の検証を破る保存データが見つかった場合は、その検出と独立した是正の根拠を記録する。

## 設計

### 分類と対象の確定

起票時の探索の起点は次の状態である。
パッケージは `internal/` への移行を受けるため、着手時に責務表から現在位置を解決する。

| モジュール | 状態と不変条件の起点 |
| --- | --- |
| Tenancy | Tenant、TenantBranding、realm と設定の整合 |
| IdManagement | User、UserLifecycle、Group、GroupMember、DynamicGroupRule、Agent と credential binding、属性の状態 |
| Authentication | LoginSession、LoginRequest、MFA の状態、TrustedDevice、FederatedIdentity と接続、WebAuthnCredential、RecoveryCode |
| OAuth2 | OAuth2Client、AuthorizationRequest、AuthorizationCodeRecord、PARRecord、ApprovalRequest、DeviceAuthorization、Consent、RefreshTokenRecord、McpResourceServer |
| Saml、WsFederation | IdP profile、service provider、relying party の保存状態 |
| IdGovernance | LifecycleWorkflow、revision、WorkflowRun、WorkflowStep と、その内側の trigger、action |
| Authorization | AuthorizationModel、RelationTuple |
| Provisioning | ProvisioningConnection、ProvisioningTask と、その内側の設定 |
| WorkloadIdentity | AgentWorkloadBinding、WorkloadTrustBundle |
| SharedSignals | stream、transmitter と receiver の設定、event、delivery、revocation epoch の保存状態 |

すべての本番 domain の `Validate`、構築関数、状態更新、Repository の Save と復元を検索する。
型ごとに、不変条件、構築元、更新元、復元元、検証の現在の所有者、今回の対象か除外か、その理由を本項目へ記録する。
上の一覧を直しただけで探索を終えず、`Validate` を持たないが呼び出し側が検証している保存状態も調べる。

有効な状態として保存または業務判断に使われ、不変条件が存在する型を対象とする。
デコード途中の入力、部分更新の値、行表現、外部プロトコル専用の表現、妥当性が状態に依存する要求は、有効なドメイン状態と分ける。
それらは境界で検証する根拠を記録し、入力を作れることだけを設計違反としない。

### 構築と読み取り

所有する domain が、必要な入力を受けて状態と error を返す構築関数を公開する。
既存の条件を一つの検証処理で強制し、不正な入力から有効な状態を返さない。
公開フィールドによる直接構築から、この関数を通した構築へ切り替える。
外側のパッケージから不変条件を破る書き込みができないよう、状態のフィールドを非公開にする。
読み取りは公開する値または snapshot を返し、可変なポインター、スライス、マップは内部状態と共有しない。
型の細部の名前は各モジュールの語彙に合わせ、全モジュールへ同じ名前の機械的なラッパーを足さない。

外部 DTO と永続化の行表現は別の構造体で保持する。
構築入力には欠落や不正な値が存在し得るが、構築に成功した状態とは区別する。
Go のゼロ値は存在し得るため、ゼロ値を有効な状態として保存または操作する経路では error として拒否する。
正規の不在やゼロ値が現在の仕様で有効な項目は、既存の意味を維持する。

### 更新と復元

変更は、既存のドメイン操作と状態遷移のメソッドを通す。
更新候補を検証してから状態へ反映し、拒否した更新では元の状態を維持する。
呼び出し側が作った snapshot をそのまま有効な状態として差し戻す経路は作らない。
メモリと PostgreSQL のアダプターは、行や snapshot から domain の復元関数を通して状態を返す。
復元関数も既存の不変条件を強制し、検証を飛ばす公開コンストラクターを用意しない。

保存ポートは有効な状態だけを受け取る。
ゼロ値や破損した復元データを拒否する場合、既存の正常データの読み書きと外部応答を維持する。
検証の error を外部へ返す方式は既存の分類に従う。
検証の場所を移すことで、認可、必要な副作用、監査、トランザクションの確定の順を変えない。

既存の保存型を公開したまま Validate の呼び出しを増やす案は、検証後の書き換えを止められないため採らない。
全識別子を別名型へ変える案は、この負債に必要な範囲を超えるため採らない。
境界は各モジュール内に維持し、HTTP 表現は wi-84348 で分離したアダプター側の型を使う。

## 計画

1. 型とその構築、更新、復元、保存の経路を分類し、対象と除外の理由を記録する。
2. 既存の不変条件と正常、拒否の振る舞いを固定し、必要な特性化を製品変更から分ける。
3. 各対象で構築、読み取り、更新、復元のインターフェースを実装し、すべての利用元を移す。
4. メモリと PostgreSQL、ジョブとバッチ、seed と fixture の経路で、検証の省略と検証後の変更を防ぐことを確認する。
5. 全対象の残存を再検索し、検証済みの範囲を設計とリスクへ反映する。

## タスク

- [ ] T001 [Inventory] 不変条件を持つ状態と入力表現を分類し、対象型と全構築、更新、復元、保存の経路を記録する。
- [ ] T002 [Characterize] 既存の正常と拒否、更新の無作用、復元と外部応答を固定し、必要な特性化を別コミットへ残す。
- [ ] T003 [Domain] 構築と更新で既存の不変条件を強制し、フィールドと可変な値の公開を制限する。
- [ ] T004 [Adapters/Usecases] HTTP、メモリと PostgreSQL、ジョブ、バッチ、seed と fixture を正規の構築と復元へ移す。
- [ ] T005 [Acceptance] 不正な構築と更新、検証後の変更、破損した復元、ゼロ値の誤用を検出し、正常データと外部契約が維持されることを確認する。
- [ ] T006 [Inventory] 対象の直接構築と書き換え、公開の検証回避、復元と保存の検証省略が残らないことを再検索する。
- [ ] T007 [Docs/Verify] 全対象を完了してからリスクの該当行を更新し、公開契約と設計、規範差分、変更耐性と最終検査を完了する。

## 検証

- `mise run test-go-package -- <package>` で構築と状態遷移の公開境界を通し、正常な状態、不正な入力、更新拒否後の状態、snapshot の変更による内部状態の不変を観測する。
- メモリと PostgreSQL の Repository に同じ契約を適用し、保存と復元で検証を飛ばさないことと、正常データの互換性を確認する。
- 必要な HTTP 契約証拠は `mise run report-contract-evidence` で照合し、変更した正式な入口の既存 REQ と operation のテストを実行する。
- `mise run test-go-mutation -- <変更した package directory>` で検証と状態遷移の故障を測る。復元で検証を飛ばす、検証前に状態を書き換える、可変な snapshot を内部と共有する故障は必要な場合だけ手動注入する。
- `mise run check-boundaries`、`mise run check-boundary-debt-ratchet -- main`、`mise run test-go-changed`、`mise run lint-go` を実行する。
- 保存表現を維持して `mise run check-contract-drift` と `mise run spec-diff` を確認し、影響範囲に従って `mise run verify` を選ぶ。
- 完了記録には全対象と除外、利用元の移行結果、正常データの互換性、拒否後の無作用、残存の再検索を記録する。

## リスク

入力 DTO と有効な状態を混同すると、欠落を許す部分更新や既存の既定値を拒否してしまう。
型の役割を分類し、既存の入力の意味を維持する。
同じ検証を新しい条件で書き直すと業務契約が変わるため、既存条件を一つの処理で再利用する。
既存の公開パッケージの外側の利用元も移行し、HTTP、seed、ジョブ、fixture のどれかだけが旧構築を使う状態を残さない。
