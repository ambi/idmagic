---
depends_on: []
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-08-27
priority: p2
change_kind: docs
evidence_policy: risk-based-v2
initial_context:
  specification:
    - docs/README.md
    - docs/glossary.md
    - docs/structure.md
    - docs/design-rules.md
    - docs/database.md
    - docs/product-overview.md
    - docs/development/specification-first-workflow.md
    - SPECIFICATION_FORMAT.md
  source:
    - backend
    - tools/check/src/specification-doc.ts
    - tools/workspace/src/check-workspace.ts
  tests:
    - tools/check/src
  stop_before_reading:
    - frontend
    - spec
    - infra
spec_impact: { kind: none, reason: "サブドメイン分類、用語定義、Aggregate 境界の判断を正本文書へ書くが、規範シナリオ、規範 ID、TypeSpec シンボルを追加も変更もしない。" }
---

# サブドメインの分類と Aggregate の境界を仕様として持つ

## Motivation

戦略的な Domain-Driven Design は厚く入っている。`docs/contexts/`、`spec/contexts/`、`backend/<context>` の三つの木が同じ名前で対応し、Context Map は関係の種類まで型付けされ、`docs/glossary.md` は「全体の用語を Context の用語集が狭めることがあり、狭めた先では狭めた定義が読み方になる。別のものを指すようになったならそれは同じ語ではない」という Published Language の中核を正しく形にしている。

一方で戦術的なパターンの規範は事実上存在しない。`Aggregate` という語は `docs/contexts/jobs/decisions.md`、`docs/database.md`、`docs/contexts/identity-management/internals.md` などで使われているが、どこにも定義されていない。`docs/structure.md` の `domain/` の説明に「エンティティ、値オブジェクト、状態遷移、純粋な検証」という 1 行があるだけで、Aggregate の境界をどこに引くか、一貫性の境界がトランザクションの境界とどう対応するか、Repository は Aggregate 単位かは書かれていない。`docs/database.md` の `tenant_id` 保持区分は「テナントに属する Aggregate は `tenant_id` を持つ」という規則を Aggregate の概念に依存させているのに、その概念の定義が体系の外にある。

戦略面にも欠落がある。21 個の Bounded Context が索引表の中で完全に対等に並んでおり、どれが競争優位の中心でどれが必要だが差別化しない領域かが書かれていない。Core と Supporting と Generic の区別が無いと、設計投資の配分も、自作するか既製品に委ねるかの判断も、その都度の裁量になる。実際には「AEAD と鍵セットの処理は Tink に委ね、nonce や認証タグの組み立てを自作しない」（`docs/database.md`）という優れた判断を既にしているが、その根拠が一般化されていないため、次の同種の判断で再利用できない。

Context Map の宣言と実 import グラフの乖離（wi-414 で記録する）は、境界をあとから引いた徴候でもある。分類と境界の見直しは、その乖離をどう解消するかの判断材料になる。

Anti-Corruption Layer についても、Context Map に Sourcing と WorkloadIdentity の 2 か所で現れるのに、ACL をどこにどう置くかの規約が無い。アダプターの命名規則 `<role>_<technology>` に ACL の役割が含まれていないため、実装がどこにあるかは読んでみないと分からない。

## Scope

- **サブドメインの分類**：全 Bounded Context を Core、Supporting、Generic に分類し、`docs/README.md` の索引表へ列として加える。分類の理由は各 Context の `decisions.md` が持つ。
- **分類の使い道**：分類が何を左右するか（設計投資の厚み、自作と委譲の判断、検証の強度）を書く。分類が何も変えないなら書く意味がないため、ここを曖昧にしない。
- **Aggregate の定義**：`docs/glossary.md` に Aggregate を、この製品で意味が固定される語として定義する。一貫性の境界であること、トランザクションの境界との対応、テナント境界との関係を含める。
- **Aggregate の境界の記録先**：各 Context のどのファイルが Aggregate の境界を持つかを定める。
- **Repository の粒度**：Repository が Aggregate 単位であるかどうかを決め、現状と食い違う場合はその理由を書く。
- **ACL の配置規約**：Anti-Corruption Layer をどのディレクトリに、どの命名で置くかを定め、Sourcing と WorkloadIdentity の現状を照合する。
- **境界の見直しの手順**：Context の境界を引き直す必要が生じたときに何をするかを、`docs/development/specification-first-workflow.md` のループのどこに接続するか決める。

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

