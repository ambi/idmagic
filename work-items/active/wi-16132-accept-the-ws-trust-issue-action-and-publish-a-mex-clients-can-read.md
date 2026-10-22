---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-23
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/ws-federation/features/active-sts.md, requirement: REQ-WSFEDERATION-005 }
  - { path: spec/modules/ws-federation/main.tsp, symbol: IdMagic.WsFederation.Operations.WsTrustIssue, impact: conforms }
  - { path: spec/modules/ws-federation/main.tsp, symbol: IdMagic.WsFederation.Operations.WsTrustMetadataExchange }
primary_use_cases:
  - id: msal-rst-issue
    requirement: REQ-WSFEDERATION-005
    observable_result: MSAL が組み立てる形の WS-Trust 1.3 の RST（Action が `.../RST/Issue`）に、署名した Assertion を持つ RSTR が返り、`RequestType` の値を Action に置いた RST は 400 で拒否される。
    boundary: acceptance
    test: { path: backend/wsfederation/internal/handlers_http/wstrust_client_interop_test.go, name: TestWsTrustIssue_AcceptsTheRSTThatMSALSends, task: test-go-race }
    fault_model: Action の照合が RequestType の値を求め、標準の Action を送るクライアントを拒否する。
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
// mexBinding は、MEX の WSDL で一つのエンドポイントを公開する方針、束縛、ポートの組である。
type mexBinding struct {
    name       string             // UserNameWSTrustBinding_IWSTrust13Async
    address    string             // https の usernamemixed
    soapAction string             // http://docs.oasis-open.org/ws-sx/ws-trust/200512/RST/Issue
    assertion  func(*etree.Element) // wsp:All に認証の方針を書く。のちに Negotiate を足す
}
```

### 着手時に一次資料から写した形

出典は MSAL.NET の `src/client/Microsoft.Identity.Client/WsTrust/MexDocument.cs`、`WsTrustEndpoint.cs`、`Internal/XmlNamespace.cs`（main ブランチ）である。

| 項目 | 形 |
| --- | --- |
| RST | SOAP 1.2。`wsa:Action`（`mustUnderstand="1"`）は `.../200512/RST/Issue`、`wsa:MessageID`、`wsa:ReplyTo`（anonymous）、`wsa:To`、`wsse:Security` の `wsu:Timestamp`（小数秒付き、期限は作成の 10 分後）と `UsernameToken`。本体の `wst:RequestSecurityToken` は `wsp:AppliesTo`、`KeyType`（Bearer）、`RequestType`（`.../200512/Issue`）の順 |
| MEX の方針 | `wsp:Policy`（`wsu:Id`）の `wsp:ExactlyOne`／`wsp:All` に、`TransportBinding`（WS-SecurityPolicy 1.2 または 2005/07 の名前空間）と、`sp:SignedEncryptedSupportingTokens`（1.2）または `sp:SignedSupportingTokens`（2005/07）を置く。その下の `UsernameToken` と `WssUsernameToken10` は、MSAL は 2005/07 の名前空間でだけ探す |
| MEX の束縛 | `wsdl:binding`（`name`）の `wsp:PolicyReference`（`URI="#<wsu:Id>"`）、`soap12:binding`（`transport="http://schemas.xmlsoap.org/soap/http"`）、`wsdl:operation` の `soap12:operation`（`soapAction` が `.../RST/Issue` なら WS-Trust 1.3） |
| MEX のポート | 最初の `wsdl:service` の `wsdl:port`（`binding="<接頭辞>:<束縛名>"`）の `wsa10:EndpointReference`／`wsa10:Address`。https の絶対 URL だけを使う |

方針は、MSAL の解析の癖に合わせ、WS-SecurityPolicy 2005/07 の `sp:SignedSupportingTokens` で揃える。
1.2 の `SignedEncryptedSupportingTokens` の下に 2005/07 の `UsernameToken` を置く形は、名前空間が混ざり、ほかのクライアントが読み違えるおそれがあるので採らない。

MEX のテストのオラクルは、MSAL の `MexDocument` の選択の手順（方針、束縛、ポートの順にたどる）を Go で書き、公開した MEX から usernamemixed のエンドポイントが得られることを確かめる。
受け入れテストの RST は、上の表の形に合わせる。
MEX の公開は既存の要件を持たないので、README の「フェデレーションメタデータ」と TypeSpec の説明に形を書き、テストは `//verify:operation WsTrustMetadataExchange` で引く。

