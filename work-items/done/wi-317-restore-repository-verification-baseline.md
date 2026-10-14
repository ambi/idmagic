---
status: completed
archived_from: 7c01ca6d91eba16858dd22cb169330da29b4785c
---

# wi-314 の最終検証で発見した既存の仕様整合・DBテスト分離・UI複雑度違反を解消する

`AgentStatus.Disabled` の wire alias 欠落を SCL 正本で補い、派生 HTML を再生成した。
TOTP の全表 plaintext 検査は条件を弱めず、共有 embedded PostgreSQL のテスト fixture を
分離した。Account profile は属性表示・編集責務を専用モジュールへ抽出して page を 509 行から
304 行へ縮小し、不要になった complexity debt 例外を削除した。本番 API・永続化契約・画面挙動は
変更していない。
test-first からの逸脱 (self-attest): T001/T002 は `just verify` の既存 RED を確認してから修正。
T003 は既存 architecture RED に対する構造リファクタリングで、抽出後に属性表示・編集の回帰
テストを追加したため、新規テスト単体の RED は先行確認していない。

全文は `git show 7c01ca6d91eba16858dd22cb169330da29b4785c:<このファイルのパス>` で読む。
