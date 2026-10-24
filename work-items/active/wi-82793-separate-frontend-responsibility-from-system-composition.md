---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-24
priority: p2
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "画面と組み立て地点の責務の宣言を分け、ログイン、同意、管理とアカウントの HTTP 契約、永続状態とイベントを維持する。" }
---

# フロントエンドの責務を System の組み立て地点から分ける

## 動機

[[wi-26339-audit-module-boundaries-against-independent-deployment-and-contract-only-coupling]] の棚卸しで、[責務表](../../docs/modules/README.md#基盤)の System 行が `frontend/` 全体を組み立て地点として宣言していることを確認した。
画面の入力、認可トランザクションの表示とポータルの OIDC RP は、結線以上の判断を含む。
別のイメージとブラウザーのキャッシュも考慮し、サーバーの組み立てと画面の変更理由を分ける必要がある。

## 対象範囲

- [既知のリスク](../../docs/design/architecture/risks.md)の System の責務の混在を解消する。
- フロントエンドの責務、公開 HTTP 契約、配置と仕様の担当を宣言し、System の組み立て地点をバックエンドの結線へ限定する。
- 責務表と境界検査の分類を揃え、Go パッケージを持たないモジュールを表す形式も確定する。
- System の機能仕様のうち、画面の保証とサーバーの入口の保証の担当を整理する。

## 対象外

- 画面の再設計、URL、認証、認可、公開 API の変更。
- フロントエンドを別のリリース系列にする運用上の保証の追加。

## 設計

入力は [System](../../docs/modules/system/README.md)、[フロントエンド設計](../../docs/design/application/frontend.md#全体構成)、`frontend/src/api/core.ts`、`frontend/src/features/`、両方の Dockerfile とする。
[境界の判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)の D8 と D9 を適用する。
フロントエンドは HTTP の契約を利用し、サーバーの内部型とデータベースを利用しない。
同じリリースの別イメージであることだけでは、独立したリリース系列だと判断しない。

| 案 | 制約への適合と変更の波及 | 判断 |
| --- | --- | --- |
| System の全体を組み立て地点のままにする | 表示判断を結線として扱い続け、画面変更で組み立て地点の説明も変わる | 不採用 |
| Frontend を別モジュールとして宣言し、System に結線と共通の入口を残す | 表示と OIDC RP の内部判断を分離し、API の内部変更を HTTP 契約で隔離できる | 採用候補 |
| 全画面を各バックエンドモジュールの実装として宣言する | サーバーと画面の配置と配布を再び一つの宣言に混ぜる | 不採用 |

着手時に、System の規範 ID を維持した配置変更で済むかと、境界検査の表形式を確定する。
型と形式の変更は構造変更として先に検証し、画面の意味を変更しない。

## 計画

1. System の仕様を、画面、サーバーの入口、結線に分類する。
2. 責務表と境界検査の形式を揃え、担当文書へ配置する。
3. 既存の利用経路と境界検査で構造変更を確認する。

## タスク

- [ ] T001 [Design] 配布、契約と仕様の担当の比較を確定する。
- [ ] T002 [Refactor] 責務表、境界検査と文書の担当を分ける。
- [ ] T003 [Verify] 境界、仕様、リンクと影響する既存の画面テストを確認する。

## 検証

- `mise run check-boundaries`、`mise run check-spec`、`mise run check-links`：分類と仕様の参照を検査する。
- 境界検査を変えたら `mise run test-tools-file -- <対象ファイル>` で未宣言と重複分類の拒否を確認する。
- 画面のコードを移す場合は該当する `test-ui-unit-file` と `test-ui-e2e-file` を選ぶ。

## リスク

仕様を移動するだけで規範 ID を変更すると参照を失うため、識別子を保つ。
別の成果物として構築できることと、任意のバージョンの組み合わせをサポートすることを区別する。
