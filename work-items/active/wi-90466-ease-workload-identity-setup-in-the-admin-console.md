---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-07
priority: p2
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/workloadidentity/features/trust-configuration.md, requirement: REQ-WORKLOADIDENTITY-008 }
  - { path: docs/modules/workloadidentity/features/trust-configuration.md, requirement: REQ-WORKLOADIDENTITY-009 }
  - { path: spec/modules/workloadidentity/models.tsp, symbol: IdMagic.Contract.AgentWorkloadBindingCreateRequest }
---

# 管理画面で、ワークロードアイデンティティの設定を試しながら作れるようにする

## 動機

信頼設定と関連付けは管理画面から登録できるが、設定が正しいかを確かめる手段が実際の交換しかない。

| 手間 | 現在の画面 |
| --- | --- |
| 発行者の情報の入力 | 発行者、JWKS の URI、audience、トラストドメインを手で入力する。発行者の `.well-known/openid-configuration` から読み込む手段がない |
| パターンの確認 | 入力したパターンにどの `sub` が一致するかを、登録の前に試せない |
| 重なりの検出 | 既存の関連付けとの重なりは、交換が `ambiguous_match` で拒否されるまで分からない（[設計判断](../../docs/modules/workloadidentity/decisions.md#複数の関連付けに一致した主体は優先順位で選ばず拒否する)の再検討の条件） |
| 関連付けの変更 | 編集できず、削除して作り直す。作り直す間は交換が `no_binding_match` で拒否される |
| 前提の不足 | 対応先の Agent が OAuth2Client に束縛されていないと交換が `agent_unbound` で失敗するが、Agent を選ぶ時点では分からない |

## 対象範囲

- 信頼設定の登録の画面で、発行者の URL から `jwks_uri` を読み込む補助を足す。
- 関連付けの画面で、`sub` の例を入力して、どの関連付けに一致するか（一致なし、一つ、複数）を表示する試行を足す。
- 関連付けの作成で、既存の有効な関連付けと重なりうるパターンを警告する。登録の拒否にするかは設計で決める。
- 関連付けの対応先の Agent とパターンを、削除せずに変更できるようにする。
- Agent の選択肢で、OAuth2Client に束縛されていない Agent を区別して表示する。

## 対象外

- SPIFFE の検証の種別のフォームへの追加。[wi-77559](../done/wi-77559-verify-jwt-svid-against-spiffe-rules.md) が扱う。
- 実際のトークンを貼り付けて検証の全段を試す機能。トークンを管理画面へ貼らせることの是非を別に判断する。

## 設計

### パターンの試行と重なりの検出

照合の規則（ワイルドカードは一つの階層の中だけで一致する）は、交換と同じ関数を使う。
試行は、サーバーの管理 API（たとえば `POST .../bindings:match`）として置き、画面に照合の規則を複製しない。

glob の二つのパターンが重なるかは、一般には例の `sub` なしに判定しにくい。
候補は次のとおり。着手時に決める。

| 案 | 内容 |
| --- | --- |
| A | 重なりを警告だけにし、登録は許す。判定は、一方のパターンを例として他方に照合する近似とする |
| B | 重なりを正確に判定し、登録を拒否する。設計判断の再検討の条件を満たすので、判断の文書も更新する |

### 発行者の情報の読み込み

読み込みは、サーバーが発行者の URL へ取得する。
管理者の入力した URL へサーバーから接続するので、既存の JWKS の取得と同じ SSRF の対策（https のみ、内部アドレスへの接続の拒否）を通す。

### 関連付けの変更

変更の操作を足すと、`AgentWorkloadBindingLifecycle` に状態を変えない更新の事象が増える。
パターンの一意性の制約は、更新でも作成と同じく確かめる。

## 計画

1. 重なりの扱い（案 A か B）を決め、仕様と TypeSpec を更新する。
2. 照合の試行と、関連付けの更新の管理 API を足す。
3. 画面の補助を足す。

## タスク

- [ ] T001 [Spec] 照合の試行、関連付けの更新、重なりの扱い、発行者の情報の読み込みを仕様と TypeSpec に書く。
- [ ] T002 [Acceptance] 関連付けの更新と照合の試行を RED にする。
- [ ] T003 [App] 管理 API を実装する。
- [ ] T004 [UI] 発行者の情報の読み込み、照合の試行、重なりの警告、関連付けの編集、束縛のない Agent の表示を足す。
- [ ] T005 [Verify] 検証を実行する。

## 検証

- `mise run test-go-package -- ./backend/workloadidentity/...`。
- `mise run check-schema`：TypeSpec の変更。
- 管理画面のテストと、関連付けを作って試す流れの `test-ui-e2e-file`（サンドボックスの外で実行する）。

## リスク

- 発行者の情報の読み込みは、サーバーから管理者の指定した URL へ接続する。SSRF の対策を JWKS の取得と共有し、別に書かない。
- 重なりの警告が近似だと、警告の出ない重なりが残る。画面の文言で近似であることを示す。
