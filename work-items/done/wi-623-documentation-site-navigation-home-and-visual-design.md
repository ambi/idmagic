---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-19
priority: p2
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "開発者向けの生成ドキュメントサイトと文書中の語彙だけを変更し、利用者向けの機能、互換性、移行手順は変わらない。"
  references: []
initial_context:
  specification:
    - DOCUMENTATION_GUIDE.md
    - WORK_ITEM_FORMAT.md
    - docs/README.md
  typespec: []
  source:
    - tools/render-docs/src/main.ts
    - tools/render-docs/src/render.ts
    - tools/workspace/src/document-layout.ts
  tests:
    - tools/render-docs/src/render.test.ts
    - tools/render-docs/src/documentation-quality.test.ts
  stop_before_reading:
    - backend
    - spec/contexts
    - docs/domain
spec_impact:
  kind: none
  reason: "生成ドキュメントサイトの区分、入口、体裁と、文書で使う語彙だけを変え、プロダクトの振る舞いと公開 API 契約は変えない。"
---

# 生成ドキュメントサイトの区分、トップページ、体裁を読み手の目的に合わせる

## 動機

サイドバーの「システム全体」は、現在ページがその枝にあるかどうかにかかわらず子文書を常に描画する。
`tools/render-docs/src/render.ts:611` の `wholeSystemContext` だけが、ほかの枝が使う `containsCurrent` の判定を通らずに `<ul>` を出力するためである。
結果として、システム設計を開いただけで無関係な 5 文書が展開される。

同じサイドバーで、プロダクト概要へたどり着くには「設計文書 > コンテキスト文書 > システム全体 > プロダクト概要」の 4 段を降りる。
Bounded Context の文書も「設計文書」の内側にあり、設計の一部という誤った包含関係を示す。
「システム設計」は 7 領域を束ねるだけの中間段で、領域の索引と同じ情報しか示さない。

運用の区分がない。
`docs/operations/` は「設計文書」の末尾に並び、`docs/runbooks/` の 10 文書はサイトにまったく載っていない。
警報の `runbook_url` が指す手順を、生成サイトからは開けない。

トップページの「主要文書」「開発とリファレンス」「Bounded Context」は、文書の置き場所をそのまま見出しにしたものである。
初めて開いた読者が何を知りたいかに対応せず、21 個の Bounded Context のカードが縦に連なる。
全リンクはサイドバーにあるため、トップページが再度リンクを並べる必要は薄い。

体裁は、カードの影と枠でページ全体を囲む構成で、本文よりも枠が目立つ。

「正本」「正準文書」は一般の日本語ではなく、初読で意味を取れない。

## 対象範囲

- 「システム全体」の枝を、ほかの枝と同じ現在位置基準の展開規則へそろえる。
- サイドバーの最上位区分を、設計文書、コンテキスト、開発文書、運用文書、リファレンス、フォーマットの順に並べる。
- Bounded Context の文書を「設計文書」の外へ出し、独立した最上位区分にする。
- `docs/` 直下の文書（プロダクト概要、用語集、全体の標準仕様、構造、シナリオ）を「設計文書」の直下へ移し、「システム全体」という枝を廃する。
- 7 領域の設計索引を「設計文書」の直下へ並べ、「システム設計」を枝から葉のリンクへ変える。
- `docs/operations/` を「運用文書」へ移し、`docs/runbooks/` を生成対象へ加えてその配下に並べる。
- トップページを、読者の目的から入る構成へ書き換える。
- `assets/site.css` を、地の文中心のリファレンスサイトの体裁へ作り直し、本文右に見出し目次を加える。
- 名札、パンくず、`<title>` から表題の Markdown 記法を外す。
- API リファレンスの OpenAPI を組み立て時に展開し、`$ref` の解決を不要にする。
- 生成サイトを HTTP で配る `serve-docs` を加える。
- 仕様に限らない文書を扱う名前を `spec` から `docs` へそろえる。
- `docs/` の構成を文書の区分と一致させ、サイドバーをディレクトリの写しへ戻す。
  `docs/domain/` を `docs/domain/` へ改名し、用語集、全体の標準仕様、構造、システム横断シナリオを
  その直下へ移し、プロダクト概要を `docs/design/` へ移す。
