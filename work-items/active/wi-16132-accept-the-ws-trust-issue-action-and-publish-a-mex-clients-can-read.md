---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-23
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/ws-federation/features/active-sts.md, requirement: REQ-WSFEDERATION-005 }
  - { path: spec/modules/ws-federation/main.tsp, symbol: IdMagic.WsFederation.Operations.WsTrustIssue }
  - { path: spec/modules/ws-federation/main.tsp, symbol: IdMagic.WsFederation.Operations.WsTrustMetadataExchange }
---

# WS-Trust の Issue を標準の Action で受け、クライアントが読める MEX を公開する

## 動機

能動的な STS には、標準に従うクライアントからの要求を受けられない欠陥が二つある。

| 欠陥 | 標準またはクライアントの形 | 現在の実装 |
| --- | --- | --- |
| RST の `Action` | WS-Trust 1.3 の 4.1 節は、Issue の要求の Action を `http://docs.oasis-open.org/ws-sx/ws-trust/200512/RST/Issue`、`RequestType` を `http://docs.oasis-open.org/ws-sx/ws-trust/200512/Issue` と定める | `requests_wstrust.ParseRST` が `Action` に `RequestType` の値（`.../200512/Issue`）を求める。標準の Action を送るクライアントは 400 で拒否される |
| MEX | MSAL（`MexDocument`）は、`wsp:Policy` の `wsp:ExactlyOne`／`wsp:All` にある `sp:TransportBinding` と `sp:SignedEncryptedSupportingTokens`（`UsernameToken`／`WssUsernameToken10`）で方針を選び、`wsdl:binding` の `wsp:PolicyReference`、`soap12:binding`、`soap12:operation` の `soapAction` で束縛をたどり、`wsdl:port` の `wsa10:EndpointReference` からエンドポイントを得る | `BuildMEX` は「簡易 MEX」で、`wsdl:binding`、方針の参照、`TransportBinding`、`UsernameToken` の方針、ポートの `EndpointReference` を持たない。MSAL は usernamemixed のエンドポイントを見つけられない |

テストは、実装と同じ誤った Action の値で RST を組み立てているため、この欠陥を観測していない。
MEX のテストも、要素名が出力に含まれることだけを確かめている。

## 対象範囲

- RST の `Action` に WS-Trust 1.3 の `.../RST/Issue` を求め、`RequestType` には `.../Issue` を求める。
- MEX を、WSDL の `binding`、`service`、`port` と WS-SecurityPolicy の方針を持つ形にし、usernamemixed のエンドポイントを WS-Trust 1.3 の束縛として公開する。
- MSAL が送る形の RST を固定の入力とする受け入れテストと、MSAL の選択の手順をオラクルにした MEX のテストを足す。
- REQ-WSFEDERATION-005 の Action の文言と、標準仕様の表の Action の値を明記する。

## 対象外

- WS-Trust 2005 の束縛（`/trust/2005/usernamemixed`）。Hybrid Join を含む Entra のクライアントは 1.3 か 2005 のどちらかがあれば足りる。
- `windowstransport` の公開。wi-98213 で扱う。

## 設計

MEX の組み立ては、現在と同じく `metadata_wsfederation` が要求ごとに行う。
方針と束縛の組を一つのデータで表し、wi-98213 が `windowstransport` の組を足せる形にする。

```go
type mexBinding struct {
    Name     string // 例 UserNameWSTrustBinding_IWSTrust13Async
    PolicyID string
    Policy   mexPolicy // UsernameToken over TLS、のちに Negotiate を足す
    Address  string
}
```

MEX のテストのオラクルは、MSAL の `MexDocument` の選択の手順（方針、束縛、ポートの順にたどる）を Go で書き、公開した MEX から usernamemixed のエンドポイントが得られることを確かめる。
受け入れテストの RST は、MSAL の `WsTrustEndpoint` が組み立てる要素と名前空間の形に合わせる。

## 計画

1. MSAL の RST と MEX の形を、MSAL のソースから一次資料として写し、テストの固定の入力とその出典を記録する。
2. RST の Action の修正と MEX の修正は、同じ能動的な STS の相互運用の欠陥なので一つの変更にする。

## タスク

- [ ] T001 [Spec] REQ-WSFEDERATION-005 と標準仕様の表に Action と MEX の形を明記する。
- [ ] T002 [Acceptance] MSAL の形の RST が 400 になることと、MSAL の手順で MEX からエンドポイントを得られないことを RED で確かめる。
- [ ] T003 [Adapter] `ParseRST` と `BuildMEX` を直し、受け入れテストと既存のファズテストを GREEN にする。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed`：`requests_wstrust`、`metadata_wsfederation`、WsFederation の HTTP の例。
- `mise run check`、`mise run check-spec-impact`。
- ミューテーションテスト：`ParseRST` の Action と RequestType の照合。

## リスク

- MEX の形は MSAL のソースに合わせるので、ほかのクライアント（Office の古い版、dsregcmd）の差は実テナントの確認（wi-79）まで残る。
