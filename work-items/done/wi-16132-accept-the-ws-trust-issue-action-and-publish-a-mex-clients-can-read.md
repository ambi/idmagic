---
status: completed
---

# WS-Trust の Issue を標準の Action で受け、クライアントが読める MEX を公開する

REQ-WSFEDERATION-005 と `WsTrustMetadataExchange` を改めた。
能動的な STS は、RST の `Action` に WS-Trust 1.3 の Issue の要求（`.../200512/RST/Issue`）を求め、`RequestType` の値（`.../200512/Issue`）を Action に置いた RST を拒否する。
MEX は、TLS と UsernameToken を求める方針、それを参照する WS-Trust 1.3 の Issue の束縛、束縛を参照するポートを WSDL で公開し、MSAL が方針、束縛、ポートの順にたどって usernamemixed を選べる。
方針は、MSAL が `UsernameToken` を WS-SecurityPolicy 2005/07 の名前空間でだけ探すので、その名前空間で揃えた。
RST と MEX の形は MSAL.NET のソースを一次資料にし、受け入れテストはその手順を Go で写したオラクルで確かめる。
WS-Trust 2005 の束縛は対象外とし、MSAL 以外のクライアントとの差は実テナントの確認（wi-79）に残る。

全文は `git log -1 -- work-items/active/wi-16132-accept-the-ws-trust-issue-action-and-publish-a-mex-clients-can-read.md` が示すコミットの親、またはそのコミットで読む。