- `docs/domain/glossary.md` から設計文書の語を取り出し、`docs/design/README.md` へ収める。
- `docs/design/README.md` を、設計文書の入口としてゼロから書き直す。
- 移動した文書を指す参照を全件付け替える。
- 文書体系の意味で使う「正本」「正準文書」「正準 Markdown」を「一次情報」「一次情報文書」へ置き換える。
- 利用者向け UI 文言の「正準 scope 値」を平易な表現へ置き換える。
- `DOCUMENTATION_GUIDE.md` が定める version の表記に反する「版」を、`docs/domain/` と `backend/` を除く範囲で「バージョン」へそろえる。
- 語彙の規則を `documentation-quality.test.ts` のゲートにし、宣言だけの状態をやめる。
- 上記に追従して `tools/render-docs/src/render.test.ts` を更新する。

## 対象外

- Bounded Context の分類と名称。`docs/domain/<context>/` の配下のファイル構成も変えない。
- API リファレンス（Swagger UI）、モデルカタログ、トレーサビリティの各ページの中身。
- `docs/releases/` の文書。
- 形式の一意性を指す「正準符号化」「正準ドット名」など、文書体系と無関係な語の置き換え。
- データの発生元を指す「正本」（PostgreSQL、Vault のスナップショット、人事システムなど）。文書体系の語ではない。
- `docs/domain/**/scenarios.feature.md` と `backend/**` の「版」。規範シナリオの本文と production のコメントに及ぶため、仕様先行の手順を通す別の作業項目が扱う。
- `revision` の訳語である「第 N 版」。version の表記ではない。
- 全文検索の追加。

## 設計

### サイドバーの区分と順序

最上位区分は次の順に並べる。
フォーマットは参照頻度が最も低いため末尾へ置く。

区分はディレクトリの写しとする。
サイドバーが `docs/` の並びと違う形を作ると、どちらが正しいのか読み手には決められない。

| 区分 | 索引 | 直下に並べるもの |
| --- | --- | --- |
| 設計文書 | `docs/design/README.md` | プロダクト概要、要求、アーキテクチャ、7 領域の設計、検証設計 |
| ドメイン設計文書 | `docs/domain/README.md` | 用語集、全体の標準仕様、構造、システム横断シナリオ、Bounded Context 21 件 |
| 開発文書 | `docs/development/README.md` | 開発文書の各ページ |
| 運用文書 | `docs/operations/README.md` | サービス管理、保守、運用手順の枝 |
| リファレンス | 生成する | API リファレンス、モデルカタログ、トレーサビリティ |
| フォーマット | 生成する | 文書ガイド、仕様フォーマット、作業項目フォーマット |

区分はすべて索引ページを持つ。
索引の無い区分は、名前だけがクリックできない例外になる。
`docs/` に対応する文書がない「リファレンス」と「フォーマット」は、下位のページを並べた索引を生成する。

区分の名前は、収める文書の種類をそのまま言う。
「コンテキスト」だけでは何の文脈を指すか決まらず、「コンテキスト文書」でも何の文書か伝わらない。
21 件は Bounded Context ごとのドメインの設計なので、枝の名札は「ドメイン設計」とする。
「ランブック」は外来語のまま意味を伝えないので、`docs/README.md` が使う「運用手順」を採る。

ドメイン設計文書は「設計文書」の子にせず、設計文書と開発文書の間の区分にする。
`docs/domain/` は `docs/design/` と別のディレクトリなので、片方をもう片方の内側へ描くと、
サイドバーがディレクトリの構成と食い違う。

区分をディレクトリの写しにできるよう、`docs/` の構成そのものを役割に合わせる。

