---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-24
priority: p2
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "SP と RP の管理を所有者の公開操作へ移し、Application の関連付け、既存の HTTP 応答、保存の確定境界とイベントを維持する。" }
---

# ApplicationCatalog の SAML と WS-Fed の管理を所有者の公開操作へ通す

## 動機

[[wi-26339-audit-module-boundaries-against-independent-deployment-and-contract-only-coupling]] の棚卸しで、ApplicationCatalog が SAML の SP と WS-Fed の RP の保存型を構築し、Repository を通して保存、更新していることを確認した。
OIDC は OAuth2 の公開管理操作を使うが、SP と RP の規則は Catalog と所有者の管理の入口に分散している。
公開された Repository への非循環 import だけでは、状態の判断が所有者に閉じているとは言えない。

## 対象範囲

- [既知のリスク](../../docs/design/architecture/risks.md)の ApplicationCatalog の SP と RP の更新を解消する。
- Saml と WsFederation に必要な管理と関連付けの公開操作を定め、Catalog は HTTP の変換と編成を行う。
- 作成、設定変更、削除、クレームの更新、Application への関連付けの各経路を照合する。
- 所有者の既存の検証、エラー、イベントと確定の順序を保つ。

## 対象外

- プロトコルと ApplicationCatalog の統合。
- SP と RP の機能追加、クレーム公開ポリシーの変更、作成全体への新たな原子性の追加。
- WS-Trust の再送防止と OAuth2 の汎用認可契約。

## 設計

入力は [ApplicationCatalog](../../docs/modules/application-catalog/README.md)、[Application のカタログ](../../docs/modules/application/features/catalog.md)、[SP の管理](../../docs/modules/saml/features/service-provider.md)、[RP の管理](../../docs/modules/ws-federation/features/relying-party.md)、`backend/applicationcatalog/internal/handlers_http/application_provisioning.go` とする。
[境界の判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)の D2、D3、D6〜D8 を適用する。
Application の管理とプロトコルの実行を別々に変更するシナリオで比べる。

| 案 | 制約への適合と変更の波及 | 判断 |
| --- | --- | --- |
| Catalog が保存型と Repository を使い続ける | 所有者の不変条件の変更が Catalog にも波及し、D2 を満たさない | 不採用 |
| 所有者の公開管理操作を Catalog が利用する | 更新規則を所有者に閉じ、Catalog に表示と入力の変換を残せる | 採用候補 |
| Catalog を組み立て地点と宣言する | 業務の更新を含む実装に対し、宣言だけを広げることになる | 不採用 |
| Saml、WsFederation と Catalog を統合する | 独立したプロトコルの変更も一つの境界へ集める | 不採用 |

着手時に、公開する型と操作のシグネチャを記録し、直接の管理 API と Catalog の検証と作用を比較する。
PostgreSQL の linker とメモリーの ApplicationID の反映を含め、部分的な失敗で残る状態も維持する。
入口間の結果の違いが見つかった場合は、構造変更で黙って揃えず、仕様の変更の要否を判断する。

## 計画

1. Catalog と所有者の管理経路の検証、保存とイベントを対照する。
2. 所有者の公開操作を定め、Catalog から保存型の変更を取り除く。
3. 残る公開パッケージと現在状態の文書を揃える。

## タスク

- [ ] T001 [Design] 各管理操作の契約、拒否と確定境界を確定する。
- [ ] T002 [Refactor] Saml と WsFederation の公開操作と Catalog の呼び出しを実装する。
- [ ] T003 [Verify] 作成、設定変更と関連付けの既存テストを両永続層で確認する。

## 検証

- `mise run test-go-changed`：Catalog と両所有者、その利用側を race 検査する。
- `mise run check-boundaries`：公開操作の宣言と依存を検査する。
- `mise run check-contract-drift` と契約テスト：公開 HTTP の応答と要求を維持する。
- `mise run check-spec` と `mise run check-links`：担当と参照を検査する。

## リスク

Catalog の表示型と所有者の保存型を同じ公開型へまとめると、隠すはずの保存知識が残る。
Application の関連付けの確定境界を拡大せず、公開操作の失敗時の契約を明記する。
