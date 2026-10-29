---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-13
priority: p2
depends_on: [wi-47518-align-shared-document-placement-and-design-records]
change_kind: refactor
spec_impact:
  kind: none
  reason: "HTTP 応答の表現をアダプターが定義し、ドメイン状態から明示的に変換する。HTTP のフィールド名、値、null と省略、ステータスコード、認可、永続状態、ドメインイベントと外向きの通知は維持する。"
---

# HTTP 応答の表現をドメイン型から分離し、アダプターで変換する

## 動機

[システム全体のリスク](../../docs/design/architecture/risks.md)には、HTTP 応答型がドメイン型をフィールドへ直接使う負債が載っている。
[アダプターと表現](../../docs/design/application/design-guidelines.md#アダプターと表現)は外部表現をアダプターで変換する設計を定めるが、既存の是正項目は確認できなかった。
ドメイン型の構造や JSON タグの変更が外部応答へ直接出る状態を解消する。

## 対象範囲

- `backend/` の本番 HTTP 応答について、直接のドメイン状態のシリアライズと、応答フィールド、入れ子、スライス、マップ、匿名構造体、型エイリアスを経由するドメイン型の使用を調べて是正する。
- 外部の名前、列挙値、null、省略、空配列、空オブジェクト、時刻と数値の表現をアダプターが定め、ドメイン状態から明示的に変換する。
- 応答形の規範は TypeSpec に維持し、型定義、変換、契約テストを照合する。
- 現行リスクの該当行と、変更したモジュールの設計を検証結果へ合わせる。

## 対象外

- 入力型だけで使われるドメイン型の一律分離。応答にも共有される型は対象とする。
- TypeSpec、エンドポイント、列挙値、JSON の省略条件、認可の変更。
- すべての応答型のコード生成、新しい汎用シリアライザー、共有 DTO パッケージの導入。
- `internal/` への移行、ドメイン状態のカプセル化。その後の変更は wi-15826 が扱う。

## 設計

### 起票時の確認箇所

| モジュール | 最初に確認する表現 |
| --- | --- |
| Tenancy | `handlers_http/branding_handler.go` の BrandingResponse が TenantFooterLink を直接持つ |
| Saml | `internal/handlers_http/admin_idp_profile_handler.go` の Profile、service provider の ClaimMappingPolicy |
| Application | `handlers_http/application_provisioning.go` の ClaimMappingRule、`admin_application_handler.go` の SignInRule |
| IdGovernance | `internal/handlers_http/admin_lifecycle_workflow_handler.go` の WorkflowTrigger と WorkflowAction、run と step の列挙値 |
| IdManagement | user の AttributeValue と RequiredAction、agent の kind と status |
| Authentication | federation の ClaimMapping と LinkingPolicy、consent の状態 |
| Provisioning | connection と task の応答、フィールドの入れ子の設定 |

この表は探索の起点であり、ここだけを直して完了しない。
構造検索と import の解決で全 HTTP 応答の対象を列挙し、本項目の設計に応答型、所有するアダプター、元のドメイン型、契約と対応テストの表を記録する。
ハンドラーが応答型を使わずにドメイン状態を返す経路も含める。
外部プロトコル専用の表現型は、その責務を確認し、ドメインの業務状態との混同がない場合だけ除外理由を記録する。

### 表現の所有者と変換

応答型は、その HTTP 契約を提供するモジュールの HTTP アダプターに置く。
同じ外部形を複数の経路が使う場合だけ、そのモジュールのアダプター内で共有する。
別モジュールのドメイン型を参照しているフィールドも、応答を返すアダプターが外部形を定義する。
新しいモジュール間依存、公開範囲、共有ライブラリは追加しない。

ドメインの列挙型は、既存の値を保って外部表現の文字列へ変換する。
入れ子の構造、ポインター、配列、マップは必要な深さまで変換する。
ドメイン型の alias、埋め込み、JSON タグの付け替え、JSON の往復やリフレクションで分離したことにしない。
変換は副作用を持たず、応答に必要な公開契約の値だけを読む。
別モジュールの非公開型への新しい参照は増やさず、既存の公開契約で必要な値を得る。

共有の DTO パッケージを作る案は、外部契約の所有者と変換の意味を離すため採らない。
ドメイン型をそのまま外部へ返し、タグの変更で整える案は、外部形への結合が残るため採らない。
新しい JSON タグと変換の期待値は現行 TypeSpec および既存契約から決め、製品の出力を写しただけのスナップショットにはしない。

### 互換性と証拠

外部形が欠落、空、null、ゼロ値を区別する場合は、その区別をケースとして固定する。
秘密値や内部状態を新しい応答型へ持ち込まない。
変換の追加だけでエラー応答を変えない。
既存の REQ を引くテストで形を固定していない対象は、既存の振る舞いの分類と特性化を先に行い、そのコミットを製品変更から分ける。
仕様に反する既存応答を発見した場合は、その形を本項目のリファクタリングで変更せず、独立した bugfix とその規範を記録する。

## 計画

1. 全応答経路を分類し、対象と除外の表、TypeSpec、契約テストの対応を記録する。
2. 既存の外部形を固定し、アダプターの応答型と変換をモジュールごとに実装する。
3. すべての対象で直接のドメイン型の使用が残らないことを再検索し、契約検査と変更したパッケージのテストを通す。
4. 負債が残る場合は対象の未実施として残し、対象をすべて是正してから現行リスクの該当行を削除する。

## タスク

- [ ] T001 [Inventory] 全本番 HTTP 応答の直接、入れ子、alias と匿名型を分類し、対象と除外の根拠、契約とテストを記録する。
- [ ] T002 [Characterize] 未被覆の既存外部形を固定し、製品変更から分けて記録する。
- [ ] T003 [Adapters] 所有するアダプターに外部応答型と明示的な変換を実装し、列挙、ポインター、入れ子、配列、マップを分離する。
- [ ] T004 [Acceptance] 正式な HTTP 入口でフィールド、null と省略、列挙、配列、時刻、認可と秘密値の不出力を検証する。
- [ ] T005 [Inventory] 全対象を再検索して直接シリアライズとドメイン型のフィールドの残存がないことを確認する。
- [ ] T006 [Docs/Verify] リスクと設計を更新し、契約、パッケージ、変更耐性と最終検査を完了する。

## 検証

- `mise run report-contract-evidence` をタスク定義の引数に従って実行し、変更予定の HTTP 契約と未解析の証拠を着手前に照合する。
- `mise run test-go-one -- <package> <test>` または `mise run test-go-package -- <package>` を、実際の引数を確認して選ぶ。
- 正式な HTTP 入口で TypeSpec に対応するフィールドと値を比較し、変換の omission 条件、列挙値、入れ子の欠落を検出する。
- `mise run check-contract-drift`、`mise run check-status-drift`、`mise run check-boundaries` で公開形と境界の整合を確認する。
- `mise run test-go-mutation -- <変換を変更した package directory>` を実行し、対応するフィールドを落とす、null と省略を入れ替える故障は必要な場合だけ手動注入する。
- `mise run test-go-changed` と `mise run lint-go` を実行し、最終差分の契約証拠と影響範囲に従って `mise run verify` を選ぶ。
- `mise run spec-diff` で規範と TypeSpec の外部契約に変更がないことを確認する。
- 完了記録に全対象と除外の表、実行した HTTP 契約テスト、秘密値の不出力、再検索の結果を残す。

## リスク

フィールド名が同じでも null と省略、nil と空配列の変換で互換性を壊す可能性がある。
既存形と TypeSpec の両方を確認し、これらの境界をテストする。
スライスとマップを浅くコピーすると、内部の状態が外部の応答へ結び付いたままになるため、必要な深さまで明示的に変換する。
入力と応答が同じ型を共有する場合は、入力の解釈を保って応答の型だけを分離する。
