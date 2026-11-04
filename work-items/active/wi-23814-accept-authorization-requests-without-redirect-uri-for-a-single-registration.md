---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-05
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/oauth2/features/authorization.md, requirement: REQ-OAUTH2-005 }
  - { path: docs/modules/oauth2/standards.md, requirement: RFC9700-REDIRECT-MATCH }
---

# 登録が一件のクライアントの、`redirect_uri` を省いた認可要求の扱いを決める

## 動機

RFC 6749 §3.1.2.3 は、登録した `redirect_uri` が一件だけなら、認可要求で省いてよいとする。
OpenID Connect Core §3.1.2.1 の認証要求では必須である。
`/authorize` のスキーマは `redirect_uri` を常に必須とし、OpenID Connect でない要求で省いても `invalid_request` を返す。
宣言にも規範の例にも、この制限は現れない。
[[wi-77372-inventory-standards-acceptance-rules]] の棚卸しで見つけた。

## 対象範囲

- `openid` を含まない認可要求で、登録が一件のクライアントが `redirect_uri` を省いたときの扱いを決める。候補は、登録値を使って受け入れるか、制限を `partial` の標準要件として宣言するかである。
- 受け入れる場合、トークン要求の `redirect_uri` も RFC 6749 §4.1.3 に合わせ、認可要求で省いたときは省いた交換を受け入れる。
- 選んだ扱いを `standards.md` の標準要件にし、省いた要求を `/authorize` へ送る試験を加える。

## 対象外

- `openid` を含む要求。OpenID Connect Core が必須としている。
- `redirect_uri` の照合規則。ループバックのポートは [[wi-26179-allow-any-port-on-loopback-redirect-uris]] が扱う。

## タスク

- [ ] T001 [Spec] 扱いを決め、標準要件と REQ-OAUTH2-005 の例を更新する。
- [ ] T002 [Acceptance] 省いた要求で RED を確かめる。
- [ ] T003 [App] 認可要求とトークン要求の検証を合わせ、GREEN にする。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `/authorize` から `/token` までの受け入れ試験：認可コードの発行と交換の両方を観測するため。

## リスク

認可要求とトークン要求で省略の扱いがずれると、認可コードの交換が失敗するか、`redirect_uri` の束縛を検査しない交換が生まれる。
