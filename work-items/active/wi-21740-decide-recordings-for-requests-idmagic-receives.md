---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-28
priority: p2
group: development-process
depends_on: []
change_kind: tooling
spec_impact: { kind: none, reason: "IdMagic が受け取る要求と文書について、実際の相手の記録を入力にした受け入れの検査を置くかを相手ごとに決め、採った相手に検査を足す。規範シナリオ、標準の宣言、TypeSpec シンボルは変えない。検査が規範に反する実装を見つけたら、別の項目に起票するか、その時点で affected_spec を宣言し直す。" }
---

# IdMagic が受け取る要求と文書について、実際の相手の記録で検証するかを相手ごとに決める

## 動機

外部の相手の記録の取り組み（wi-65625、wi-27706、wi-49024、wi-21837、wi-94480、wi-20912）は、IdMagic が要求を送る相手の応答を、一次資料の例と実物の記録で置き換えた。
手書きの応答を記録に変えただけで、SCIM、OAuth、OIDC、MCP、Vault、SET の相手に対する欠陥が見つかった。
手書きの入力は、実装を書いた人の思い込みを実装と共有するためである。

一方、IdMagic が要求を受け取る側は洗い出していない。
[テスト方針の外部契約と設定の利用](../../docs/development/testing.md#外部契約と設定の利用)は、受け取る側にも実際のクライアントの要求を入力にした受け入れの検査を求めるが、新しく実装する作業と版を上げる作業にだけ適用している。
既存の実装の受け入れの検査の入力は、多くが実装を書いた人の組み立てた要求である。
2026-11-28 の調査では、そのような受け取る側の二箇所（上流の SAML IdP の応答、SAML の HTTP-Redirect の署名）で、一次資料と食い違う実装が見つかった（[[wi-53342-accept-upstream-saml-assertions-signed-without-a-response-signature]]、[[wi-75603-verify-saml-redirect-signatures-over-the-received-encoding]]）。

調査の時点で分かっている相手は次のとおりである。

| 受け取るもの | 受け取る側 | 今の検査の入力 | 記録の出どころの候補 |
| --- | --- | --- | --- |
| SCIM の要求（Entra ID、Okta のプロビジョニング） | `sourcing/internal/scim` | 手書きの要求。PATCH の `op` は小文字、値は真偽値 | Microsoft Learn「Tutorial: Develop and plan provisioning for a SCIM endpoint」の要求の例、Okta の SCIM の文書の例 |
| WS-Federation のサインインと WS-Trust の RST | `wsfederation` | 手書きの要求 | Microsoft のプロトコル文書（MS-WSFED、MS-WSTR 系）の例、Windows のクライアント |
| MCP ゲートウェイが受ける MCP の要求 | `mcpgateway`、`shared/http/server_http` の E2E | `modelcontextprotocol/go-sdk` のクライアント | Claude Code、VS Code などの実際のクライアントを手元の受け口へつないだ記録 |
| WebAuthn の登録と認証の応答 | `authentication/internal/webauthn` | テストの中のソフトウェア認証器（attestation は `none`） | ブラウザーの仮想認証器、WebAuthn Level 3 と FIDO の例 |
| OAuth と OIDC の OP への要求 | `oauth2` | 手書きの要求 | 実在の RP のライブラリ。[[wi-33-protocol-conformance-ci]] の適合試験との分担を決める |
| SP のメタデータと上流の IdP のメタデータの取り込み | `saml`、`authentication` の連携 | 手書きの XML | 公開されている SaaS の SP メタデータ、Entra ID の公開の連携メタデータ |
| SSF のストリーム管理の要求（受信者から） | `sharedsignals/internal/handlers_http` | 手書きの要求 | SSF 1.0 の例。[[wi-36958-conform-to-shared-signals-framework-1-0]] との分担を決める |

Microsoft Learn の SCIM の文書（2026-11-28 に本文を読んだ、ms.date 2026-09-15）では、Entra ID は利用者のメールの更新を `"op": "Replace"`、`"path": "emails[type eq \"work\"].value"` で送る。
Sourcing はこの path を受け付けない（[[wi-248-scim-complex-value-filter-bracket-syntax]]）。
記録の再生は、このような未対応の形を、実際の相手の操作として見えるようにする。

## 対象範囲

- 上の相手と、洗い出しで見つかる受け取る側の相手ごとに、既存のテストを振る舞いごとに並べ、記録の再生を置くかを決める。判断の基準は wi-94480 と同じく、記録の形が結果を左右し、既存のテストが扱えない分岐があるかである。
- 採った相手に、実物または一次資料の例から取った要求の記録を置き、受け入れの検査で再生する。期待値は一次資料の文面から書く。
- 検査が見つけた食い違いは、規範への違反なら別の項目に起票し、未対応の任意の機能なら既存の項目の優先度を見直す。
- 採らなかった相手は、理由と見直す契機を要約に書く。

## 対象外

- IdMagic が出力し、外部が解析するもの（Assertion、メタデータ、トークン）の検証。[[wi-13863-verify-idmagic-outputs-with-independent-real-consumers]] が扱う。
- 上流の SAML の応答と SAML の Redirect の署名。上の二つの項目が扱う。
- Kerberos と LDAP。Samba AD と OpenLDAP の実物で既に試験している。
- Entra ID の実テナントでの確認。[[wi-79-entra-real-tenant-end-to-end-verification]] が扱う。

## 設計

記録の形式と再生は `testing_httprecord` を使い、送信側と同じ部品で受信側の再生も書けるかを着手時に確かめる。
送信側の再生はクライアントへ記録の応答を返すが、受信側の再生は記録の要求を IdMagic の HTTP の入口へ送り、応答と作用を一次資料から書いた期待値と比べる。
部品に足すのは、記録の要求を `http.Handler` へ送る関数程度で済む見込みである。

記録の値のうち、相手の識別子、時刻、署名に縛られる値は、テストの固定値へ置き換えると記録の意味を失う。
置き換えが要る値は、署名のない要求に限り、置き換えた欄を記録の取得方法に書く。

## 計画

1. 受け取る側の HTTP の入口と文書の取り込みを洗い出し、上の表を確定する。
2. 相手ごとに採否を決める。
3. 採った相手に記録と再生を置き、重複する手書きの検査を整理する。

## タスク

- [ ] T001 [Inventory] 受け取る側の入口と、今の検査の入力の出どころを洗い出す。
- [ ] T002 [Decision] 相手ごとに記録の採否と出どころを決める。
- [ ] T003 [Recording] 採った相手の記録と再生を置く。
- [ ] T004 [Follow-up] 食い違いを起票するか、既存の項目の優先度を見直す。
- [ ] T005 [Verify] 差分から選んだ検査を実行する。

## 検証

- 記録の要求を一次資料に反する形へ変えると、受け入れの検査が落ちる。
- `mise run test-go`、`mise run lint-go`。

## リスク

実際のクライアントの記録は、その版のクライアントの振る舞いしか表さない。
取得日とクライアントの版を記録に添え、相手の版が変わったときに取り直す。