Event Storming は境界の引き直しが必要と判断された場合の手順として `docs/development/specification-first-workflow.md` から参照する形を採り、常時の工程には入れない。人間 1 人とエージェントで行う形式は成立するが、境界が動かない限り費用に見合わない。

### 分類が左右するもの（着手前に確定）

分類が左右するのは次の 3 つである。

- **自作と委譲**：Generic に分類した領域では、標準実装またはライブラリへ委ねられる部分を自作しない。`docs/database.md` の「AEAD と鍵セットの処理は Tink に委ねる」がこの規則の既存の実例である。
- **モデリングと文書の厚み**：Core の Context は固有の語彙を持ち、`decisions.md` と `internals.md` に判断と機構を書く。Generic の Context は CRUD 相当の形のまま置いてよく、書くものが無いのに `internals.md` を作らない。
- **統合したときにどちらが歩み寄るか**：Core と非 Core が接するとき、モデルを曲げるのは非 Core の側である。Core は相手の都合に合わせて語彙を変えない。

**検証の強度は左右しない。** これは Scope で候補に挙げていたが採らない。検証の強度を決めるのは work item の `risk` と `docs/threat-model.md` であって、サブドメインの分類ではない。両者を結ぶと、Supporting に分類したテナント境界や認証の検証を弱めてよいという読みが成立してしまう。実際には Tenancy は Supporting でありながら、その拒否は最も強く検証されなければならない。分類は「重要度の順位」ではなく「差別化の所在」であり、この 2 つを混ぜないために否定形を明示して書く。

### 分類の結果

| 区分 | Context |
|---|---|
| Core | IdManagement, Authentication, OAuth2, Authorization, WorkloadIdentity |
| Supporting | System, Tenancy, IdGovernance, Application, Audit, ClaimMapping, Provisioning, Sourcing, Seeding, SigningKeys, SharedSignals |
| Generic | ApiTokens, Jobs, DataKeys, Saml, WsFederation |

判断が割れたのは Tenancy と Authentication である。Tenancy はすべての Aggregate が属する境界を持ち、壊れたときの被害は最大だが、マルチテナントであること自体は競合との差にならないため Supporting とする。ここで Core にすると、分類が差別化の所在ではなく重要度の順位になり、上の「検証の強度は左右しない」が破れる。Authentication は逆に、標準が決めるのはプロトコルだけで、ステップアップ、信頼済みデバイス、登録強制、復旧はこの製品が定義するポリシーであり、委譲できる相手が存在しないため Core とする。

### Repository の粒度（未解決だった論点の決着）

調査の結果、現状はほぼ Aggregate root 単位である。`TenantRepository`、`UserRepository`、`GroupRepository`、`AgentRepository`、`OAuth2ClientRepository`、`ConsentRepository`、`JobRepository`、`DataKeyRepository`、`LifecycleWorkflowRepository` などがいずれも 1 つの root に対応する。したがって規範は現状に合わせ、「Repository は Aggregate root 単位」と定める。現状と異なる規範を選ぶと、広範な改修を文書で暗黙に約束することになる。

同時に、`Repository` と `Store` の呼び分けという未文書の規約が実在することが分かった。`Repository` は Aggregate を永続化し、`Store` は不透明な鍵で引く短命なプロトコル状態（`SessionStore`、`PARStore`、`DeviceCodeStore`、`AuthorizationCodeStore`、各 `ReplayStore`）と、Aggregate に属さない資産（`ApplicationIconStore`、`TenantBrandingAssetStore`、`UserCSVArtifactStore`）を持つ。この規約は `docs/design-rules.md` に書く。合わない名前は 2 件あり、負債として同じ場所に記録する：`signingkeys/ports/KeyStore` は `SigningKey` という Aggregate を持ちながら `Store` を名乗り、`sourcing/scim/ports/ScimRepository` は `ScimUserRef` と `ScimGroupRef` の 2 つを 1 つの Repository で扱う。

### Aggregate の名前をどこに書くか（Design の修正）

当初案は「名前は各 Context の `glossary.md`、境界の判断は `decisions.md`」だった。前半は残すが、`decisions.md` に Aggregate root の一覧を書くことはしない。`SPECIFICATION_FORMAT.md` は `decisions.md` と `internals.md` が目録を持つことを退けており、理由の無い一覧はそこでいう「言い換えた規則」そのものだからである。`decisions.md` に書くのは、切り方に判断が要った境界だけとする。判断の要らなかった Context には書かない。

