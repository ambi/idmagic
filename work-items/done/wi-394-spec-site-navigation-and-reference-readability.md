---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-22
priority: p2
depends_on: []
change_kind: tooling
spec_impact:
  kind: none
  reason: 生成される仕様サイトの見出し・並び・折りたたみ・参照ページの索引付けだけを変え、正規文書の内容も製品挙動も変えない。
---

# 生成仕様サイトのナビゲーションと参照ページを読める形に整える

## Motivation

HTML 出力された仕様サイトを読むと、案内が実際の文書構造を映していない箇所がいくつかある。

- context を開くと `glossary.md`、`scenarios.md` のようなファイル名が並ぶ。読み手が選ぶ手掛かりは「どの種類の内容か」であって、ファイル名ではない。しかも並びがアルファベット順で、`decisions.md` が `glossary.md` より前に来る。正規レイアウトが定めている「用語 → 規格 → 状態 → 決定 → 内部 → シナリオ」という読む順とずれている。
- Whole System も同じくアルファベット順で、語の定義である Glossary が Deployment や Persistence より後ろに沈む。
- Glossary の表で Term 列が 1 文字ずつ折り返され、`InterfaceStability` のような語が縦に割れて読めない。
- サイドバーに Method 3 件、Whole System 11 件、Contexts 21 件が常時並び、それ自体がスクロールを持つ。現在地を探す前にサイドバーを読む羽目になる。
- API Reference と Model Catalog は全体網羅で、context から自分の担当分へ降りる導線がない。Model Catalog に至っては全モデルが `IdMagic.Contract` 名前空間 1 つに属しているため、名前空間による見出しが 119 件のフラットな一覧を 1 つ作るだけで、索引として働いていない。

## Scope

- サイドバーの子項目のラベルを、ファイル名ではなく文書自身の H1 から作る。context 配下では文書名が context 名を頭に繰り返すので、その繰り返しだけを落とす。
- サイドバーと Home の並びを正規レイアウトの定義順にする。context 配下は `CONTEXT_DOCUMENTS`、Whole System 配下は `ROOT_DOCUMENTS` の順を正本とし、入力のアルファベット順に依存しない。
- サイドバーのグループをすべて `<details>` に揃える。既定で閉じるのは Method だけで、現在地が Method の文書のときは開く。
- Whole System の子を、Contexts と同じく「今いる場所の子だけ開く」規則に揃える。
- Glossary の Term 列が語の途中で折り返されないようにする。
- Model Catalog と Model 詳細の説明文を Markdown として描画する。TypeSpec の doc コメントは Markdown であり、`` `expires_at` `` がバッククォートのまま出ている。
- Model Catalog の Filter models を実際に動かす。入力欄はあるがサイトスクリプトが読み込まれておらず、既存の絞り込みが最初から無効だった。
- 各 context の index ページに、その context が所有する API 操作とモデルの一覧を生成する。API Reference と Model Catalog は全体網羅のまま残し、そこへ絞り込み付きで降りられるようにする。
- Model Catalog の見出しを、名前空間ではなく所有 context にする。context ごとにアンカーを持たせ、context ページから直接指せるようにする。

## Out of Scope

- context ごとに `api.html` / `models.html` を分割生成すること。全体網羅ページと二重管理になる。
- 正規 Markdown 側の見出しやファイル名の変更。
- サイドバーの検索、絞り込み、ツリーの永続化。
- Swagger UI の差し替えや API Reference のレンダリング方式の変更。

## Design

**ラベルは H1 から作り、ファイル名からは作らない。** 「拡張子を落として頭文字を大文字にする」でも読めるものにはなるが、`api-rules.md` が `Api Rules` に、`states.md` が `States` になる。文書は既に `# API Rules`、`# OAuth2 State Transitions` という H1 を持っており、ページを開いたときに見える名前と一致させたほうが迷いがない。context 配下の H1 は例外なく `<context の H1> ` で始まるので（21 context 全件で確認済み）、その接頭辞だけを落として `Glossary`、`State Transitions` を得る。接頭辞が一致しない文書が現れたら H1 をそのまま使う。

**並びの正本は `CONTEXT_DOCUMENTS` / `ROOT_DOCUMENTS`。** 現状は `main.ts` が入力パスを `sort()` しているだけで、順序が偶然アルファベット順になっている。並びを `render.ts` 側の属性にして、正規レイアウトの定義順を索引に引き写す。これで並びの意図が検査対象の場所に載り、入力順に左右されなくなる。順序を別表として持たない理由も同じで、`check` が既に持っている一覧が唯一の定義である。

**折りたたみの見た目は全グループで揃える。** 既定で閉じたいのは Method だけだが、Method だけを `<details>` にすると、開閉の三角が付くグループと付かないグループが並び、同じ階層のものが別種に見える。4 グループすべてを `<details>` にし、違いは初期状態だけにする。Method は 3 件だが、サイトの入口としては仕様本体より優先度が低く、常時開いている必要がない。Whole System と Contexts は既定で開いたままにする。Contexts の一覧そのものは常に見えているべきである。Whole System の子を現在地基準で畳んでも到達性は保たれる。Home のカードから Whole System へ入れば、その場で子が開く。

**doc コメントは Markdown として描画する。** TypeSpec の doc コメントは Markdown であり、`` `expires_at` `` のようなコード表記を含む。カタログはこれを HTML エスケープして出していたので、バッククォートがそのまま読み手に見えていた。インライン Markdown として描画し、生 HTML は無効のままにする。エスケープしていた頃と同じく、doc に紛れ込んだマークアップは実行されない。

