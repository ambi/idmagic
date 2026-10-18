---
status: completed
---

# User と Group の SCIM 由来の表示を IdManagement の照会ポートから取得する

`spec-diff` は規範仕様の差分なしを示した。
User と Group の詳細の SCIM 由来を、IdManagement が公開する読取りだけのポートへ移し、Sourcing の既存の対応照会を正式な HTTP の経路へ結んだ。
対象のテナント、由来の表示、未配線と照会失敗時の扱い、永続状態とイベントは維持し、既存テストを変更していない。
IdManagement から Sourcing への逆向きの参照をなくし、Sourcing を含む循環の3件を負債台帳から除いた。
先行する wi-61185 と合わせて循環の負債は53件から49件になった。
SCIM の書き込みの原子性と監査・クォータの是正は wi-11577 の対象として残る。

全文は `git log -1 -- work-items/active/wi-64444-read-user-source-through-an-identity-management-port.md` が示すコミットの親、またはそのコミットで読む。
