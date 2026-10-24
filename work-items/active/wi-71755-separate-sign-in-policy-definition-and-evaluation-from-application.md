---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-24
priority: p2
depends_on: []
change_kind: refactor
spec_impact:
  kind: none
  reason: "サインイン条件の担当を分け、アプリごとの上書きとテナント既定、MFA、信頼済みデバイス、再認証とネットワーク条件の許可と拒否、保存状態、イベントを維持する。"
---

# サインインポリシーの定義と評価を Application から分ける

## 動機

Application はアプリのメタデータと割り当てに加え、サインイン条件の定義と評価を所有する。
しかし、テナントの既定のポリシーはアプリ以外のサインインにも使われ、Authentication が Application の公開型と評価器を利用している。
アプリ単位で設定できることだけを所有の根拠にせず、認証強度、再認証の期限、ネットワーク条件を変更する担当を独立させたい。

## 対象範囲

- [サインインポリシー](../../docs/modules/application/features/sign-in-policy.md)の定義、検証、既定と上書きの選択、評価の担当を分ける。
- Application の割り当ての関門と、Authentication の認証および追加認証との契約を整理する。
- `application_sign_in_policies` と `tenant_default_sign_in_policies` の所有者、Application の削除と同時に消す保証を比較する。
- OIDC、SAML、WS-Fed、アカウント画面の既存の許可、拒否、追加認証の結果とイベントを保つ。

## 対象外

- MFA、再認証、ネットワーク条件、既定と上書きの意味の変更。
- ACL、関係タプル、管理ロール、OAuth のグラントと同意の再設計。
- 「ポリシー」という語だけを理由に、すべての認可とサインイン条件を一つへ集約すること。

## 設計

入力は Application の [責務](../../docs/modules/application/README.md)、[サインインポリシー](../../docs/modules/application/features/sign-in-policy.md)、[上書きの判断](../../docs/modules/application/decisions.md)、[認可設計](../../docs/design/security/authorization.md)、[テーブルの所有](../../docs/design/data/database.md#所有と書き込みの境界)とする。
実装の起点は `backend/application/domain/sign_in_policy.go`、`backend/application/usecases/sign_in_policy.go`、`backend/application/usecases/access_gate.go`、Authentication の `enforceDefaultSignInPolicy` とアカウントの認証器解除である。

[境界を選ぶ判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)の D1、D2、D4、D6、D7、D8 を適用する。
同じサーバーの内部の再配置なので、別々の配布物を理由とする D9 は分割の根拠にしない。

| 案 | 隠す判断と変更の波及 | 比較する点 |
| --- | --- | --- |
| Application に残す | アプリの関門と同じモジュールに評価を持つが、アカウントのサインインも Application に依存する | 現状を比較基準にする。テナント既定の変更で必要になる知識を確認する |
| SignInPolicy を独立させる | 条件の定義、実効規則の選択、検証と評価を隠し、Application は適用を要求する | 第一候補。設定の保存と評価を一緒に移す案と、評価だけを移す案で、公開する内部知識と更新の原子性を比較する |
| Authentication に集約する | 認証事実と条件の判断が近くなる | アプリの割り当てからの利用で循環しないか、認証の実行と条件の判断を独立に変更できるかを比較する |
| AccessControl に集約する | 認可とサインイン条件を一つの担当に置く | 資源への権限と、認証強度および追加認証の状態を同じ理由で変更するかを確認する。名称の類似だけでは採用しない |

着手時に、アプリの表示名だけを変える場合、MFA の適用条件を変える場合、テナント既定をアカウント画面へ適用する場合の変更を各案で列挙する。
保存先の移管、アプリの存在確認、削除時の cascade、同じトランザクションへの参加とイベントの順序を比較してから、公開型と操作を確定する。
認証結果は読み取り専用の事実として渡し、Application やポリシーの評価器から Authentication の非公開実装へ依存しない。
既存の仕様が保証しない条件を追加する判断は、この構造変更に含めない。

## 計画

1. 設定の保存、評価、適用と認証の実行の現在の呼び出し元と作用を照合する。
2. 上表の候補と変更シナリオを比較し、所有者と公開契約を決める。
3. 既存の結果を固定したまま移設し、依存、テーブル所有と文書を合わせる。

## タスク

- [ ] T001 [Design] ポリシーの所有者、公開契約、保存と削除の原子性を比較して確定する。
- [ ] T002 [Refactor] 定義、検証、選択と評価を採用した担当へ移す。
- [ ] T003 [Verify] 各プロトコルとアカウントの成功、拒否、追加認証、解除の制限とイベントを確認する。

## 検証

- 既存のサインインポリシーと Application の関門、Authentication のテストを維持する。
- テナント既定の配線、アプリの上書きの選択、認証事実の解釈を誤る故障を、その結果を観測できる最小の境界で検出する。
- `mise run test-go-changed`、`check-boundaries`、`check-boundary-debt-ratchet` で利用側と循環の不在を確認する。
- 保存先の所有者を変える場合は PostgreSQL の既存契約と削除時の残存状態を検査する。
- 公開 HTTP の組み立てを変える場合は契約証拠を着手時に照合し、既存の経路を実行する。
- `mise run check-spec`、`check-work-items`、`check-links` で参照と宣言を合わせる。

## リスク

評価器だけを移し、規則と保存型が Application に残ると、公開する内部知識を増やすだけになりうる。
ポリシーの所有者から Application の存在確認を呼び、Application も評価器を呼ぶと循環するため、要求する側の契約と組み立てで解決する案を比較する。
