---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-24
priority: p2
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "認可の契約と管理ロールの表示の担当を分け、トークンの発行と失効、管理 API の許可と拒否、永続状態とイベントを維持する。" }
---

# プロトコルをまたぐ認可の契約と管理ロールの表示を OAuth2 から分ける

## 動機

[[wi-26339-audit-module-boundaries-against-independent-deployment-and-contract-only-coupling]] の棚卸しで、OAuth2 の依存先の数より、プロトコルに依存しない判断の配置に問題があることを確認した。
`backend/oauth2/ports/authorizer.go` の AuthZEN 契約を Authorization も使い、`backend/oauth2/internal/token/usecases/role_policies.go` は User、Tenant、SCIM、署名鍵などの管理権限と HTTP の操作を一覧へ写す。
管理権限の追加がトークンの機能スライスの変更になるため、隠す判断と担当を分ける必要がある。

## 対象範囲

- [既知のリスク](../../docs/design/architecture/risks.md)の OAuth2 の汎用認可契約と管理表示の混在を解消する。
- 汎用の認可ポート、管理ロールの一覧、TypeSpec から導く規則と、その規則を利用するモジュールの責任を分ける。
- OAuth2 を利用する ApiTokens と Authorization、組み立て地点の依存を見直す。
- OAuth2 のトークン発行、同意、クライアントと委譲の規則は、引き続き OAuth2 に残す。

## 対象外

- 依存の数だけを理由にした OAuth2 の全面分割。
- AuthZEN のワイヤ契約、管理ロール、認可の規則、トークンの発行条件の変更。
- WS-Trust の再送防止と ApplicationCatalog の SP、RP 管理。

## 設計

入力は [OAuth2](../../docs/modules/oauth2/README.md)、[Authorization](../../docs/modules/authorization/README.md)、[認可設計](../../docs/design/security/authorization.md)、上記のコード、`backend/shared/policy/` と `backend/shared/spec/` とする。
[境界の判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)の D2〜D6 と D8 を適用する。
同じサーバーの内部構成なので、D9 は分割の直接の根拠にしない。

| 案 | 制約への適合と変更の波及 | 判断 |
| --- | --- | --- |
| 現状を維持し、OAuth2 の公開ポートを使う | 技術的には非循環だが、無関係な管理権限の変更が token の機能へ波及する | 不採用 |
| Authorization の責務を認可契約と管理表示まで拡張する | 関係タプルの評価との役割を比べ、OAuth2 への逆依存を取り除けるなら独立した判断にできる | 第一候補 |
| 要求側の小さな認可ポートと管理表示の担当を分ける | 単純な構造適合で足りる部分に使えるが、管理権限の意味の所有者を別に定める必要がある | 比較候補 |
| 業務の権限表を共有ライブラリに移す | TypeSpec から生成する表現と、人が実装する業務判断を混同し、D5 を満たさないおそれがある | 一括移設は不採用 |

管理 API を追加するシナリオと、OAuth のグラントを追加するシナリオで、変更する所有者、公開契約と検証範囲を比較する。
着手時に、仕様バインディングとして共有できる表現と、認可の意味を解釈する実装を分けて、最終の配置と公開シグネチャを確定する。
許可と拒否の意味が変わる案は、この構造変更に含めない。

## 計画

1. 汎用認可の利用側と TypeSpec の規則から、要求と結果の契約を特定する。
2. 管理表示と認可契約の担当を決め、既存の判断を維持して移す。
3. OAuth2、Authorization、ApiTokens と組み立ての依存と文書を揃える。

## タスク

- [ ] T001 [Design] 認可の意味と生成する表現の担当、公開型と操作を確定する。
- [ ] T002 [Refactor] 認可の契約と管理ロールの表示を移す。
- [ ] T003 [Verify] 既存の管理ロール、AuthZEN、トークンと管理 API の拒否を確認する。

## 検証

- `mise run test-go-changed`：変更箇所と利用側の既存テストを race 検査する。
- `mise run check-boundaries`：循環と共有ライブラリへの業務依存を検査する。
- 公開 HTTP の配線を変える場合は `check-contract-drift` と既存の契約テストを選ぶ。
- `mise run check-spec` と `mise run check-links`：規範の担当と参照を検査する。

## リスク

Authorization への一括移設で、認可評価からプロトコルへの循環を作りうる。
管理 API の拒否と、認可の問い合わせそのものの結果を区別して維持する。
