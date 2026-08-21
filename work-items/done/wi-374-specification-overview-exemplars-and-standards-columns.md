---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-15
depends_on: []
change_kind: docs
affected_spec:
  - { path: spec/contexts/authentication/standards.md, requirement: NIST63B4-PASSWORD-MINIMUM }
---

# SPECIFICATION.md の Overview に模範例を与え、Standards の 2 列を定義し、Authorization Boundary を Design へ畳む

## Motivation

`spec/**/SPECIFICATION.md` は AI が書いており、セクションによって品質が揃わない。調査すると、品質はセクションごとに与えられている制約の量に比例していた。Scenarios と State Transitions は `SPECIFICATION_FORMAT.md` に文法と例があり検査もあるので安定している。Glossary と Standards は文法の記述が一行も無いが、既存文書の模倣だけで 22 文書の形が揃っている。一方 Overview・Authorization Boundary・Design は文法も例も検査も無く、文書ごとにばらつく。

つまり品質を支えているのは規則そのものではなく、「単位が定まっていること」と「隣に良い実例があること」である。Glossary が規則ゼロで揃っているのがその証拠になる。

現に次の実害が観測されている。

- Standards の `Adoption` と `Strength` の意味も値集合もどこにも定義されておらず、`spec/contexts/authentication/SPECIFICATION.md:61` に `excluded` かつ `MUST` という解釈不能な行が生じている。Statement が製品側ではなく規格側の視点で書かれているためである。
- Overview に、責務境界とは無関係な文が混ざっている。`spec/contexts/sourcing/SPECIFICATION.md:14` の将来計画、`spec/contexts/oauth2/SPECIFICATION.md:12` と `spec/contexts/system/SPECIFICATION.md` の読み順の案内がそれにあたる。これらは現在状態の仕様が持つべき内容ではなく、模倣の手本としても悪い。
- Authorization Boundary が各 Context で操作ごとのスコープ割り当てを散文で再掲している。正本は TypeSpec の operation に付く `x-api-token-scopes` であり、`tools/check/src/admin-scopes.ts` が全 admin operation に宣言を強制したうえで語彙との対応を双方向に検査している。散文側は検査されないため、片方だけが古くなる。しかもこれを禁じる規則は既に存在しており、`spec/contexts/api-tokens/SPECIFICATION.md` の Authorization Boundary 本文に埋まっている——他の Context の文書を書くときに誰も読まない場所にある。
- そもそも正準セクションのうち 2 つが同じ責任を主張している。`SPECIFICATION_FORMAT.md:56-57` は Design が `security boundaries` を記録すると定めており、Authorization Boundary との境目がない。実際 `claim-mapping` はこの節を省略し、`jobs` の節は「HTTP エンドポイントを持たず、同一プロセス内の Go 呼び出しだけなので、テナントを越える経路がない」という、独立した最上位セクションを持つ内容ではないものになっている。

## Scope

- Overview の模範となる文書を 2 本仕上げ、`.claude/skills/spec-change/SKILL.md` からそれを指す。規則は増やさない。
- 模範を汚している 3 文（将来計画 1 文、読み順の案内 2 文）を削除する。
- `SPECIFICATION_FORMAT.md` に、Overview が述べる内容を数行、Standards の節（表の形と 2 列の意味）、Authorization Boundary の節を追加する。
- `spec/contexts/authentication/SPECIFICATION.md:61` の `excluded` × `MUST` 行を製品側の宣言へ書き換える。
- 正準セクション `Authorization Boundary` を廃止し、内容を Design 配下の `Authorization boundary` H3 へ移す。薄いものは隣接する機構の説明へ畳む。移動と同時に、自明なスコープ割り当ての再掲を取り除く。スコープ語彙と、操作名から `read` / `write` を判断できない場合の結論と理由は残す。
- Authorization Boundary の規則を `spec/contexts/api-tokens/SPECIFICATION.md` の本文から `SPECIFICATION_FORMAT.md` へ引き上げる。
- `tools/check/src/specification-doc.ts` の `SECTION_ORDER` から `Authorization Boundary` を外し、Standards の検査を追加する。`specification-doc.test.ts` にケースを足す。

## Out of Scope

