---
status: pending
authors: [tn]
risk: low
created_at: 2026-11-05
priority: p2
group: api-contract
depends_on: []
change_kind: docs
spec_impact: { kind: none, reason: "実装が既に受け入れている要素と、既にある受け入れの制限を標準要件と試験に加える作業であり、製品の振る舞いと既存の規範の結果は変えない。標準要件の追加で規範が変わると分かれば、着手時に affected_spec へ切り替える。" }
---

# 連携プロトコルで相手が省略できる要素の受け入れと受け入れの制限を、標準要件と試験に加える

## 動機

[[wi-77372-inventory-standards-acceptance-rules]] の棚卸しで、実装が受け入れているのに、標準要件も、省いた形を送る試験もない要素が、OAuth2、SAML、WS-Federation、外部 OIDC の四つのモジュールで見つかった。
受け入れの標準要件と試験がなければ、後の変更がこれらを必須にしても検出できない。
外部 OIDC では、受け入れの制限（RS256 だけ）が宣言にないため、ES256 の OP と接続できないことを利用者が事前に知る手段もない。
四つは同じ手順（`standards.md` に受け入れの行を足し、省いた要求を送る試験から参照する）で直すので、一つの作業にする。

## 対象範囲

| モジュール | 要素 | 現在の受け入れ | 足すもの |
| --- | --- | --- | --- |
| OAuth2 | `private_key_jwt` の `client_id` の省略（RFC 7521 §4.2） | 共通のクライアント認証が全入口で受け入れる | PAR、イントロスペクション、失効、デバイス認可、CIBA の入口で、省いた要求を送る試験 |
| OAuth2 | デバイス認可の `scope` の省略（RFC 8628 §3.1、RFC 6749 §3.3） | `openid` をデフォルトにし、`openid` を持たないクライアントは `invalid_scope` | デフォルトの範囲と失敗の受け入れの標準要件と試験 |
| OAuth2 | `client_credentials` の `scope` の省略（RFC 6749 §4.4.2） | クライアントの登録の範囲をデフォルトにする | 受け入れの標準要件と試験 |
| OAuth2 | 失効の `token_type_hint`（RFC 7009 §2.1） | ヒントを読まずにすべての種類を探す | 受け入れの標準要件と、ヒントを外した要求の試験 |
| OAuth2 | DPoP の `htu`（RFC 9449 §4.3） | 正規化せずに完全一致で比べる。SHOULD の正規化をしない | 制限を `partial` の標準要件として宣言する |
| OAuth2 | 検査の項目だけを並べた標準要件（`RFC7523-CLIENT-ASSERTION`、`RFC9449-PROOF` など） | — | 相手が省略または追加できる要素があれば、受け入れの標準要件へ分ける |
| SAML | AuthnRequest の `Destination`、`AssertionConsumerServiceURL`、`ProtocolBinding`、`NameIDPolicy` の省略（SAML 2.0 Core §3.4.1） | `ValidateSignIn` が SP のデフォルトの ACS URL と NameID 形式で受け入れる。`NameIDPolicy` の `unspecified` もデフォルトとして扱う | 受け入れの標準要件と、省いた要求を SSO の入口へ送る試験 |
| WS-Federation | `wsignin1.0` の `wreply`、`wctx`、`wfresh` の省略（WS-Federation 1.2） | `resolveReplyURL` が `wreply` を省いた要求を最初の許可済みの返信先で受け入れる | 受け入れの標準要件と、省いた要求を受動的な入口へ送る試験 |
| 外部 OIDC | 上流の ID Token の署名アルゴリズム | RS256 でだけ検証する | 制限を `partial` の標準要件として宣言する |
| 外部 OIDC | トークン応答の `token_type`、`expires_in`、`refresh_token` | 読まずに受け入れ、`id_token` だけを読む | 受け入れの標準要件と、それを参照する試験 |

既存の試験が省いた形を送っていれば、新しい試験を足さず、その試験から標準要件を参照する。

## 対象外

- 実装が拒否する要素。棚卸しが起票した bugfix の各項目が扱う。SAML の ACS のインデックスと HTTP-POST 以外の `ProtocolBinding` も、拒否を宣言済みなので扱わない。
- WS-Trust の能動的な STS。`MSAL-WSTRUST-RST` が MSAL の要求の形を宣言している。
- 外部 OIDC の RS256 以外のアルゴリズムへの対応。必要になれば feature として起票する。`kid` の省略は [[wi-41129-accept-upstream-id-tokens-without-kid-for-a-single-key]] が扱う。

## 検証

- `mise run check-spec`：加えた標準要件の形式と参照を確かめるため。
- 加えた HTTP の試験：省いた形を製品の入口が受け入れることを観測するため。

## リスク

試験で受け入れられないと分かった要素は、この項目で直さず bugfix として起票する。
