---
status: completed
archived_from: edcdcb174b3f9c7b70667470e8a721e51acbd42f
---

# 作業項目の識別番号を、未使用の 5 桁から無作為に選ぶ

`mise run spec-diff` は `no normative specification change against main` を返す。
変更は起票の側の規則、それを支える検査と `mise` タスク、既存 3 組の改名にとどまり、製品にも規範文書にも届かない。
新規の作業項目は `mise run work-item-number` が出す未使用の 5 桁を使う。
**番号の衝突を見る検査は、どこにも無かった。** 作業項目の id は拡張子を除いたファイル名全体なので、既存の重複検査は題名が違えば別の記録として通す。
実際、`work-items/done/` には 112、126、127 を共有する 3 組が存在したまま、`mise run check-work-items` は緑だった。
**例外は置かず、3 組を解消した。** 起票時の依頼では既知の例外として登録する予定だったが、例外は報告されない衝突を恒久的に残し、同じ番号を引いた将来の起票まで見逃す恐れがある。
各組のうち git に後から現れた側を、`wi-3`、`wi-42`、`wi-245` へ移した。
**移す先を 5 桁にしなかった理由は、スキーマが番号を時期の代理として読むからである。** 作業項目スキーマは 410 以上に `evidence_policy` を、452 以上に `documentation_impact` を要求する。
2026 年 7 月に完了した 3 件を 5 桁へ移した時点で `mise run check-work-items` は 18 件の schema 違反を報告した。
番号がその代理として働き続ける 410 未満の空き番号のうち、`git log --diff-filter=A --all` が一度も記録していない 3、42、245 を選んだ。
同じ条件の空き番号 269 は過去に `wi-269-audit-events-api-500-fix.md` として存在した形跡があり、外れた。
**参照の張り替えは機械的には終わらない。** 完全な id を含む 18 ファイルは置換で済んだが、`wi-126` のような番号だけの言及は、どちらの記録を指すかが文脈にしかない。
Go と TypeScript のコメント 5 箇所を含む 30 箇所あまりを読み分け、`ADR-084`（`wi-127-postgres-column-type-policy`）や ADR-086（`wi-126-admin-and-account-ui-consistency-and-navigation-policy`）のように番号を保つ側を指すものはそのまま残した。
**空き枠の警告は検査を落とさない。** `CheckOutcome` には警告の経路がないため、`checkWorkItems` の `ok` の根拠を所見の件数へ移し、警告は所見の有無にかかわらず出力へ並べた。

全文は `git show edcdcb174b3f9c7b70667470e8a721e51acbd42f:<このファイルのパス>` で読む。
