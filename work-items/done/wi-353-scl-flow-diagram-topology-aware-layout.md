---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# Flows 派生図をエッジのトポロジに沿って配置し、自己ループと flow 横断遷移を正しく描画する

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
SCLスキーマ(`spec/scl.yaml`/`docs/domain/*.yaml`/`scl-v3.schema.json`)は
Out of Scope どおり変更していない。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