| 文書 | 移動後 | 理由 |
| --- | --- | --- |
| `docs/contexts/` | `docs/domain/` | 収めているのは Bounded Context の一覧だが、まず伝えるべきはそれがドメイン設計であること |
| `docs/product-overview.md` | `docs/design/product-overview.md` | 設計文書を読み始める前に読む |
| `docs/glossary.md` | `docs/domain/glossary.md` | 自身が「Context を跨いで意味が固定される語」と宣言する。Context ごとの `glossary.md` に対するシステム全体の側 |
| `docs/standards.md` | `docs/domain/standards.md` | Context ごとの `standards.md` に対するシステム全体の側 |
| `docs/scenarios.feature.md` | `docs/domain/scenarios.feature.md` | Context ごとの `scenarios.feature.md` に対するシステム全体の側 |
| `docs/structure.md` | `docs/domain/structure.md` | 節の過半が Context の内部構造と Context 間イベントである |

`docs/glossary.md` の「設計文書の用語」の節は、ドメインではなく設計文書の語を並べている。
節ごと `docs/design/README.md` へ移す。

`docs/design/README.md` は題名が「システム設計」で、内容は 7 領域の索引表だけだった。
設計文書の入口として書き直し、層の対応、7 領域、設計文書の用語を持たせる。
`docs/domain/README.md` は無かったので新設し、区分の入口にする。

移動したパスを指す参照は、work item 265 件、`backend/`、`tools/`、`.agents/skills/` を含めて全件付け替える。
`spec-diff` は履歴のリビジョンを読むので、当時のパスを現在の配置へ写してから同定する。
そうしないと、移動しただけの文書が「規範の追加」として現れ、完了済み記録の `documentation_impact` まで不整合になる。

「システム設計」の枝は廃し、7 領域を区分の直下へ上げる。
この索引は 7 領域と同じことしか示さず、名前も抽象的で、階層を一つ増やすだけである。
書き直した `docs/design/README.md` が区分の索引になるので、ページ自体はそこから読む。

### 展開規則

枝が子を描画するのは、現在ページがその枝の内側にある場合だけとする。
`systemTree` の `directory` が使う `containsCurrent` を、`docs/` 直下の文書を並べる枝と Bounded Context の枝にも同じ形で適用する。
現在の実装では、`wholeSystemContext` が無条件に、`contextBranch` が `openContext` 比較で判定しており、規則が三通りに分かれている。
判定を `containsCurrent` 一つに寄せ、規則の分岐をなくす。

ただし、子を出すかどうかと開くかどうかは別の問いである。
子を HTML から落とすと、その枝の内側にいないページからは子へ到達できなくなる。
到達性は `validateSiteLinks` が全ページについて検査するので、これは生成の失敗として現れる。
到達させるために枝を開いたままにすると、無関係なページを開いただけで一覧が展開される。

両立させるため、枝は `details` として組む。
子は常に HTML へ載せ、`open` を付けるかどうかだけを現在位置で決める。
読み手は自分で開くこともでき、開閉記号がそのまま「子を持つ」という手がかりになるので、
リンクの色や太さで子の有無を示す必要もなくなる。
名札だけの枝への例外は要らない。

### 運用文書の出力先

`docs/operations/**` の出力先を `specification/operations/**` から `operations/**` へ変え、`docs/development/**` と同じ形にする。
`docs/runbooks/**` は `operations/runbooks/<名前>.html` へ出力し、サイドバーでは「ランブック」の枝の下に並べる。
ランブックはファイル名の自由な集合であるため、`docs/development` と同じ `procedureDocuments` の走査で読み込む。
`RenderedDocument` には `operations`、`operations-child`、`runbook` の区分を加える。

採用しない案として、ランブックを最上位区分にする構成がある。
ランブックは運用手順の一種であり、`docs/operations/README.md` が入口を定めるため、運用文書の内側に置く。

### トップページ

トップページは `docs/README.md` そのものとする。
区分の説明はレンダラー側に書かず、`docs/README.md` を `index.html` へ出力する。
同じ案内を二か所に書くと、片方だけが古くなる。

`docs/README.md` は、サイト名、プロダクトを一文で述べる紹介、区分の表、読み順、執筆上の境界を持つ。
紹介の一文はリポジトリの `README.md` と同じものを使い、言い換えない。