**Term 列は `overflow-wrap` の適用範囲を狭めて直す。** `body` に効いている `overflow-wrap:anywhere` は表のセルにも継承され、最小内容幅が 1 文字になる。自動テーブルレイアウトはこれを見て Term 列をいくらでも詰められると判断し、長い定義列に幅を渡す。セルだけ `break-word` に戻せば最小内容幅が最長語になり、列が語より狭くならない。加えて Term 見出しを持つ表を描画時に見分けて第 1 列を `nowrap` にし、語が折り返される余地を残さない。CSS の列幅指定を書かないのは、Glossary 以外の表（状態遷移、規格）に同じ幅規則を押し付けたくないからである。

**context の所有関係は宣言位置から導く。** API 操作の所有 context は、`@tag` を宣言している名前空間のソースファイルが `spec/contexts/<slug>/` にあることから決まる。モデルも同様に宣言ファイルの位置で決まる。どちらも TypeSpec コンパイラーから取れるので、タグ名と context を対応づける表を新たに置く必要はない。登録簿を作らずに標準レイアウトから導出する、という既存の方針をそのまま延長する。`@tag` の正規表現探索でも同じ対応表は得られるが、TypeSpec の意味を正規表現で再実装することになるので採らない。

**絞り込みはクエリ文字列で渡す。** context ページから API Reference へは `api/index.html?tag=Jobs` で入り、Swagger UI の `filter` にその値を渡す。Swagger UI が生成する `#/Jobs` のアンカーは実行時にしか存在せず、生成リンク検査が参照先の id を確認できない。クエリ文字列なら参照先ページの存在検査は効いたまま残る。検査側はリンク解決時にクエリを落とす。Model Catalog へは context 見出しのアンカー `models/index.html#context-jobs` で入る。こちらは生成物に id があるのでフラグメントまで検査できる。

**サイドバーに API/Model の子項目は足さない。** 導線は context ページ本体に置く。サイドバーが長いという指摘を受けている場所に、context あたり 2 行を足す変更を同時に入れる筋はない。

## Plan

1. `render.ts` のサイドバーを直す（ラベル、並び、Method の `<details>`、Whole System の畳み込み）。
2. `site.css` の表セルの `overflow-wrap` と、Term 表の第 1 列の折り返しを直す。
3. `typespec-catalog.ts` に所有 context の導出とタグ→context の対応を足す。
4. Model Catalog を context ごとの見出しに組み替え、アンカーを付ける。
5. context の index ページに API 操作とモデルの節を生成し、絞り込み付きで参照ページへ繋ぐ。
6. 生成リンク検査でクエリ文字列を落とす。
7. `mise run render-spec-docs` で再生成し、出力を確認する。

## Tasks

- [x] T001 [Tooling] サイドバーのラベルを H1 由来にし、context 子の接頭辞を落とす。
- [x] T002 [Tooling] サイドバーと Home の並びを `CONTEXT_DOCUMENTS` / `ROOT_DOCUMENTS` の順にする。
- [x] T003 [Tooling] 全グループを `<details>` に揃え、Method だけ既定で閉じ、Whole System の子を現在地基準で開く。
- [x] T004 [Tooling] 表セルの `overflow-wrap` を戻し、Term 表の第 1 列を折り返さない。
- [x] T005 [Tooling] TypeSpec カタログに所有 context とタグ対応を足す。
- [x] T006 [Tooling] Model Catalog を context 見出しに組み替え、アンカーを付ける。
- [x] T007 [Tooling] context ページに API 操作とモデルの節を生成し、絞り込みリンクを繋ぐ。
- [x] T008 [Tooling] doc コメントをインライン Markdown として描画する。
- [x] T009 [Tooling] Model Catalog にサイトスクリプトを読み込ませ、Filter models を動かす。
- [x] T010 [Verify] `tools/render-spec-docs` のテストを更新し、`mise run verify` を通す。

## Verification

- `mise run test-tools`
- `mise run render-spec-docs`
- `mise run verify`

## Risk Notes

- 生成物のみの変更で、バックエンドとフロントエンドには触れない。
- サイドバーから常時見える項目が減るので、到達性検査（`validateSiteLinks`）が全ページ到達を保証し続けることをテストで押さえる。
- `@tag` を持たない context、または context 外で宣言されたモデルが将来現れても、context 別の節を空にして全体網羅ページへ落ちるだけにする。

## Completion

- **Completed At**: 2026-08-22
- **Summary**:
  生成仕様サイトのサイドバーが、ファイル名のアルファベット順ではなく、正規レイアウトが定める順序と各文書の H1 を示すようになった。グループは 4 つとも同じ折りたたみで、Method だけが既定で閉じ、Whole System の子は現在地が Whole System のときだけ並ぶ。Glossary の Term 列は語の途中で折り返されない。各 context の index ページは、自分が所有する API 操作とモデルを持ち、タグで絞った API Reference と Model Catalog の該当見出しへ降りられる。Model Catalog の見出しは、意味を持たない単一名前空間ではなく所有 context になった。TypeSpec の doc コメントは Markdown として描画される。Model Catalog の Filter models は、サイトスクリプトが読み込まれていなかったため入力しても何も起きない状態だったが、実際に絞り込むようになった。
- **Verification Results**:
  - `mise run test-tools` - passed (156 tests)
  - Model Catalog の絞り込みを happy-dom 上で実行し、836 件が `expires_at` で 6 件、`jobs` で 25 件に絞られ、空になったグループが隠れることを確認
  - `mise run render-spec-docs` - passed (977 pages, 137 documents, 330 operations, 836 TypeSpec symbols)
  - `mise run verify` - passed