- **Design セクション**。全 22 文書で Design 配下の H3 は 123 個あり、共通の規約が無い。名前が揃っているのは `Design Decisions`（15 文書）と `Internal Interfaces`（14 文書）だけで、どちらも `SPECIFICATION_FORMAT.md` に記載が無い。機械的に判定できるのは見出し言語の統一程度で、内容の質——決定を述べているのか単なる叙述か、根拠が「なぜ他ではないか」を含むか——は判断でしか見られない。3 型（`Design Decisions` / `Internal Interfaces` / `Persistence`）を抜き出して自由記入欄を縮める案も検討したが、実効的な検査が付くのは 2 型のみで残りは結局レビュー判断になるため、中途半端を避けて見送る。良い方法が見つかった時点で改めて扱う。なお `Internal Interfaces` には `- Input invariant: <述語(引数)>` / `- Result invariant: …` という半形式の記法が 8 文書 35 行で自然発生している（`data-keys`、`jobs`、`identity-management` ほか）。文書化も検査もされていないが、将来 Design を扱うときの出発点になる。
- **認可境界の内容そのものの削除**。節は廃止するが、中身は捨てない。`signing-keys` の「管理 API のレスポンスに秘密鍵素材を含めることは、いずれの権限でもできない」（アクセス制御ではなくデータ流出境界）、`ws-federation` の「いずれの経路でも、未登録の宛先にトークンを発行することはない」（4 種の認証機構を横断する fail-closed 不変条件）、`provisioning` の「配信そのものは管理者の権限を借りない」（権限伝播）、`sourcing` の取り込み中に解決する参照のテナント封じ込めは、いずれも TypeSpec にも `backend/shared/spec/policy.go` の規則表にも書けない。ロール要件（`admin` / `system_admin`、有効かつ認証済み、所属テナント）も TypeSpec にはなくコードにしかないため、散文が唯一の仕様である。捨てるのは操作ごとのスコープ割り当てだけである。
- **Glossary の追認**。既に安定しており、文書化しなくても壊れていない。
- **正規シナリオと状態遷移表の変更**。本件は散文と表の整形だけで、規範的な意味を変えない。
- **YAML 化**。目的が品質ではなく執筆と検証の人間工学であり、混ぜると本件が中途半端になる。別 work item で評価する。

## Design

- Overview には規則を書かず、模範文書を指す。Glossary と Standards が規則ゼロで 22 文書揃っている実績から、この種の散文には規則より実例のほうが効くと判断する。ただし模倣が効くのは隣に良い実例があるときだけなので、模範を先に仕上げてから契約を書く。
- 模範は `spec/contexts/data-keys/SPECIFICATION.md`（所有するものと、所有しないもの＋委譲先を 2 段落で述べる形）と `spec/contexts/sourcing/SPECIFICATION.md`（帰属の判定基準を通信方向ではなく外部権威の有無で定義し、除外先を名指しする形）とする。
- 模範の置き場所を 2 つに分ける。`SPECIFICATION_FORMAT.md` には汎用の作例（リポジトリ名を使わない良い形と悪い形）を置き、実在の模範文書 2 本への参照は `.claude/skills/spec-change/SKILL.md` に置く。`SPECIFICATION_FORMAT.md` は将来リポジトリ外へ切り出す予定でリポジトリ固有のパスを書けず、`CLAUDE.md` は毎回読まれる常時コンテキストなので、仕様を書くときにだけ要る情報の置き場所としては不適切だからである。
- Overview には機械検査を付けない。「将来」「本書は…読む」といった語句を拒否する検査を試作したところ、22 文書で誤検出 0・実違反 2 件を検出できたが、この方式は対象語が原理的に閉じない。同じ意味を表す言い回しは無数にあり、追い続ける保守費が検出できる違反の価値を上回る。閉じた値集合を持つ Standards とはこの点が決定的に違う。Overview は作例と模範文書だけで支える。ただし試作が見つけた 2 件（`identity-governance` の将来計画、`identity-management` の読み順）は実際の違反なので直す。
- Standards の 2 列は別の軸として定義する。`Adoption` はその規格の機能・規則を採るか (`required` / `optional` / `partial` / `excluded`)、`Strength` は採った場合に本製品が課す強さ (`MUST` / `MUST NOT` / `SHOULD` / `MAY`) とする。この定義は実データ 128 行と整合する。`optional` × `MUST` が 9 行あるが矛盾しない——「機能を提供するかは任意、提供するなら MUST」を表しており、2 列が独立した軸であることの実例になる。矛盾するのは `excluded` × `MUST` の 1 行だけである。
- `spec/SPECIFICATION.md` の `### Reading order` は残す。ルート文書は全体の入口であり、Context 文書の Overview とは役割が違う。frontmatter が `context: repository` であることで検査側から区別できる。
- Authorization Boundary の規則は新設ではなく移設である。`spec/contexts/api-tokens/SPECIFICATION.md` が既に「正本は `x-api-token-scopes`」「網羅する表はここにも各 Context にも置かない」「各 Context には、その Context が使うスコープ語彙と、操作名から `read` / `write` の割り当てを判断できない場合の結論と理由だけを記載する」と定めている。守られていないのは規則が悪いからではなく、置き場所が悪いからなので、`SPECIFICATION_FORMAT.md` へ引き上げる。api-tokens 側には、`ApiTokenScope` 語彙そのものを所有する Context としての記述を残す。
- 各 Context の是正は文の削除ではなく圧縮とする。`saml:read` / `saml:write` のような語彙は残し、「参照だけを許可する」「変更を許可する」という操作名から自明な対応だけを落とす。自明でない結論——`provisioning` の「接続テスト、On-Demand Provision、Full Resync、Quarantine の解除、配信の再試行はいずれも下流を変えるため変更系に属する」、`application` のテナントデフォルトサインインポリシーを `settings:*` に対応させる判断、`audit` の「対になる変更系のスコープは存在しない」、`identity-governance` のドライラン、`workloadidentity` の JWKS 再取得、`authentication` の MFA 免除と認証イベントバケット、`sourcing` の Discovery——はいずれも残す。これらが api-tokens の言う「判断できない場合の結論と理由」にあたる。
- `identity-management`、`oauth2`、`tenancy` は既に `users:*` / `oauth-clients:*` / `settings:*` という圧縮形を使っており、そのまま模範になる。圧縮はせず、移動だけ行う。
- 移動先の H3 名は全文書で `Authorization boundary` に統一し、`## Design` 直下の最初の H3 に置く。セキュリティレビューが Context 横断で認可境界を読むとき、節が無くなっても grep で辿れるようにするためである。内容が薄い文書を隣接する機構へ畳む案も検討したが採らない。`jobs` の「HTTP エンドポイントを持たない」のような文はセキュリティ上の主張であり、機構の説明に混ぜると埋もれる。見出しの統一による横断可読性のほうが、見出し 1 つ分の節約より価値がある。
- 節を残して `SPECIFICATION_FORMAT.md` §3 の Design から `security boundaries` を外す案も採れるが、採らない。実データを見ると各 Context の内容は「ロール要件」「データ流出境界」「権限伝播」「テナント封じ込め」「fail-closed の既定」が混在しており、いずれも Design が扱う現在設計とその根拠である。責任の重複を消すなら、少ないほうの節を畳むのが素直である。
- 検査を書いたあと全 22 文書へ走らせ、落ちた箇所を検査の妥当性の評価に使う。既存の良い文書を落とす検査は、検査のほうが間違っているとみなして削るか緩める。