採用しない案として、「製品を知る」「設計を読む」のような目的別のカードを置く案がある。
目的の区切りはサイドバーの区分と一致しないので、読者は同じ場所へ二通りの名前で案内される。
Bounded Context の一覧もトップページへは置かない。サイドバーが常に持つ。

### 表題の扱い

表題は本文の見出しとしては Markdown のまま組むが、名札、パンくず、`<title>` では地の文になる。
`# \`/token\` のエラー率` のような表題をそのまま流すと、記法が読み手へ出る。
囲みの種類を数え上げると強調や参照を取りこぼすので、インラインとして組んでからタグを外す。

### API リファレンスの参照解決

Swagger UI 5.x は `#/components/schemas/...` のような内部参照であっても、`baseDoc` の URI を
取得して解決する。
メモリ上の文書から読むのは `baseDoc` が falsy のときだけで、`url` を与えない場合でも
`document.baseURI` から埋めるため falsy にはならない。
`file://` で開いたページはどの URI も取得できないので、この読み方では解決が成立しない。

| 文書の渡し方 | `file://` で開く |
| --- | --- |
| 公開した JSON を `url` で指す | 読み込み自体が失敗し、何も表示されない |
| 埋め込んだ文書から作る `blob:` URL | 表示されるが、操作を開くと解決の警告が出る |
| 文書そのものを `spec` で渡す | 同じく解決の警告が出る |

いずれも渡し方の違いであって、解決が要る限り結果は変わらない。
そこで**組み立て時にすべての `$ref` を展開**し、解決そのものを不要にする。

| 判定内容 | 結果 |
| --- | --- |
| 展開後に残る `$ref` | 0 件 |
| 展開した節が保つモデル名 | `$$ref` に元の参照を残すので Swagger UI は名前を表示できる |
| 自分自身へ戻る参照 | 1 箇所。SCIM の属性が入れ子の属性を持つ形であり、型だけの節へ置き換えて打ち切る |
| `api/index.html` の大きさ | 1.4 MiB から 6.4 MiB |

依存の更新は原因ではない。
`22c61d46^` の swagger-ui-dist は 5.32.14、現在は 5.32.15 だが、`baseDoc`、`absolutifyPointer`、
`extractFromDoc` の出現箇所は両版で一致しており、参照解決の実装は変わっていない。

`mise run serve-docs` は生成サイトを HTTP で配る経路として残す。
参照を展開したので API リファレンスには不要だが、配信した状態で確かめたいときに使う。
配信のために依存を増やさず `Bun.serve` だけで書き、`site/` の外へ出る経路は 404 にする。

### ディレクトリへの参照

`[設計](design/)` のようなディレクトリへの参照は、リポジトリでは読めるが生成サイトには
対応するページがない。
リンクの解決先が見つからないとき、その段の `README.md` を次に探す。
文書側の書き方を変えずに、どちらの読み方でも行き先が残る。

### 名前

生成サイトは要求と仕様だけでなく、設計、開発、運用の文書も載せる。
`spec` を名前に持つと、扱う対象を実際より狭く宣言することになる。

| 対象 | 変更後 | 理由 |
| --- | --- | --- |
| `tools/render-docs/` | `tools/render-docs/` | 生成サイト全体を作る |
| `render-spec-docs`、`check-rendered-spec` | `render-docs`、`check-rendered-docs` | 同上 |
| サイトの出力接頭辞 `specification/` | `docs/` | `docs/` 直下の文書を載せる場所である |
| `renderSpecificationSite` | `renderDocumentationSite` | 同上 |

`compile-spec`、`check-spec`、`verify-spec`、`spec-diff` は `spec/` の TypeSpec と規範文書を扱うので、
名前が対象と一致している。変えない。
`spec-render` は TypeSpec から派生する成果物の再生成をまとめる入口であり、そのまま残す。

### 体裁

参照する体裁は、地の文を主役に置き、枠と影で囲まない構成である。

