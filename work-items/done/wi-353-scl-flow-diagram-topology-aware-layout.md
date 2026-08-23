---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-10
depends_on: []
---

# Flows 派生図をエッジのトポロジに沿って配置し、自己ループと flow 横断遷移を正しく描画する

## Motivation

`tools/scl-to-html/src/render-scl.ts` の `renderDiagram` は、`flows`/`states` から派生する
SVG 図のノード配置を `√n × √n` の単純グリッドに index 順で並べているだけで、エッジの
接続関係を一切見ていない。この実装は以下の2点で実データを正しく表現できていない。

1. 同一 view への複数遷移(自己ループ、`to` を明示しない、あるいは同じ view を指す
   `action`)は `from === to` になり、`sx=sy=cx,cy` で線分の長さが 0 になる。結果として
   複数のラベルが完全に同一座標へ積み重なり判読できない。
   実例: `spec/contexts/tenancy.yaml` の `SystemTenantManagement` flow は
   `SystemTenants` view の5アクション(`create`/`update`/`disable`/`enable`/
   `update_quota`)全てが同一 view に留まる設計で、5本の線と5個のラベルが
   完全に重なる。
2. `flows.<Name>.views.<View>.does[].to` は SCL 上、同一ファイル内の**別 flow の
   view** を指してよい(flow をまたぐ画面遷移。`SPECIFICATION_CORE_LANGUAGE.md`
   §3.8.1 が明示的に許容している)。しかし `renderFlows`
   (`tools/scl-to-html/src/render-scl.ts:1047`)は診断対象の flow 自身の
   `views` だけを `DiagramNode`化するため、`renderDiagram` 内の
   `positions.get(edge.to)` が解決できず、その edge は無言で描画から
   落ちる(`if (!from || !to) return ''`)。
   実例: `spec/contexts/authentication.yaml` の `Login` flow は5 action中3つ
   (`Totp`/`MfaEnrollment`/`ForgotPassword` への遷移)が別 flow の view を
   指しており、生成される図には残り2本(`external: true` の action)しか
   現れない。ログインという最重要フローの遷移が図では6割欠落する。
3. カードのフォールバック表(`does` テーブル)の「To」列はプレーンテキストで、
   `Interface` 列と異なりハイパーリンクされていない。cross-flow 遷移先を
   読者が手動で探す必要がある。
4. `tools/scl-to-html/src/render.test.ts` の diagram 関連アサーションは
   `id="diagram-flow-demo"` の存在確認のみで、自己ループ・並行/重複 edge・
   flow 横断遷移のケースを一切カバーしていない。

これは `spec/scl.yaml`・`docs/contexts/*.yaml`・`tools/check/schemas/scl-v3.schema.json`
のいずれにも起因しない、`tools/scl-to-html` 側の描画実装の不具合であり、SCL の
スキーマ自体(views/sees/does 記法、ADR-112 で確立)は変更しない。

`renderDiagram` は `states` セクションの図とも共有されているため、本修正は
states の図にも副次的に効くが、対応が急務なのは「同一 view から複数 action が
分岐する」形が頻出する flows 側であり、本 work item のスコープも flows の
実データ・テストに限定する。

## Scope

- `tools/scl-to-html/src/render-scl.ts` の `renderDiagram`(ノード配置・edge 描画)
  と `renderFlows`(diagram 用ノード/edge の組み立て、does テーブルの To 列)
- `tools/scl-to-html/src/render.test.ts`(自己ループ・並行 edge・flow 横断遷移を
  再現するテストケースの追加)

## Out of Scope

- `spec/scl.yaml` / `docs/contexts/*.yaml` の `flows` セクションの記法変更
  (views/sees/does 記法自体は変更しない)
- `tools/check/schemas/scl-v3.schema.json` / `tools/check/src/scl-semantics.ts`
  の変更(到達可能性・参照解決ロジックは現状のままで問題なく、今回直すのは
  描画のみ)
- `states` セクションの実データ・専用テストの追加(`renderDiagram` 共有による
  副次的な改善は享受するが、states 側の実データ検証は別途)
