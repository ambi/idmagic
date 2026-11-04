---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-05
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/sharedsignals/features/receiver.md, requirement: REQ-SHAREDSIGNALS-010 }
  - { path: docs/modules/sharedsignals/standards.md, requirement: RFC9493-SUBID-FORMAT }
---

# 受信した SET の主体を、最上位の `sub_id` から解決する

## 動機

Shared Signals Framework 1.0 §3.1 は、SET の主体を最上位の `sub_id` クレームで表すことを MUST とする。
§3.1.1 は、既存の CAEP と RISC のイベントがイベント内の `subject` を使ってもよいとしつつ、最上位の `sub_id` を必ず含めるよう送信側へ求める。
§3.1.2 は、新しいイベントの型にイベント内の `subject` を使わせない。
したがって、イベント内の `subject` を省き `sub_id` だけを持つ SET は標準に適合する。

`extractCaepEventAndSubject` は、イベント内の `subject` だけを読み、ない SET を主体を解決できないものとして拒否する。
SSF の宣言は送信側の `SSF10-SUB-ID` だけで、受信側の標準要件はない。
[[wi-77372-inventory-standards-acceptance-rules]] の棚卸しで見つけた。

## 対象範囲

- 最上位の `sub_id` を主体として読み、`format` の解釈と `trusted_issuer` の照合は現在の規則（`RFC9493-SUBID-FORMAT`、`RFC9493-SUBID-ISS-SUB`）を適用する。
- 最上位の `sub_id` とイベント内の `subject` の両方がある場合の扱いを決める。食い違えば拒否する案を基本にする。
- IdMagic の送信側が書く独自の `subject` の形を、引き続き受け入れる。
- `standards.md` に受信側の SSF の標準要件を加え、`sub_id` だけの SET を受信の入口へ送る試験を加える。

## 対象外

- 送信側の SSF 1.0 への適合。[[wi-36958-conform-to-shared-signals-framework-1-0]] が扱う。
- `complex` などの未対応の `format`。`RFC9493-SUBID-FORMAT` が拒否を宣言している。

## タスク

- [ ] T001 [Spec] 受信側の標準要件と REQ-SHAREDSIGNALS-010 の例を加える。
- [ ] T002 [Acceptance] `sub_id` だけの SET で RED を確かめる。
- [ ] T003 [App] 主体の解決を直し、GREEN にする。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- SET の受信の HTTP の受け入れ試験：失効の反映と、受信したイベントの記録を観測するため。

## リスク

主体の取り出し口が二つになるため、片方だけに `trusted_issuer` の照合を書くと、別の発行者の主体を失効できる。
両方の取り出し口を一つの解決関数へ通す。
