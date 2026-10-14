---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 管理画面・ログイン画面の使い勝手を直す (forgot link 位置 / 編集ダイアログ統合 / dashboard / refresh ボタン)

4 件の UX ノイズを 1 本の WI として 4 commit に分けて直した。
どれも単独では小さいが、放置すると管理者と end user の体験を地味に
削り続ける箇所だったため、まとめて整理した。
backend (Go) には触らず、すべて UI のみで完結している。
part 1 (LoginPage): "パスワードを忘れた場合" リンクを password 入力欄の
下、submit ボタンの直後に移した。tab 順が
`username → password → submit → forgot link` で安定した。
part 2 (UserEditorDialog): 属性編集とロール編集の 2 ダイアログを統合し、
Profile セクションと Roles セクションを 1 画面に並べた。ロールに変更が
ある場合だけ "review changes" 二段階確認を出すという原則は維持した。
part 3 (AdminDashboardPage): `/admin` を独立した landing page にした。
カード 4 枚 (users / clients / granted consents / 24h audit events) +
直近 5 件の監査イベント + クイックリンクで、何が起きているかを 1 画面で
把握できるようにした。集計用の新規 endpoint は作らず、既存 list endpoint
を Promise.all で叩いて UI 側で count する形に留めた。
part 4 (refresh icon): admin 5 ページの "更新" / "再読込" テキストボタンを
IconRefresh のみのアイコンボタンに置き換えた。"...を追加" ボタンの隣で
primary 操作と並んで見えていた refresh が、副次的な affordance として
正しく扱われる位置に収まった。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