- Authorization/Objectives/Scenarios のスキーマ拡張(別途検討中、本 wi の対象外)

## Design

**レイアウト**: 現在の index 順グリッド配置を、エッジ方向を考慮した簡易階層
レイアウトに置き換える。`entry` view をランク0とし、`does[].to` を辿った
BFS到達距離でランクを割り当て、同一ランク内は登場順に横方向へ並べる
(dagre/ELK/Mermaid が採用する Sugiyama 系レイアウトの簡易版。フル実装の
交差最小化までは行わず、ランク分けのみで十分に自己ループ問題は解消できる)。

**自己ループ**: `edge.from === edge.to` の場合は直線ではなく、ノード右上から
出て右上に戻る円弧(SVG `path` の `A` コマンド)で描画し、同一ノードに対する
複数の自己ループはオフセットを変えて重ならないようにする。ラベルも円弧に
沿わせて分散配置する。

**flow 横断遷移**: `renderFlows` で1つの flow の diagram を組み立てる際、
`to` が自 flow の `views` に存在しない場合は「他 flow の view」として
別種のノード(`kind: 'cross-flow'`、ラベルに所属 flow 名を付記、
href は該当 flow のカードアンカーへリンク)を動的に追加してから
`renderDiagram` に渡す。現状の `__external__` 合成ノードと同じパターンを
「flow 外部だが SCL 内」のケースに拡張する形。

**並行 edge**: 同一 `from`→`to` の組に対して複数 edge がある場合(ラベルが
異なる別々の action が同じ view へ遷移する場合)、各 edge の直線をわずかに
オフセットさせるか曲線化し、ラベルが重ならないよう縦方向にずらす。

**does テーブルの To 列**: `renderFlows` のカード生成部で、`to` が
自 flow 内 view なら現状通りプレーンテキスト、他 flow の view と判定できた
場合は該当 view のノードアンカー(または cross-flow ノードの href)へ
リンクする。

**検討したが採用しない代替案**:
- dagre/ELK 等の外部グラフレイアウトライブラリの導入 —
  `tools/scl-to-html` は依存ゼロの deterministic レンダラであることが
  設計上の前提(ファイル冒頭コメント「No I/O, deterministic」)であり、
  新規ランタイム依存を持ち込むコストが今回の不具合の規模に見合わない。
  簡易ランクベースレイアウトで自己ループ・flow 横断の実害は解消できる。
- SCL 自体を Mermaid 等の別フォーマットへ分解 — ADR-103/wi-183 で
  ソース・オブ・トゥルースの複数形式化として既に却下済みであり、
  本件は出力側の描画バグに過ぎないため再検討しない。

## Plan

1. `renderDiagram` にランクベースのノード配置(BFS distance from entry)と
   自己ループ/並行 edge 描画を実装する。
2. `renderFlows` で cross-flow ノードの動的追加と does テーブルの To 列
   リンク化を実装する。
3. `render.test.ts` の `sampleScl()` 由来の既存フローを壊さない形で、
   自己ループ(同一 view への複数 action)・flow 横断遷移(2 flow 構成)を
   持つ新規フィクスチャ/テストケースを追加し、生成された SVG に重複座標が
   無いこと・cross-flow edge が描画されること・To 列がリンクされることを
   アサートする。
4. `just scl-render` で実データ(`tenancy.yaml` の `SystemTenantManagement`、
   `authentication.yaml` の `Login`)を再生成し、`spec/idmagic.full.html` を
   目視確認する。

## Tasks

- [x] T003 [Test] `render.test.ts` に自己ループ・並行edge・flow横断遷移の
      再現テストを追加する。RED: 4件とも意図通り失敗を確認
      (`does not collapse self-loop edges onto identical coordinates` /
      `draws cross-flow transitions instead of silently dropping them` /
      `offsets parallel edges between the same view pair` /
      `links the does-table To column for cross-flow navigation`)。
- [x] T001 [Tool] `renderDiagram` をランクベースレイアウト
      (`rankNodes`: entry/`initial` node から BFS 到達距離でランク付け、
      同一ランク内は列方向に並べる)+自己ループ(円弧 path、複数ループは
      高さをずらす)/並行edge(同一 from/to ペアを法線方向にオフセット)
      描画に書き換える → GREEN。