## Plan

1. 模範を汚している 3 文を削除し、Standards の矛盾行を直す。実例が先、契約が後。
2. `SPECIFICATION_FORMAT.md` と `CLAUDE.md` を書く。
3. 検査とテストを追加し、全 22 文書へ走らせて妥当性を確認する。
4. `just spec-diff` で差分に規範的変化が無いことを確かめ、`just verify-spec` を通す。

## Tasks

- [x] T001 [Docs] `spec/contexts/sourcing/SPECIFICATION.md:14` の将来計画 1 文、`spec/contexts/oauth2/SPECIFICATION.md:12` と `spec/contexts/system/SPECIFICATION.md` の読み順の案内を削除する。
- [x] T002 [Spec] `spec/contexts/authentication/SPECIFICATION.md:61` の `NIST63B4-PASSWORD-MINIMUM` を製品側の宣言へ書き換え、`Strength` を整合させる。
- [x] T003 [Spec] 節を持つ全 20 文書の `## Authorization Boundary` を `## Design` 配下の `### Authorization boundary` へ移す。移動と同時に自明なスコープ割り当ての記述を圧縮し、語彙と、判断できない場合の結論と理由は残す。
- [x] T004 [Docs] `SPECIFICATION_FORMAT.md` に Overview の内容規定と Standards の節を追加し、§3 の正準セクションから `Authorization Boundary` を外す。§6 を、認可境界は Design が持ち、操作ごとのスコープ割り当ては契約側が正本である旨へ改訂する。`spec/contexts/api-tokens/SPECIFICATION.md` に埋まっていた規則はここへ移設する。
- [x] T005 [Docs] `.claude/skills/spec-change/SKILL.md` に Overview の模範文書 2 本への参照を追加する。
- [x] T006 [Tooling] `tools/check/src/specification-doc.ts` の `SECTION_ORDER` から `Authorization Boundary` を外し、Standards の検査を実装する。`specification-doc.test.ts` にケースを追加する。
- [x] T007 [Verify] `just check-spec` を全文書へ、`just test-tools` と `just typecheck-tools`、`just check-admin-scopes`、`just spec-diff`、`just verify-spec` を通す。
- [x] T008 [Completion] 完了記録を追加して `work-items/done/` へ移動する。

