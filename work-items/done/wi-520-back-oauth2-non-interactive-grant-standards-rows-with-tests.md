---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 非対話グラント（Device / CIBA）が宣言する標準 8 行にテストを対応付け、標準の被覆台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。規範の差分は無い。
変わったのは、非対話グラントが宣言する 8 行に対する観測の有無である。
`tools/check/standards-coverage-debt.json` の `untested` は 38 件から 30 件へ減り、本項目が持つ
8 件は 1 件残らず消えた。テストが名指す ID は 257 件になった。製品コードは変えていない。
増えたのは backend のテスト 7 本と frontend の注記 1 件である。
着手前の想定と実測が食い違ったのは `excluded` の 3 行である。想定は「その機能を使う
リクエストが通らないこと」と「`auth_req_id` が発行されていないこと」の対だったが、
`/bc-authorize` は `user_code` も `client_notification_token` も無視して 200 を返し、
`backchannel_token_delivery_mode` を名指した動的クライアント登録も 201 で成功する。
RFC 6749 §3.1 が未知のリクエストパラメーターを無視することを求めている以上、これは
宣言した採用の未達ではないので、観測の型を「能力を広告していないこと」と「機能を
名指しても結果が名指さない場合と一致すること」の対へ決め直した。
`CIBA-CORE-SIGNED-REQUEST` だけは拒否が実際に起きるので拒否の型で読み、平文と矛盾する
`request` を添えたとき平文側が効くことで、JWT を読んで採用しない実装と、そもそも
読んでいない実装とを区別できるようにした。
ポーリングの 2 行は時計を差し替えられない入口を相手にする。有効期間の幅を保ったまま
レコード全体を過去へずらす形にしたので、製品が作らないレコードを作ることなく、
`slow_down` が終端でないことと期限切れの両方を 1 本のコードで読めている。
`CIBA-CORE-BINDING-MESSAGE` は「運ぶこと」と「見せること」の 2 つを言っているので、
承認一覧 API と承認画面の 2 か所に同じ id を対応付けた。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