| 対象 | 変更後 |
| --- | --- |
| ページの枠 | `.document`、`.reference-page`、`.model-detail` のパネル枠と影を外し、本文を背景色の上に直接置く |
| 本文幅 | 72 文字相当（約 760px）で固定し、余った幅は見出し目次へ配分する |
| 見出し目次 | 本文右に固定し、`h2` と `h3` を並べる。現在位置をアクセント色で示す。1200px 未満では隠す |
| 字送り | 本文 16px、行間 1.75、見出しは字間をわずかに詰める |
| 色 | 中間色の灰を基調に、アクセント 1 色。明暗どちらの配色でも同じ階調差を保つ |
| サイドバー | 階層の縦罫を外し、1 段ごとに 18px の字下げで階層を示す。枝は開閉記号を持ち、葉はその幅だけ頭をそろえる。リンクと名札の色と太さは子の有無で変えず、現在ページだけを左端のアクセントバーと色で示す |
| 表 | 縦罫を外し、横罫と見出し行の背景だけで区切る |
| コード | 枠線と淡い背景を与え、本文との境界を明示する |

見出し目次は組み上げた本文の HTML にある見出しから作る。
綴りから `id` を作り直すと、囲み記号の中の行や `contextReference` が生成した節を取りこぼす。
現在位置は `site.js` の交差監視で更新する。

本文幅 760px と目次 216px を並べるには 1200px 程度の視野幅が要るため、折り返しは 900px ではなく 1200px に置く。
サイドバーが隠れる 900px とは別の境界である。

## 計画

1. サイドバーの区分、順序、展開規則、運用文書の掲載範囲、トップページの節構成を期待値としてテストへ書き、失敗を確認する。
2. `main.ts` の読み込み対象へ `docs/runbooks` を加え、`render.ts` の区分と出力先を変える。
3. `navigation` を書き換え、展開判定を `containsCurrent` へ統一する。
4. トップページを書き換える。
5. `site.css` を作り直し、見出し目次を追加する。
6. 「正本」「正準文書」を置き換える。
7. サイトを再生成し、全体を検証する。

語彙の置き換えは、置換対象が改行をまたがないため `sd -s` で一括し、`git diff --stat` で着弾件数を確かめる。
置換後に文意が崩れる箇所は個別に推敲する。

## タスク

各タスクの赤・緑の確認には `mise run test-tools-file -- render-spec-docs/src/render.test.ts` を使い、
ツール全体の型と静的検査は `mise run typecheck-tools` と `mise run lint-tools` を、
生成物との整合は `mise run check-rendered-spec` を使う。

- [x] T001 [Spec] `DOCUMENTATION_GUIDE.md` と `docs/README.md` の記述を、新しい区分と語彙へ合わせる。
- [x] T002 [Acceptance] サイドバーの区分順序、非現在枝の非展開、ランブックの掲載、トップページの節構成を `render.test.ts` で RED にする。
- [x] T003 [App] `main.ts` と `render.ts` の掲載対象、区分、出力先を変える。
- [x] T004 [App] `navigation` の展開判定を `containsCurrent` へ統一し、区分の入れ子を平らにする。
- [x] T005 [App] トップページを目的別の構成へ書き換える。
- [x] T006 [App] `site.css` を作り直し、見出し目次を `render.ts` と `site.js` へ加える。
- [x] T007 [App] 「正本」「正準文書」「正準 Markdown」「正準 scope 値」を置き換え、version の表記を「バージョン」へそろえ、どちらも `documentation-quality.test.ts` のゲートにする。
- [x] T008 [Verify] `mise run spec-render` で再生成し、変更を検証する。

## 検証

- `mise run test-tools-file -- render-spec-docs/src/render.test.ts`
- `mise run spec-render`
- `mise run verify`

## リスク

出力先の変更により `specification/operations/*` の URL が失われる。
サイトは追跡しない生成物で、外部から参照される固定 URL ではないため、転送は用意しない。

CSS の全面的な作り直しは、Swagger UI、Mermaid 図、シナリオのキーワード表示など、既存の要素別指定に影響する。
再生成後に API リファレンス、アーキテクチャ図を含むページ、シナリオページを実際に開いて確認する。

