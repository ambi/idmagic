---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 管理画面ヘッダと UserDetails の情報階層を直す (ブランドリンク / アバターメニュー / アクションの優先度)

AdminShell のブランドを `/admin` へのリンクにし、ユーザー名とアバターを
Radix DropdownMenu のトリガに変更した。ログアウト単独アイコンは廃止し、
アカウント概要、パスワード変更、ログアウトをメニューに集約した。
UserDetails は編集を primary action にし、無効化・再有効化・削除を
kebab menu に移した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
