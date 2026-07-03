---
id: idp-wi-77-strip-wi-adr-references-from-scl
title: "scl.yaml から wi / ADR / commit 参照を除去し純粋な仕様文にする"
created_at: 2026-06-27
authors: ["tn"]
status: completed
risk: low
---
# Motivation
scl.yaml はすべての層の最内にある純粋な仕様正本であり、外側のレイヤ (work-item /
ADR / git 履歴) に依存してはならない。しかし現状の scl.yaml には description などに
`(wi-69)`、`(ADR-064)`、`(commit a5e2ec8)` のような参照が ~345 箇所混入している。

これらは「どの wi で追加されたか」「どの ADR の決定か」という経緯・決定方針であって、
仕様の意味そのものではない。経緯は work-item / ADR 側が持つべきで、scl.yaml はより静的で
純粋な仕様文として記述されるべきである。本 WI は scl.yaml 全体から wi / ADR / commit 参照を
機械的かつ慎重に除去し、各文を意味を保ったまま平叙文へ整える。

# Scope
- **scl**: scl.yaml 内の `(wi-NN)` / `(ADR-NNN)` / `(commit <hash>)` / `[[wi-...]]` / `[[ADR-...]]` 参照を除去し、残る文を自然な仕様文に整える。意味は保持する。
- **verification_only**: 仕様の意味不変を yaml-check と既存テストで担保する。

# Out of Scope
- [[wi-76-fold-advanced-protocol-settings-into-application-editor]] が触れる型の意味変更。 本 WI は参照除去のみで、フィールド追加・削除はしない。
- work-item / ADR 側の記述変更。経緯はそちらに残す。
- 生成 HTML のレイアウト変更 (参照除去に伴う再生成は行う)。

# Verification
- [object Object]
- [object Object]
- 手動: `grep -nE 'wi-[0-9]|ADR-[0-9]|commit [0-9a-f]{7}' spec/scl.yaml` が 0 件になる ことを確認する。

# Risk Notes
機械的な除去だが、文に織り込まれた参照を雑に消すと文意が壊れる。括弧の閉じ忘れや
二重スペースなど整形の崩れにも注意する。型・フィールド・制約の定義そのものには触れない。

# Completion
- **Completed At**: 2026-06-28
- **Summary**:
  SCL 正本から wi / ADR / commit 参照を全除去し、純粋な仕様文に整えた。本 WI
  起票後に context が物理分割 (wi-31) されたため、scope の「scl.yaml」を SCL spec
  全体 (spec/scl.yaml + spec/contexts/*.yaml) と解釈して適用した。除去前は 6 ファイル
  計 138 箇所の参照があり、除去後は 0 件。型・フィールド・制約・列挙・scenario の
  意味は不変。
- **Verification Results**:
  - [object Object]
  - [object Object]
  - 手動: `grep -rnE 'wi-[0-9]|ADR-[0-9]|commit [0-9a-f]{7}' spec/scl.yaml spec/contexts/*.yaml` が 0 件、整形崩れ (空括弧・行頭句点・区切り残り) も 0 件であることを確認した。
