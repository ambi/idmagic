---
status: completed
---

# 効果のない Agent のロールを画面、API、データから取り除く

Agent の `roles` を要求、応答、ドメイン、保存列、管理画面から除いた。
REQ-IDMANAGEMENT-073 と REQ-IDMANAGEMENT-075 をロールのない登録と更新へ揃え、REQ-IDMANAGEMENT-032 と REQ-IDMANAGEMENT-033 の適用対象を User と Group に限った。
REQ-IDMANAGEMENT-101 と TypeSpec の要求、応答、イベント、エラーの説明も同期した。

未知の `roles` は既存の JSON デコードの規則で無視する。
ロールだけの更新は、保存した内容、更新時刻とイベントを変えない。
User と Group のロールの規則は保ち、Agent の分類ラベルは対象外とした。

公開前のため非推奨の期間や移行は設けず、宣言するスキーマから列を除いた。
main のデモ改善を取り込み、所有者の選択、公開鍵の登録と管理画面だけで設定する経路を維持した。