## 計画

1. MSAL の RST と MEX の形を、MSAL のソースから一次資料として写し、テストの固定の入力とその出典を記録する。
2. RST の Action の修正と MEX の修正は、同じ能動的な STS の相互運用の欠陥なので一つの変更にする。

## タスク

- [x] T001 [Spec] REQ-WSFEDERATION-005 と標準仕様の表（`WSAddressing-MessageIDToAction`）に Action の URI を明記し、README の「フェデレーションメタデータ」と `WsTrustMetadataExchange` の説明に MEX の方針、束縛、ポートの形を書いた。
- [x] T002 [Acceptance] `wstrust_client_interop_test.go` の `TestWsTrustIssue_AcceptsTheRSTThatMSALSends` は 400 `wstrust: unsupported Action` で、`TestWsTrustMEX_LetsMSALFindTheUsernameMixedEndpoint` は MSAL の手順でエンドポイントを一つも選べずに失敗した。
- [x] T003 [Adapter] `ParseRST` の Action の照合を `ActionIssue`（`.../RST/Issue`）にし、`BuildMEX` を `mexBinding` の組から方針、束縛、ポートを書く形にした。既存のテストの RST とファズテストの種の Action を標準の値に改め、束縛名を AD FS と同じ `UserNameWSTrustBinding_IWSTrust13Async` にした。
- [x] T004 [Verify] 変更を検証した。結果は検証の節に書いた。

## 検証

- `mise run test-go-changed -- main`：677 件が成功、失敗とスキップは 0 件。
- `mise run check`、`lint-go`、`check-spec-impact`、`check-operation-references`、`check-boundaries`：成功。
- ミューテーションテスト（変更行、`main` と比較）：`requests_wstrust` は 1 件を検出した。`metadata_wsfederation` の変更行には変異の対象がない。
- 手動の故障注入：MEX から方針の参照、WS-Trust 1.3 の `soapAction`、ポートのエンドポイント参照、`TransportBinding`、`WssUsernameToken10` をそれぞれ落とし、`TestWsTrustMEX_LetsMSALFindTheUsernameMixedEndpoint` が五つとも検出した。

## リスク

- MEX の形は MSAL のソースに合わせるので、ほかのクライアント（Office の古い版、dsregcmd）の差は実テナントの確認（wi-79）まで残る。

## 要約

REQ-WSFEDERATION-005 と `WsTrustMetadataExchange` を改めた。
能動的な STS は、RST の `Action` に WS-Trust 1.3 の Issue の要求（`.../200512/RST/Issue`）を求め、`RequestType` の値（`.../200512/Issue`）を Action に置いた RST を拒否する。
MEX は、TLS と UsernameToken を求める方針、それを参照する WS-Trust 1.3 の Issue の束縛、束縛を参照するポートを WSDL で公開し、MSAL が方針、束縛、ポートの順にたどって usernamemixed を選べる。
方針は、MSAL が `UsernameToken` を WS-SecurityPolicy 2005/07 の名前空間でだけ探すので、その名前空間で揃えた。
RST と MEX の形は MSAL.NET のソースを一次資料にし、受け入れテストはその手順を Go で写したオラクルで確かめる。
WS-Trust 2005 の束縛は対象外とし、MSAL 以外のクライアントとの差は実テナントの確認（wi-79）に残る。
