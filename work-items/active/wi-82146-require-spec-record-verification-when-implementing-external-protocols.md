---
status: pending
authors: [tn]
risk: low
created_at: 2026-11-28
priority: p2
group: development-process
depends_on: []
change_kind: docs
spec_impact: { kind: none, reason: "テスト方針の文書に、外部の標準や製品への対応を実装する作業で一次資料の例による検証を同じ項目で行うことと、記録の再生、模倣、既存の単体テストの分担を書き足す。規範シナリオ、標準の宣言、TypeSpec シンボルを追加も変更もしない。" }
---

# 外部の標準や製品への対応を実装するときに、一次資料の例による検証を同じ作業項目で行うと定める

## 動機

消費する側の契約を記録の再生と模倣で確かめる作業は、これまで実装の後に別の項目で行ってきた。
SCIM（[wi-422](../done/wi-422-external-contract-and-failure-verification.md)）、上流のトークンエンドポイント（[wi-27706](../done/wi-27706-verify-upstream-oauth-token-endpoints-with-recordings.md)）、上流の OIDC IdP（[wi-65625](../done/wi-65625-verify-upstream-oidc-identity-providers-with-recordings.md)）、上流の MCP サーバー（[wi-49024](../done/wi-49024-verify-upstream-mcp-servers-with-recordings.md)）の四件である。
四件はどれも、実装済みの規範違反を一つ以上見つけた。

見つけた誤りは、期待値を実装者ではなく一次資料が決めたことから来ている。
既存の単体テストは、解析器を書いた人が相手の応答も手で書いていた。
そのため同じ思い込みがテストと実装の両方に入り、テストが通っていた。
wi-49024 では、空文字列の `nextCursor` を一覧の終わりと読む誤りが、記録を作るためにページングの節を転記したところで見つかった。

[テスト方針の外部契約と設定の利用](../../docs/development/testing.md#外部契約と設定の利用)は記録と模倣の作り方を定めている。
一方で、いつ作るかと、既存の単体テストとどう分担するかは定めていない（2026-11-28 に確かめた）。
その結果、次の二つが起きた。

- 誤りは実装の時点で見つけられたのに、後追いの項目まで残った。
- 後追いの項目が既存の単体テストと照らさずに模倣の検査を足し、同じ分岐を二度検査するテストができた。整理は [wi-80931](wi-80931-consolidate-record-and-twin-tests-with-existing-unit-tests.md) が扱う。

## 対象範囲

- [テスト方針の外部契約と設定の利用](../../docs/development/testing.md#外部契約と設定の利用)に次を書き足す。
  - 外部の標準や製品への対応を新しく実装する作業と、対応する版を上げる作業は、同じ作業項目の中で一次資料の例による検証を置く。
    要求を送る側なら記録と再生を、要求を受ける側なら一次資料の要求の例を入力にした受け入れの検査を置く。
    どちらも期待値は一次資料の文面から書く。
  - 記録は実装の前に作る。
    一次資料の節を転記する作業そのものが、実装の前提の誤りを見つける手順だからである。
  - 着手時に、触れるクライアントの既存の単体テストを振る舞いごとに並べる。
    そのうえで、再生と模倣は既存のテストが扱えない部分だけを埋める。
  - 再生は、仕様の形が意味を持つ往復に絞る。
    成功の本文、カーソル、ストリームのメッセージ、本文のあるエラーがこれにあたる。
    状態コードだけで決まる往復は、既存の単体テストに任せる。
  - 模倣は、状態をまたぐ検査（ページの列、変更の通知、鍵の回転、組み合わせた障害）を既存のテストが扱えないときだけ置く。
    一つの応答で足りる分岐は手書きの関数で一度だけ検査する、という既存の規則との関係を明記する。
- [外部制約と利用側の成立](../../docs/development/specification-first-workflow.md#外部制約と利用側の成立)から、上の規則へのリンクを一文で足す。
  `implement-work-item` スキルは手順 2 と 4 からこの二つの節を参照しているので、スキルは変えない。

## 対象外

- 既存のテストの整理。[wi-80931](wi-80931-consolidate-record-and-twin-tests-with-existing-unit-tests.md) が、この項目で書く分担に従って行う。
- 残りの外部の相手への適用。[wi-94480](wi-94480-decide-recordings-for-the-remaining-external-counterparts.md) と [wi-21837](wi-21837-verify-vault-and-openbao-transit-with-recordings.md) が、この規則に従って行う。
- 実在の相手への常時の接続と、CI での実物の起動。

## 設計

規則はテスト方針の既存の節に足し、新しい節は作らない。
記録の作り方と同じ節にあれば、記録を作る人がいつ作るかと分担も同時に読めるからである。

着手の手順（`implement-work-item` スキル）へ直接書く案は採らない。
スキルは手順の順序を定めるもので、テストの水準と分担はテスト方針が所有する（[一次情報源の割り当て](../../docs/formats/documentation-guide.md#一次情報源の割り当て)）。
スキルはすでに[外部制約と利用側の成立](../../docs/development/specification-first-workflow.md#外部制約と利用側の成立)と[外部契約と設定の利用](../../docs/development/testing.md#外部契約と設定の利用)を名指しで読ませている。

作業項目の形式に「記録」の欄を足して機械で求める案も採らない。
有無だけを検査すると、中身のない記録を作らせることになる。
これは[既存製品との比較](../../docs/formats/work-item-format.md#既存製品との比較)を検査しない理由と同じである。

## 計画

1. テスト方針の節に、時期、対象、分担の規則を書き足す。
2. ワークフローの節からリンクを足す。
3. `japanese-tech-writing` スキルで推敲する。

## タスク

- [ ] T001 [Docs] テスト方針の外部契約と設定の利用に、時期、対象、分担の規則を書き足す。
- [ ] T002 [Docs] 外部制約と利用側の成立からリンクを足す。
- [ ] T003 [Verify] `mise run check-links` と `mise run check-work-items` を通す。

## 検証

- `mise run check-links`、`mise run check-work-items`。
- 文書だけの変更なので、ほかの検査は実際の差分から選ぶ。

## リスク

規則が重く読まれると、小さな版の追従でも記録の作成を求めることになる。
対象を「一次資料に要求と応答の例がある相手」とし、例がない相手は [wi-94480](wi-94480-decide-recordings-for-the-remaining-external-counterparts.md) と同じく、費用と見つけられる故障を比べて採否を決めると書く。
