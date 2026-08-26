---
depends_on: []
status: pending
authors: [tn]
risk: medium
created_at: 2026-08-27
priority: p2
change_kind: docs
spec_impact: { kind: none, reason: "サブドメイン分類、用語定義、Aggregate 境界の判断を正本文書へ書くが、規範シナリオ、規範 ID、TypeSpec シンボルを追加も変更もしない。" }
---

# サブドメインの分類と Aggregate の境界を仕様として持つ

## Motivation

戦略的な Domain-Driven Design は厚く入っている。`docs/contexts/`、`spec/contexts/`、`backend/<context>` の三つの木が同じ名前で対応し、Context Map は関係の種類まで型付けされ、`docs/glossary.md` は「全体の用語を Context の用語集が狭めることがあり、狭めた先では狭めた定義が読み方になる。別のものを指すようになったならそれは同じ語ではない」という Published Language の中核を正しく形にしている。

一方で戦術的なパターンの規範は事実上存在しない。`Aggregate` という語は `docs/contexts/jobs/decisions.md`、`docs/database.md`、`docs/contexts/identity-management/internals.md` などで使われているが、どこにも定義されていない。`docs/structure.md` の `domain/` の説明に「エンティティ、値オブジェクト、状態遷移、純粋な検証」という 1 行があるだけで、Aggregate の境界をどこに引くか、一貫性の境界がトランザクションの境界とどう対応するか、Repository は Aggregate 単位かは書かれていない。`docs/database.md` の `tenant_id` 保持区分は「テナントに属する Aggregate は `tenant_id` を持つ」という規則を Aggregate の概念に依存させているのに、その概念の定義が体系の外にある。

戦略面にも欠落がある。20 個の Bounded Context が索引表の中で完全に対等に並んでおり、どれが競争優位の中心でどれが必要だが差別化しない領域かが書かれていない。Core と Supporting と Generic の区別が無いと、設計投資の配分も、自作するか既製品に委ねるかの判断も、その都度の裁量になる。実際には「AEAD と鍵セットの処理は Tink に委ね、nonce や認証タグの組み立てを自作しない」（`docs/database.md`）という優れた判断を既にしているが、その根拠が一般化されていないため、次の同種の判断で再利用できない。

Context Map の宣言と実 import グラフの乖離（wi-414 で記録する）は、境界をあとから引いた徴候でもある。分類と境界の見直しは、その乖離をどう解消するかの判断材料になる。

Anti-Corruption Layer についても、Context Map に Sourcing と WorkloadIdentity の 2 か所で現れるのに、ACL をどこにどう置くかの規約が無い。アダプターの命名規則 `<role>_<technology>` に ACL の役割が含まれていないため、実装がどこにあるかは読んでみないと分からない。

## Scope

- **サブドメインの分類**：全 Bounded Context を Core、Supporting、Generic に分類し、`docs/README.md` の索引表へ列として加える。分類の理由は各 Context の `decisions.md` が持つ。
- **分類の使い道**：分類が何を左右するか（設計投資の厚み、自作と委譲の判断、検証の強度）を書く。分類が何も変えないなら書く意味がないため、ここを曖昧にしない。
- **Aggregate の定義**：`docs/glossary.md` に Aggregate を、この製品で意味が固定される語として定義する。一貫性の境界であること、トランザクションの境界との対応、テナント境界との関係を含める。
- **Aggregate の境界の記録先**：各 Context のどのファイルが Aggregate の境界を持つかを定める。
- **Repository の粒度**：Repository が Aggregate 単位であるかどうかを決め、現状と食い違う場合はその理由を書く。
- **ACL の配置規約**：Anti-Corruption Layer をどのディレクトリに、どの命名で置くかを定め、Sourcing と WorkloadIdentity の現状を照合する。
- **境界の見直しの手順**：Context の境界を引き直す必要が生じたときに何をするかを、`DEVELOPMENT.md` のループのどこに接続するか決める。

## Out of Scope

- Context の実際の分割・統合。本 work item は分類と定義を持つところまでを担う。
- wi-414 が記録する境界負債の解消。
- Aggregate の境界に合わせたコードの再編成。
- Event Storming の実施そのもの。手法として採るかどうかは Design で論じるが、実施は分類の結果を見て判断する。

## Design

