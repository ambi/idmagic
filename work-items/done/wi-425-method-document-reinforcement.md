---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 方法論文書に、判断の可逆性・テストダブルの方針・記録の根拠を足す

`mise run spec-diff` は `no normative specification change against main` を返す。製品の規範シナリオと TypeSpec は変わっていない。変わったのは方法論の平面で、`risk` から独立した `reversibility` 軸が work item の frontmatter に加わり、`irreversible` は risk が `low` でも完了前の独立検証を要求する。この規則は証拠契約・検証のはしご 4 段目・完了記録の Independent Verification 欄・スキーマの完了時条件の 4 箇所を貫通し、最後のものだけが機械検査になる。`docs/development/specification-first-workflow.md` は Refactoring 節を得て、リファクタリングの定義（テストが動かないこと）、行う時（GREEN 直後）、止める時（次の振る舞いを足せる形になった時）と、リファクタリングのみの work item が not-applicable 経路で何を代替検査として記録するかを持つ。独立検証がペアレビューの代替であることと、Parnas と Clements の合理的再構成が記録の順序の根拠であることが明記された。`docs/development/testing.md` はテストダブル節を得て、フェイク・記録するダブル・スタブの選択を「効果を読み戻せるか」の一軸で決め、回数が仕様そのものである場合を第三の枠として持つ。同節は `db_memory` と `db_postgres` の同値性を保証する共有契約テストが存在しないことを負債として明記する。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
