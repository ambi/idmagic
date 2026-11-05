---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-06
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-046 }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-012 }
---

# 止めた Agent のクライアントには、どのグラントでも新しいトークンを発行しない

## 動機

[wi-10874](../done/wi-10874-inventory-promises-kept-by-every-entry-path.md) の棚卸しで、Agent の状態と所有者を確かめる `ResolveIssuableAgent` を、一部のグラントだけが呼ぶことが分かった。

| グラント | Agent の状態と所有者の確認 |
| --- | --- |
| `client_credentials` | あり（`ResolveIssuableAgentWithoutApproval`） |
| トークン交換 | あり |
| 承認とバックチャネル認可 | あり |
| 認可コード、リフレッシュトークン、デバイスコード | なし（`RequestingAgentID` は識別子だけを引く） |

[OAuth2 の token の設計](../../docs/modules/oauth2/features/token.md)は「すべてのトークン発行経路で `status` を安全側に検査」すると書くが、[REQ-OAUTH2-046](../../docs/modules/oauth2/features/token.md) が定めるのは `client_credentials` だけである。
`Disabled` か `Killed` の Agent に束縛したクライアントは、リフレッシュトークンで新しいアクセストークンを得られる。
イントロスペクションは REQ-OAUTH2-012 で `active=false` を返すが、JWT をその場で検証するリソースサーバーは受け入れる。
規範の取りこぼしであり、設計の記述に対しては違反である。

## 対象範囲

- REQ-OAUTH2-046 を、Agent に束縛したクライアントのすべてのグラントへ広げ、Agent の状態の確認を加える。
- 認可コード、リフレッシュトークン、デバイスコードでも、止めた Agent と所有者が `Active` でない Agent のクライアントへ発行しない。

## 対象外

- 人間の承認の要否（REQ-OAUTH2-050）。グラントごとの現在の判定を変えない。
- 発行済みのトークンの無効化。REQ-OAUTH2-012 が扱う。

## 設計

### 所有の形

候補 A を採る。
[REQ-OAUTH2-073](../../docs/modules/oauth2/features/guardrail.md) のガードレールと同じく、グラントごとのユースケースではなく、トークンの要求ごとに作る発行器で確認する。
`token_handler.go` の `RequestingAgentID` を、状態と所有者を確かめて拒否する操作に置き換え、グラントの分岐の前で一度だけ呼ぶ。
`client_credentials` の分岐の個別の確認は、承認の区分の判定だけを残す。
応答は既存と同じ 401 と `invalid_client` にする。

## タスク

- [ ] T001 [Spec] REQ-OAUTH2-046 の対象をすべてのグラントへ広げ、例を足す。
- [ ] T002 [Test] 無効化した Agent のクライアントがリフレッシュで発行を受けることを RED にする。
- [ ] T003 [App] 確認をグラントの分岐の前へ移す。

## 検証

- `mise run test-go-package -- ./backend/oauth2/internal/handlers_http`
- `mise run test-go-package -- ./backend/oauth2/internal/usecases`
- `mise run check-spec`

## リスク

- 公開契約の上で、リフレッシュの失敗の応答が `invalid_grant` から `invalid_client` に変わるクライアントがある。TypeSpec の宣言を確かめる。
