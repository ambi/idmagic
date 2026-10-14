---
status: completed
archived_from: 924c770d7f542c1d44715f6f01d44b72e53153b6
---

# 照合が誤削除ガードの閾値を超える deprovision を計画したら、何も作らずに接続を隔離する

`mise run spec-diff` が示す規範の差分は `REQ-PROVISIONING-011` だけである。
`EX-PROVISIONING-011-01` と `-02` は、誤削除ガードが検出する場所を「full resync」から「照合」へ改めた。
`EX-PROVISIONING-011-03`（割合の閾値を超える）と `-04`（閾値ちょうどでは隔離しない）を足した。
実装では、照合の計画が上限で打ち切る前の deprovision を数え、件数または割合の閾値を超えたら何も作らずに隔離の理由を返す。
照合のユースケースは最新の接続を隔離し、保存の後に `ConnectionQuarantined` を発行する。
隔離の保存と発行は連続失敗の経路と共有する `quarantineConnection` に取り出した。
Full Resync、書き込み時の捕捉、猶予期間つき削除の実体化にはガードを置かない。その理由は `internals.md` の「誤削除ガード」節にある。
`EX-PROVISIONING-011-01` は通知の行を含むので、台帳の `blocked_by` を [[wi-33641-notify-quarantine-to-the-notification-email]] へ付け替えた。

全文は `git show 924c770d7f542c1d44715f6f01d44b72e53153b6:<このファイルのパス>` で読む。