### ACL の配置規約

Context Map は ACL を Sourcing と WorkloadIdentity の 2 か所で宣言しているが、実装は既に一貫した形を持っている。相手 Context の語彙を自分のポートへ翻訳するアダプターを `<role>_<相手 Context>` という名前のパッケージに置く形で、`sourcing/scim/source_idmanagement`、`provisioning/source_idmanagement`、`authorization/principals_idmanagement`、`oauth2/policy_tenancy` の 4 件が該当する。置き場所を決めているのは「翻訳する側」ではなく「Context Map が依存を許す側」である。`provisioning/source_idmanagement` は下流である Provisioning に立ち、`sourcing/scim/source_idmanagement` は逆に上流である Sourcing に立って IdManagement のポートを満たす。IdManagement が Sourcing を知ることを避けるためであり、この非対称は規約として書かないと読んだだけでは分からない。既存のアダプター命名規則 `<role>_<technology>` の `technology` の位置に Context 名が入る形なので、規則自体は増やさず `docs/structure.md` の同じ段落を広げる。

### 分類を飾りにしないための検査

索引表の列は、放置すれば新しい Context が分類なしで追加される。`tools/check/src/subdomain-classification.ts` に、`docs/README.md` の Context 索引表が `docs/contexts/` の全ディレクトリを 1 行ずつ持ち、各行が `Core` / `Supporting` / `Generic` のいずれかを宣言していることを確かめる純関数を置き、`check-workspace --documents` から呼ぶ。README の本文とディレクトリ一覧は引数として入り、この関数はファイルシステムを読まない。これが本 work item の Acceptance と Unit の RED を与える。

## Plan

1. 分類、Aggregate の洗い出し、Repository の粒度、ACL の実装位置を現状調査する（完了、上の Design に記録）。
2. `subdomain-classification` の検査を書き、Unit RED と Acceptance RED を確認する。
3. `docs/glossary.md` に Aggregate を定義し、`docs/database.md` の保持区分がその定義で読めることを確認する。
4. `docs/design-rules.md` に、Aggregate 境界、Repository の粒度、`Repository` と `Store` の呼び分け、分類が左右するものを書く。
5. `docs/structure.md` に ACL の配置規約を書く。
6. `docs/README.md` の索引表へ `Subdomain` 列を足し、検査を GREEN にする。
7. 各 Context の `decisions.md` へ分類の理由を、`glossary.md` へ Aggregate root であることを書く。判断が要った境界だけを `decisions.md` に足す。
8. 境界の見直しの手順を `docs/development/specification-first-workflow.md` へ接続する。

## Tasks

- [x] T001 [Baseline] 全 Context の仮分類、Aggregate の洗い出し、Repository の粒度と ACL の実装位置の現状調査を行う。
- [x] T002 [Acceptance] `mise run check-spec` が `docs/README.md` の分類欠落で落ちることを確認する（Acceptance RED）。
- [x] T003 [App] `tools/check/src/subdomain-classification.ts` の Unit RED を確認し、GREEN にする。
- [x] T004 [Spec] `docs/glossary.md` に Aggregate を定義する。
- [x] T005 [Spec] `docs/design-rules.md` に Aggregate 境界、Repository の粒度と命名、分類が左右するものを書く。
- [x] T006 [Spec] `docs/structure.md` に ACL の配置規約を書く。
- [x] T007 [Spec] `docs/README.md` の索引表へ `Subdomain` 列を足す。
- [x] T008 [Spec] 各 Context の `decisions.md` と `glossary.md` へ分類の理由と Aggregate root を書く。
- [x] T009 [Spec] 境界の見直しの手順を `docs/development/specification-first-workflow.md` へ接続する。
- [x] T010 [Verify] `mise run check-spec` と `mise run verify` を通す。

## Verification

- `mise run check-spec` が通る。
- `docs/database.md` の `tenant_id` 保持区分と `docs/contexts/jobs/decisions.md` の「別テナントの Aggregate の識別子」という記述が、新しい定義と矛盾しない。
- `docs/README.md` の索引表の全行に分類が入っていることを `subdomain-classification` の検査が確かめる。
- `mise run verify`

## Risk Notes

サブドメインの分類は主観が入りやすく、書いた直後は正しく見えても、根拠が書かれていなければ半年後には誰も守らない。分類そのものより「分類が何を左右するか」を先に決めることで、飾りになることを防ぐ。左右するものを決められないなら、分類は書かないほうがよい。

