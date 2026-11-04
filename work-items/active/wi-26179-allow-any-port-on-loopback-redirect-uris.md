---
status: pending
authors: [tn]
risk: high
created_at: 2026-11-05
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/oauth2/standards.md, requirement: RFC9700-REDIRECT-MATCH }
  - { path: docs/modules/oauth2/standards.md, requirement: CIMD02-REDIRECT-VALIDATE }
---

# ループバックの IP の `redirect_uri` で、要求時の任意のポートを許す

## 動機

RFC 8252 §7.3 は、ループバックの IP の `redirect_uri` について、要求時に任意のポートを指定できるようにすることを認可サーバーへ MUST とする。
ネイティブアプリと CLI の MCP クライアントは、OS から得た一時的なポートで待ち受けるためである。
RFC 9700 §4.1.3 も、完全一致の例外としてこの扱いを認める。
`RedirectURIAllowed` はバイト単位の完全一致だけを許し、登録した `http://127.0.0.1/callback` と異なるポートの要求を拒否する。
Client ID Metadata Document の `redirect_uris` も同じ照合を通る。
RFC 8252 は `standards.md` に宣言されていない。
[[wi-77372-inventory-standards-acceptance-rules]] の棚卸しで見つけた。

## 対象範囲

- `standards.md` に RFC 8252 の節を加え、ループバックの IP のポートの扱いを宣言する。
- `http` スキームで、ホストが `127.0.0.1` か `[::1]` の登録値に限り、ポートだけが異なる `redirect_uri` を許す。スキーム、ホスト、パス、クエリは完全一致のままとする。
- 登録時と CIMD の文書の検証でも同じ規則を使う。
- ポートの異なる要求を `/authorize` へ送る試験と、ホストやパスの異なる要求を拒否する試験を加える。

## 対象外

- `localhost` の名前。RFC 8252 §8.3 は `localhost` を推奨せず、名前の解決に依存するため、許すかどうかを T001 で決め、許さないなら `excluded` として宣言する。
- private-use の URI スキーム。照合は完全一致のままとする。

## タスク

- [ ] T001 [Spec] RFC 8252 の節と、`RFC9700-REDIRECT-MATCH` の例外の記述を加える。
- [ ] T002 [Acceptance] ポートの異なる要求で RED を確かめ、ホストとパスの違いは拒否されたままであることを固定する。
- [ ] T003 [Domain] 照合の関数に例外を加え、GREEN にする。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `RedirectURIAllowed` の単体試験とファジング：例外が接頭辞の一致やホストの取り違えを生まないことを確かめるため。
- `/authorize` の受け入れ試験：認可コードがポートの異なる宛先へ届くことを観測するため。

## リスク

照合を緩めるため、例外の判定を誤ると認可コードを別の宛先へ渡す。
例外はループバックの IP の字句に限り、URL を解析した結果と元の文字列の両方で確かめる。
