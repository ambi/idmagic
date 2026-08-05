---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-026: NIST SP 800-63B-4 整合のパスワードポリシー

## コンテキスト

Phase 0 で IdP に求められる「商用品質のパスワードポリシー」を定義する。
README ロードマップは深化方向として 4 項目（文字種要件・ユーザー識別子との
類似禁止・共通パスワード辞書・パスワード履歴）を挙げていたが、現行 NIST
SP 800-63B-4 §3.1.1.2 はそのうち 2 つ（文字種要件・periodic rotation）を
明示的に **採用しない** よう推奨している。これと衝突する control を実装する
かどうかは設計判断であり、ADR が必要。

仕様核 `spec/scl.yaml` には既に「長さのみを強制し、文字種混在ルールは課さない」
という宣言があり、HIBP 等の漏洩データベース検査は別 port
（`BreachedPasswordChecker`）に切ってある。本 ADR は NIST 整合性を明文化し、
類似禁止と共通パスワード辞書のみを追加する。

## 決定

NIST SP 800-63B-4 §3.1.1.2 に整合させ、長さ (`min_length=12` / `max_length=128`)・ユーザー
識別子との類似禁止・共通パスワード辞書のみを control として採用し、同節が明示的に非推奨とする
文字種混在 (composition rule) と periodic rotation は採用しない。パスワード履歴の再利用禁止と
外部漏洩データベース検査は、それぞれ別 port (`PasswordHistoryRepository` /
`BreachedPasswordChecker`) を要する別関心事として本 ADR では保留し、後続の ADR-027 / ADR-028 で
定める。

現在の設計は [`backend/authentication/ARCHITECTURE.md`](../backend/authentication/ARCHITECTURE.md)
の Password lifecycle セクションにある。

## 影響

- `password-policy.ts` のシグネチャに optional `context?: { username?; email? }`
  が増える。既存呼び出し（seed）は context を渡すよう更新。
- 共通パスワード辞書ファイル `common-passwords.ts` を追加。中身は port では
  なく Layer 3 の定数 — `BreachedPasswordChecker` port とは目的が異なる
  （前者は offline baseline、後者は外部知識）。
- `bootstrap/seed.ts` のデフォルト demo password を `alice-password` から
  `demo-password-1234` に変更（前者は新ポリシーで `similar_to_identifier`
  違反となるため）。dev.sh が既に渡している値と一致する。

## 却下した代替案

- **大規模辞書 (rockyou / SecLists) を bundle**: 配布バイナリサイズと
  メモリ消費に対して効果が薄い。HIBP k-anonymity への port で代替するほうが
  カバレッジ・更新性とも優れる。
- **類似閾値に Levenshtein / sequence-matcher を採用**: 実装コストが増える
  割に substring containment より誤検知が減るとは限らない。識別子長下限を
  4 文字に切る単純な containment で実用上の lower-hanging fruit を取る。
- **文字種要件を opt-in flag として今同時に導入**: 「未要求の flexibility」
  に当たる。要件が現れた時に別 ADR で追加する。