Aggregate の境界を全 Context について一度に書こうとすると、実態の調査が終わらないまま推測で書くことになりやすい。Core に分類した Context から順に書き、Generic の Context は名前の列挙にとどめてよい。

Repository の粒度を現状と異なる規範に決めると、広範な改修を暗黙に約束することになる。現状に合わせる判断も、理由を書けるなら正当である。

## Completion

- **Completed At**: 2026-08-29
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範シナリオ、規範 ID、TypeSpec シンボルはいずれも増減していない。変わったのは、それらを読むための語彙と規則である。`docs/glossary.md` が `Aggregate` と `Subdomain` を定義し、`docs/design-rules.md` が Aggregate の境界の引き方、トランザクションとの対応、Repository は Aggregate root 単位という規則、`Repository` と `Store` の接尾辞が種類を区別しないという事実、そしてサブドメインの区分が左右する 3 つと左右しない 1 つ（検証の強度）を持つ。`docs/structure.md` が Anti-Corruption Layer の配置規約を持ち、`docs/README.md` の索引表が全 21 Context の区分を宣言する。各 Context の `decisions.md` はその区分の理由を、`Core` 5 件の `glossary.md` は Aggregate root の名前を持つ。`SPECIFICATION_FORMAT.md` は索引表が区分を宣言することを *(checked)* の規則として記載する。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`
  - **Requirement**: N/A: 文書と検査の変更であり、製品の観測可能な振る舞いを 1 件も変えないため、対応する規範シナリオが無い。
  - **Observed Failure**: `fail  docs/README.md:80: the context index table declares no Subdomain column; its header must read | Specification context | Subdomain | Go package | Responsibility |` で終了コード 1。列を足した後は通る。さらに `docs/contexts/zz-probe/` を作って索引に載せない状態を作ると `fail  docs/README.md:80: docs/contexts/zz-probe/ is not listed in the context index table` で落ちることを実地で確認した。
  - **Detection Reason**: この検査が区別するのは「分類を書いた作業ツリー」と「分類を書かずに Context を増やした作業ツリー」である。索引表を人が読んで確かめる案では、増えた 1 行が空欄であることは誰も見ないまま通る。実際に落ちる 2 つの経路（列が無い場合と、ディレクトリが索引に無い場合）を両方観測した。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/subdomain-classification.test.ts`
  - **Requirement**: N/A: 同上。検査対象は正本文書の体裁であり、製品の振る舞いではない。
  - **Observed Failure**: 実装を `return []` だけの骨組みにした状態で 11 件中 8 件が失敗した。失敗はいずれも「拒否すべきものを拒否していない」形であり、`rejects a value outside Core, Supporting, and Generic`、`rejects a context directory the index table does not list`、`rejects the same context listed twice` などが該当する。
  - **Detection Reason**: 最も起こりやすい誤った実装は「何でも受け入れる検査」であり、骨組みがそれそのものである。8 件の失敗は、受け入れてはならない入力の種類ごとに 1 件ずつ立っている。
- **Change-Resistance Results**:
  変更した純粋ロジックは `verifySubdomainClassification` の 1 関数である。6 種類の変異を入れて実行した。
  - M1 索引に無い Context ディレクトリの検出を落とす → 検出（1 失敗）
  - M2 値集合の検査を「空でなければよい」に弱める → 検出（3 失敗）
  - M3 重複行を黙って読み飛ばす → 検出（1 失敗）
  - M4 最初の所見で打ち切る → **当初は生存した**。`reports every unclassified row` が件数だけを見ており、読み飛ばした行が「索引に載っていない Context」という別の所見に化けて件数が合っていた。所見の中身まで固定する主張へ書き換えて検出（1 失敗）。
  - M5 行番号を 0 起点で報告する → 検出（1 失敗）
  - M6 索引表をヘッダー全体ではなく接頭辞で選ぶ → **当初は生存した**。列が無い場合の所見が、行ごとの区分の所見に化けても `toContain('Subdomain')` を満たしていた。`declares no Subdomain column` を要求する主張へ書き換えて検出（1 失敗）。
  この方法の限界として、区分そのものの妥当性（ある Context が `Core` であるべきかどうか）はどの変異でも検出できない。それは検査ではなく `decisions.md` に書かれた理由が担う。
- **Verification Results**:
  - `mise run check-spec` - passed
  - `mise run check-links` - passed
  - `mise run test-tools` - passed（263 件）
  - `mise run verify` - passed