サブドメインの分類は `docs/README.md` の既存の索引表へ列を足す形にする。別ファイルにしないのは、Context の一覧が 2 か所に分かれると必ず片方が古くなるからである。分類の理由を索引表に書かないのは、理由は Context ごとに違い、表のセルに収まらないためである。理由はその Context の `decisions.md` が持つ。

分類の判断基準には 2 つの候補がある。Core Domain Chart（モデルの複雑さと事業上の差別化の二軸）と、Wardley Map（進化の度合いと価値連鎖上の位置）である。採るのは前者である。後者は市場と調達の判断に強いが、この製品にとって差し迫っているのは「どの Context に設計と検証の厚みを配分するか」であり、そこには二軸の分類で足りる。Wardley Map は判断の入力を大きく増やす割に、出力が同じところへ収束する。

Aggregate の定義を `docs/glossary.md`（全体の用語集）に置くのは、この語が既に複数の Context の文書で使われており、Context ごとに違う意味を持つべきではないからである。同じ理由で、`docs/database.md` の `tenant_id` 保持区分が依存している概念が体系の内側へ入る。

Aggregate の境界をどこに記録するかには 2 案ある。各 Context の `glossary.md` に載せる案と、`decisions.md` に判断として書く案である。採るのは `decisions.md` である。境界は「なぜここで切ったか」を伴わなければ次の変更で守られず、`SPECIFICATION_FORMAT.md` は理由の無い項目を「言い換えた規則であって判断ではない」として退けている。ただし Aggregate の名前そのものは `glossary.md` に載せ、定義と判断を分ける。

Event Storming は境界の引き直しが必要と判断された場合の手順として `DEVELOPMENT.md` から参照する形を採り、常時の工程には入れない。人間 1 人とエージェントで行う形式は成立するが、境界が動かない限り費用に見合わない。

未解決の論点として、Repository が現状 Aggregate 単位になっているかどうかは調査していない。なっていない場合、規範を現状に合わせるか現状を規範に合わせるかの判断が要る。実装着手前に確定させる。

## Plan

1. 全 Context を Core、Supporting、Generic のいずれかに仮分類し、判断が割れるものを洗い出す。
2. 分類が何を左右するかを決める。左右するものが無ければ、分類自体を Out of Scope へ移す。
3. `docs/glossary.md` に Aggregate を定義し、`docs/database.md` の保持区分がその定義で読めることを確認する。
4. 各 Context の Aggregate を洗い出し、境界の判断を `decisions.md` へ書く。既に暗黙に守られている境界を記述として書き、守られていない箇所は負債として記録する。
5. Repository の粒度と ACL の配置を現状と照合し、規範を決める。
6. 索引表へ分類の列を足し、境界の見直しの手順を `DEVELOPMENT.md` へ接続する。

## Tasks

- [ ] T001 [Baseline] 全 Context の仮分類、Aggregate の洗い出し、Repository の粒度と ACL の実装位置の現状調査を行う。
- [ ] T002 [Spec] 分類が左右するものを決め、`docs/README.md` の索引表へ列を足す。
- [ ] T003 [Spec] `docs/glossary.md` に Aggregate を定義する。
- [ ] T004 [Spec] 各 Context の `decisions.md` へ Aggregate の境界と分類の理由を書く。
- [ ] T005 [Spec] Repository の粒度と ACL の配置規約を定め、現状との差を負債として記録する。
- [ ] T006 [Verify] `mise run check-spec` を通し、Aggregate を使う既存の記述が新しい定義で読めることを確認する。

## Verification

- `mise run check-spec` が通る。
- `docs/database.md` の `tenant_id` 保持区分と `docs/contexts/jobs/decisions.md` の「別テナントの Aggregate の識別子」という記述が、新しい定義と矛盾しない。
- `docs/README.md` の索引表の全行に分類が入っている。
- `mise run verify`

## Risk Notes

サブドメインの分類は主観が入りやすく、書いた直後は正しく見えても、根拠が書かれていなければ半年後には誰も守らない。分類そのものより「分類が何を左右するか」を先に決めることで、飾りになることを防ぐ。左右するものを決められないなら、分類は書かないほうがよい。

Aggregate の境界を全 Context について一度に書こうとすると、実態の調査が終わらないまま推測で書くことになりやすい。Core に分類した Context から順に書き、Generic の Context は名前の列挙にとどめてよい。

Repository の粒度を現状と異なる規範に決めると、広範な改修を暗黙に約束することになる。現状に合わせる判断も、理由を書けるなら正当である。