- [x] T002 [Tool] `renderFlows` で flow 横断遷移先を cross-flow ノードとして
      diagram に追加し(`viewOwner` map で所属 flow を解決)、does テーブルの
      To 列を該当 flow へのリンクにする → GREEN。
- [x] T004 [Verify] `just scl-render` で実データを再生成し、`tenancy.yaml`/
      `authentication.yaml` の図を目視確認する。`SystemTenantManagement` は
      5本の自己ループが高さの異なる円弧として分離され、`Login` は cross-flow
      遷移3本(`Totp`/`MfaEnrollment`/`ForgotPassword`)が図と does テーブルの
      両方に現れ、does テーブルの To 列がリンク化されていることを確認した。

## Verification

- `just test-tools`
- `just typecheck-tools`
- `just check-scl`
- `just scl-render`
- `just verify-ui`
- `just check-ids`

## Risk Notes

- `renderDiagram` は `states` セクションの図とも共有されているため、レイアウト
  変更は states の見た目にも影響する。state machine は fan-out が少なく
  破壊的な見た目の変化は想定していないが、`scl-render` 後の目視確認で
  states 側の図も壊れていないか確認する。
- 簡易ランクレイアウトは本格的な交差最小化を行わないため、view 数・分岐数が
  非常に多い flow では依然として交差が残り得る。今回のスコープは
  「自己ループ・並行edgeの完全重なり」と「flow横断遷移の消失」という
  実害の解消であり、美観上の最適レイアウトは目標としない。

## Completion

- **Completed At**: 2026-08-10
- **Summary**:
  `tools/scl-to-html/src/render-scl.ts` の `renderDiagram` を、index 順の
  `√n × √n` グリッド配置から、`rankNodes`(entry/`initial` node から
  `does[].to` を辿った BFS 到達距離でランク付けし、同一ランク内は列方向に
  並べる)を使った階層レイアウトへ書き換えた。`edge.from === edge.to` の
  自己ループは直線ではなく円弧 `path` で描画し、同一ノードへの複数自己ループは
  弧の高さをずらして重ならないようにした(上部余白 `topPad` を最大ループ数に
  応じて動的に確保)。同一 `from`/`to` ペアを持つ並行 edge は法線方向へ
  オフセットして分離した。`renderFlows` には `viewOwner`(view名→所属flow名の
  逆引き)を追加し、`does[].to` が自 flow 外の view を指す場合(flow をまたぐ
  navigation、`SPECIFICATION_CORE_LANGUAGE.md` §3.8.1 が許容)に
  `kind: 'cross-flow'` の合成ノードを diagram へ動的追加して edge の
  無言ドロップを解消し、does テーブルの「To」列もその view の所属 flow への
  リンクに変えた(`tools/scl-to-html/src/page.ts` に対応 CSS も追加)。
  `render.test.ts` に自己ループ・並行edge・cross-flow遷移・To列リンクの
  4テストを追加し、修正前に意図通り失敗すること(RED)を確認してから実装した。
  実データ(`spec/contexts/tenancy.yaml` の `SystemTenantManagement`、
  `spec/contexts/authentication.yaml` の `Login`)で `just scl-render` 後の
  `spec/idmagic.full.html` を確認し、5本の自己ループが分離した円弧として
  描画され、`Login` flow の cross-flow 遷移3本(`Totp`/`MfaEnrollment`/
  `ForgotPassword`)が図とテーブル双方に正しく現れることを確認した。
  SCLスキーマ(`spec/scl.yaml`/`docs/contexts/*.yaml`/`scl-v3.schema.json`)は
  Out of Scope どおり変更していない。
- **Verification Results**:
  - `just test-tools` - passed (301 tests)
  - `just typecheck-tools` - passed
  - `just check-scl` - passed (27 files)
  - `just scl-render` - passed
  - `just verify-ui` - passed (562 tests, build succeeded)
  - `just check-ids` - passed (510 record ids)
