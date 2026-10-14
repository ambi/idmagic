---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# WsFederation が宣言する標準 6 行にテストを対応付け、標準の被覆台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
差分は `docs/domain/ws-federation/standards.md` の 6 行に対する被覆の状態である。6 行すべてが
その行の `Statement` を区別できる入力と観測を持つテストを得て `tools/check/standards-coverage-debt.json`
から消え、台帳は 17 件から 11 件になった。名指しを持つ id は 270 件から 276 件へ増えた。
新設したテストは Go 6 件（部分試験を数えると 14 件）で、**製品コードは 1 行も変わっていない**。
**この文書は「テストが揃っているので注記だけで終わる」という Risk Notes の予想が外れた。**
6 ファイル 14 テストが既にあったが、T001 の棚卸しでは 6 行すべてが区別できていなかった。
`WSS-UsernameTokenPassword` には誤ったパスワードのテストが存在せず、`WSTrust13-IssueBearer` は
`SubjectConfirmation` を 1 箇所も読んでおらず、`WSAddressing-MessageIDToAction` の 3 つのうち
`Action` には例示テストが無く、`excluded` の 2 行は観測が逆向きになっていなかった。
注記だけで消化できた行はゼロである。
**欠陥は 1 件も見つからなかった。** 6 行はいずれも宣言した採用を満たしており、切り出した work item は無い。
既存テストを 2 件整理した。`TestWsTrustUsernameMixed_RejectsMismatchedTo` は
`TestWsTrustAddressing_ValidatesMessageIDToAndAction` の `To` の部分試験に完全に含まれるので削除し、
`TestWsTrustUsernameMixed_RejectsExpiredTimestampAndReplay` はリプレイの半分を同じ試験へ移して
`TestWsTrustUsernameMixed_RejectsExpiredTimestamp` に絞った。同じ判断を強さの違う 2 つのテストが
観測する状態を残さないためである。
**測ったことが 2 つある。** 1 つは `sd` の複数行置換で、パターンが複数行にわたると一致せず、
終了状態も 0 のまま何も置換しない。故障 F4 と F11 はこれで「検出されず」と出たが、実際には注入自体が
起きていなかった。`perl -0pi -e` で入れ直すと 2 件とも検出された。**注入した故障が本当に入ったことを
`git diff --stat` で確かめない限り、「検出されず」は読めない。** もう 1 つはコンパイルの通らない故障で、
F7 の最初の形（条件から `!ok` を落とす）は `declared and not used: ok` でビルドが落ちた。ビルド失敗は
`FAIL` を出すので、粗い判定では「検出された」と読めてしまう。`(!ok && false)` に変えて入れ直した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