語彙の一括置換は、「正準符号化」のように意味の異なる語まで巻き込む危険がある。
置換対象を「正本」「正準文書」「正準 Markdown」「正準 scope 値」の語形に限り、`docs/`、ルートの方法論文書、`tools/`、`frontend/` の該当箇所だけを変える。

## 完了

- **Completed At**: 2026-09-19
- **Summary**:
  `mise run spec-diff` reports no normative specification change against `main`: the normative ids, their text
  and their coverage are untouched, and the tools that read history now map a document's former path onto its
  current one so a move is not counted as an addition.
  `docs/` is arranged by what each document is, and the sidebar is a plain mirror of it. `docs/contexts/` became
  `docs/domain/`, because what a reader needs first is that these are domain design documents, not that they
  happen to be a list of Bounded Contexts. The glossary, the adopted standards, the cross-context scenarios and
  the repository structure moved under it, since each is the system-wide side of a document every Context holds
  under the same name — or, for `structure.md`, is mostly Context internals and inter-context events. The
  product overview moved into `docs/design/`, because it is what a reader reads before any design document.
  `docs/design/README.md` was an abstract title over an index of the seven areas; it is now the design
  division's entrance, and the seven areas sit directly under it. `docs/domain/README.md` is new. Every
  division now has an entrance, including the two generated ones, so no division is a name that cannot be
  opened.
  Sidebar branches are `details` elements: their children are always in the HTML, so every page is reachable,
  and only the ancestors of the current page are open, so no unrelated list expands. Indentation is 18px per
  level, a leaf aligns with its sibling branch's label, and a link looks the same whether or not it has
  children — the disclosure marker says that instead.
  The API reference no longer asks Swagger UI to resolve anything: the document is dereferenced when the page is
  built, so no `$ref` is left. Reading the bundled Swagger UI 5.32.15 showed why no way of handing it the
  document helps — it always sets `baseDoc` from the document URL and resolves even an internal `#/…` pointer by
  retrieving that URI, which a page opened as a file cannot do. The published URL then loads nothing, and the
  `blob:` URL and `spec` both load but warn when an operation is expanded. Dereferencing removes the retrieval
  instead of relocating it. Each expanded node keeps its `$$ref`, so Swagger UI still shows the model name; the
  one self-referential schema, a SCIM attribute holding nested attributes, is cut off with a typed placeholder;
  and `api/index.html` grows from 1.4 MiB to 6.4 MiB. The dependency bump from 5.32.14 to 5.32.15 was ruled out
  — the resolver code is identical in both.
  `docs/README.md` is the site's top page rather than a second copy of the same guidance; a title reaches a nav
  label, a breadcrumb and `<title>` as plain text; the stylesheet puts prose on the page background with a
  760px measure and a sticky heading outline; `docs/operations/README.md` names the runbooks it owns; and a
  link to a directory resolves to that directory's `README.md`.
  The names say what they hold: `render-spec-docs` became `render-docs`, `check-rendered-spec` became
  `check-rendered-docs`, the `specification/` output prefix became `docs/`, `method/` became `format/`, and
  `renderSpecificationSite` became `renderDocumentationSite`. `compile-spec`, `check-spec`, `verify-spec` and
  `spec-diff` keep their names because they do act on `spec/`.
  Finally the words: `正本`, `正準文書`, `正準 Markdown` and the user-facing `正準 scope 値` became `一次情報`,
  `一次情報文書` and plain wording wherever they named the documentation system, while the uses that name where
  data originates stayed; `版` became `バージョン` outside `docs/domain/` and `backend/`. Two tests enforce both
  rules, which `DOCUMENTATION_GUIDE.md` previously only stated.
- **Acceptance RED Evidence**:
  - **Test**: `mise run test-tools-file -- render-docs/src/render.test.ts`
  - **Requirement**: N/A: 生成ドキュメントサイトと文書の配置の変更であり、プロダクトの規範的な振る舞いを変えない。
  - **Observed Failure**: 16 tests failed. The rendering threw
    `generated pages are unreachable: specification/runbooks/async-jobs.html`, because a runbook was
    classified as a whole-system child and no navigation entry pointed at it.
  - **Detection Reason**: `validateSiteLinks` walks the link graph from `index.html`, so a document published
    without an entrance fails the build instead of shipping as an orphan page. The same assertion caught two
    later regressions: the collapse rule hiding the children of a branch that owns no page, and
    `docs/design/README.md` losing its only inbound link when its branch was dropped.