## Verification

- `just check-spec`
- `just test-tools`
- `just typecheck-tools`
- `just spec-diff`
- `just verify-spec`

## Risk Notes

Standards の値集合を閉じると、新しい標準を追加するときに既定値へ収まらない事例が出る可能性がある。実データ 128 行が既に 4 値と 4 値に収まっていることを根拠に閉じるが、収まらない事例が出た時点で値集合そのものを見直す。

## Completion

- **Completed At**: 2026-08-15
- **Summary**:
  正準セクションを 7 個から 6 個へ減らした。`Authorization Boundary` を廃止し、節を持っていた 20 文書の内容を `Design` 配下の `### Authorization boundary` へ移した。`SPECIFICATION_FORMAT.md` §3 が既に Design の担当として `security boundaries` を挙げており、2 つの正準セクションが同じ責任を主張している状態だったのを解消した。H3 名は全文書で統一し、`## Design` 直下の最初の H3 に置いたので、Context 横断で認可境界を読む経路は残っている。`SECTION_ORDER` から外したため、以後 `## Authorization Boundary` を書いた文書は `just check-spec` が落とす。
  移動と同時に、10 文書から操作ごとのスコープ割り当ての再掲を取り除いた。`x-api-token-scopes` が正本で `just check-admin-scopes` が 210 operation の宣言と `ApiTokenScope` 語彙の対応を双方向に検査しており、散文側は検査されない片割れだった。スコープ語彙と、`provisioning` の変更系判定や `audit` の「対になる変更系スコープは存在しない」のような自明でない結論は残した。この規則自体は `spec/contexts/api-tokens/SPECIFICATION.md` の本文に埋まっていたので、`SPECIFICATION_FORMAT.md` §7 へ引き上げた。
  `SPECIFICATION_FORMAT.md` に Standards の節 (§5) を新設し、これまでどこにも定義が無かった `Adoption` と `Strength` を独立した 2 軸として定義した。`Adoption` は規格の機能を採るか、`Strength` は採った場合に課す強さである。`optional` × `MUST` が矛盾しないこと、`excluded` が義務を持てないことを明記し、値集合・ID の一意性・正準ヘッダー行・`excluded` の義務禁止を検査に落とした。既存で唯一矛盾していた `NIST63B4-PASSWORD-MINIMUM`（`excluded` × `MUST` で Statement が規格側の視点）を、製品側の宣言へ書き換えた。
  Overview は規則ではなく作例で支えることにした。`SPECIFICATION_FORMAT.md` に良い形と悪い形の作例を置き、実在の模範文書 2 本 (`data-keys`、`sourcing`) への参照は `.claude/skills/spec-change/SKILL.md` に置いた。模範を汚していた 5 文（`sourcing`・`identity-governance` の将来計画、`oauth2`・`system`・`identity-management` の読み順の案内）を削除した。
  Overview の語句を機械検査する案は試作して破棄した。22 文書で誤検出 0・実違反 2 件を検出できたが、対象語が原理的に閉じず、保守費が検出価値を上回る。閉じた値集合を持つ Standards との違いをこの work item に記録した。Design セクションは方法が無いため対象外とし、その判断根拠（H3 123 個に規約なし、機械判定できるのは見出し言語程度）を Out of Scope に残した。
- **Verification Results**:
  - `just check-spec` - passed（22 文書、レンダラー dry-run 25 document(s) 含む）
  - `just check-admin-scopes` - passed（210 operation(s)）
  - `just test-tools` - passed（117 tests、Standards 検査 6 ケースと廃止セクション 1 ケースを追加）
  - `just typecheck-tools` - passed
  - `just verify-spec` - passed
  - `just spec-diff` - `no normative specification change against main`（散文の移動と整形だけで、規範的な意味は変えていない）

Authorization Boundary の廃止は 20 文書の構造変更であり、`just spec-diff` の差分が大きくなる。認可の**意味**を変えないことが受け入れ条件なので、移動した文はそのまま、削除するのは操作ごとのスコープ割り当ての記述に限る。部分適用は、節を持つ文書と持たない文書が混在してどちらが正か分からなくなる最悪の状態になるため、節を持つ 20 文書すべてを一度に移す。`SECTION_ORDER` から外すことで、以後 `## Authorization Boundary` を書いた文書は `just check-spec` が落とす。
