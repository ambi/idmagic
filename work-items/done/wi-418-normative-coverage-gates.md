---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 規範とシナリオの被覆をゲートにする

`mise run spec-diff` は `no normative specification change against main` を返す。規範行もシナリオも増減せず、`docs/domain/standards.md` の冒頭の文だけが実態に合う表現へ変わった。意味の差は仕様の内容ではなく、仕様と実装の対応が検査されるかどうかにある。`mise run check-spec` は今後、規範行 154 件とシナリオ 306 件のそれぞれについて、製品のテストがその ID を名指ししているか、負債台帳が理由付きで保持しているかのどちらかを要求する。着手時点で名指しがあるのは 134 件で、残りは `tools/check/standards-coverage-debt.json`（134 件）、`tools/check/example-coverage-debt.json`（89 件）、既存の `tools/check/security-refusal-debt.json`（102 件）が保持する。3 つの台帳は互いに素であり、縮む方向にしか動かない。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
