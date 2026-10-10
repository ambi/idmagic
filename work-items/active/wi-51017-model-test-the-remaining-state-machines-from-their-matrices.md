---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: [wi-86874-model-test-identity-management-state-machines-from-their-matrices, wi-35451-rewrite-remaining-context-specifications-in-the-requirement-format]
change_kind: tooling
spec_impact:
  kind: none
  reason: "状態遷移表から予測するテストを加えるだけで、要件、TypeSpec の契約、永続状態、発行するイベント、外向きの呼び出しは変えない。表と実装の食い違いは、仕様にない振る舞いの分類に従って別に扱う。"
---

# IdManagement 以外の状態機械に、状態遷移表から予測するモデルベースのテストを加える

## 動機

wi-35451 は、IdManagement と Tenancy 以外のすべての Context の状態機械に状態遷移表（マトリクス形式）を加え、`mise run check-spec` で表の存在と遷移の表との一致を求めた。
表と実装の食い違いを実行して検出するテストは、User のライフサイクル（wi-89346）と、wi-86874 が扱う IdManagement の状態機械にしかない。
書き直しの中で、表と実装の食い違い（同意の段を経由しない認可、期限を確かめないデバイス認可の拒否など）を読んで見つけたが、読んで見つけられる量には限りがある。

## 対象範囲

- 状態遷移表を持つ IdManagement 以外の状態機械に、`backend/shared/testing_statematrix` で表を読むモデルベースのテストを加える。全セルを一度ずつ試す網羅と、種を固定した乱数の操作列の二つを置く。
- 時刻と非同期の処理（ジョブの実行、配信の再試行）は、wi-86874 が決める方法でテストから起こす。
- 表と実装の食い違いを、仕様にない振る舞いの分類に従って扱う。

## 対象外

- IdManagement の状態機械（wi-86874 が扱う）。
- 見つけた食い違いの実装の修正。分類の (c) は別に起票する。

## 設計

wi-86874 の組み立て（操作の列の名前から要求への対応、括弧の条件の作り方、状態の名前から永続値への変換だけをテストに置く）に従う。
同じ組み立てが Context をまたいで繰り返されるなら、予測と比較を `testing_statematrix` へ移す。
状態機械ごとに、正式な入口（HTTP の境界かユースケース）のどちらで操作を起こすかを、表の操作の列が観測できる最も外側で選ぶ。

## 計画

Context ごとにテストを加え、Context ごとにチェックポイントを作る。
状態の数と操作の数が多い OAuth2 の AuthorizationCodeFlow は、内部の遷移を列に持つので、ユースケースの境界で試す。

## タスク

- [ ] T001 [Test] Authentication、OAuth2 の状態機械のテストを加える。
- [ ] T002 [Test] Application、ApiTokens、SigningKeys、DataKeys、WorkloadIdentity、SharedSignals のテストを加える。
- [ ] T003 [Test] Jobs、Provisioning、IdentityGovernance、Sourcing、Seeding、Audit、SAML、WS-Federation、System のテストを加える。
- [ ] T004 [Verify] 表のセルと実装に故障を入れて、テストが検出することを確かめる。

## 検証

- `mise run test-go-changed`
- `mise run verify`

## リスク

- 状態機械の数が多く、テストの組み立てが Context ごとに重複しやすい。
  二つ目の Context で共通部分が見えた時点で `testing_statematrix` へ移す。