- **Unit RED Evidence**:
  - **Test**: `mise run test-tools-file -- render-docs/src/documentation-quality.test.ts`
  - **Requirement**: N/A: 文書の語彙を確かめるツール単体テストである。
  - **Observed Failure**: 47 citations were reported against an expected empty list, naming every file and
    line that used `正本`, `正準文書`, `正準 Markdown` or `正準 scope`.
  - **Detection Reason**: The assertion reports each citation with its path and line rather than a count, so a
    partial replacement is visible as the lines it left behind. It caught two of my own regressions later: a
    `git checkout` that reverted `docs/README.md`, and new comments that wrote `全体版`.
- **Change-Resistance Results**:
  低リスクの文書生成とツールの変更なので、変異テストは実施していない。
  代わりに、規則の分岐を外す故障を 3 件注入して確かめた。
  第一に、枝の子を現在位置で絞る実装（`details` へ移す前の形）では、
  `generated pages are unreachable: operations/runbooks/async-jobs.html` で生成が落ちた。
  子を常に HTML へ載せる規則がこの失敗を消し、`open` の判定だけが残った。
  第二に、設計の枝を平らにする当時の分岐を `tree.directories.map(directory)` へ戻すと、
  `renders system documents as a directory tree` が期待する形を見つけられずに落ちた（22 pass, 1 fail）。
  第三に、`docs/development/local-development.md` の置換を 1 行だけ戻すと、語彙のゲートがその行を
  引用して落ちた（4 pass, 1 fail）。
  文書の移動については、履歴のパスを写す前の状態が `spec-diff` で 627 行の「規範の追加」を出し、
  完了済み記録 8 件の `documentation_impact` が `check-repository` で落ちることを確かめた。
  写す実装を入れると、どちらも消えた。
- **Verification Results**:
  - `mise run spec-diff` - passed（normative specification change なし）
  - `mise run test-tools` - passed（568 tests）
  - `mise run typecheck-tools` - passed
  - `mise run lint-tools` - passed
  - `mise run check-spec` - passed
  - `mise run check-links` - passed（867 documents）
  - `mise run check-rendered-docs` - passed
  - `mise run render-docs` - passed（1090 pages, 195 documents）
  - `mise run check-work-items` - passed
  - `mise run verify` - passed
  - 展開後の `api/index.html` に `"$ref":` が 0 件であることを確かめた。`$$ref` は残り、
    打ち切りは 1 箇所である。
  - 公開 JSON を `url` で指す形、`blob:` URL、`spec` で渡す形は、いずれも `file://` では
    解決できないことを利用者の観測で確かめた上で退けた。
  - swagger-ui-dist 5.32.14 と 5.32.15 のバンドルを突き合わせ、`baseDoc` 18 箇所、
    `absolutifyPointer` 4 箇所、`extractFromDoc` 3 箇所がいずれも一致することを確かめた。
    依存の更新はこの症状の原因ではない。
  - `mise run serve-docs` - `/`、`/docs/design/index.html`、`/domain/index.html`、`/reference/index.html`、
    `/format/index.html`、`/operations/index.html`、`/api/index.html`、`/openapi/idmagic.openapi.json` が
    すべて 200。`/../mise.toml` は 404。
  - `mise run test-ui-e2e` - 実行できていない。E2E はスタックを起動するため、前回の中断で残った
    プロセスが `http://localhost:8082/health` を占有し、以降の起動が拒否された。変更した
    フロントエンドは `AdminSettingsPage.i18n.ts` の案内文 1 件とコメント 3 件だけで、
    `localized-ui.spec.ts` はこの文言を参照しない。`test-ui-unit` は `verify` の内側で通っている。
