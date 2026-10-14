---
status: completed
archived_from: 0d89fe6d56243c535512758cbada9f7034aeb79d
---

# 正規シナリオを起点にコードと work item を引ける索引と検索を追加する

仕様サイトに Traceability ページ（`spec/generated/docs/traceability/index.html`、サイドバーの
References から到達）を追加した。全 249 正規シナリオについて、定義位置へのアンカーリンク、ID を
書いているコード・テストのパス、言及している work item を表にする。参照はレンダリング時に作業ツリー
を走査して収集するため、authored な索引を持たず乖離しない。現状は 249 中 7 シナリオが参照を持つ。
併せて `just spec-where <term>` を追加した。仕様・コード/テスト・work item を横断して該当位置だけを
返し、仕様本文を丸ごと読ませない初手を提供する。`DEVELOPMENT.md` の Context economy をこの導線に
更新した。
当初案にあった「新規 REQ にテスト注記が無ければ失敗」というラチェット検査は入れていない。テストは
実在しており、検査が守るのは注記の付け忘れだけで、摩擦に見合わないと判断した。

全文は `git show 0d89fe6d56243c535512758cbada9f7034aeb79d:<このファイルのパス>` で読む。
