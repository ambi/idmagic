---
status: completed
---

# Entra と Microsoft のクライアントとの相互運用の要件を、出典付きで WsFederation の標準仕様にまとめる

WsFederation の標準仕様に、Microsoft Entra の IdP の要件、複数のドメインのフェデレーション、Entra ID の信頼のクレームの規則、sourceAnchor、Microsoft Graph の internalDomainFederation、MSAL.NET の WS-Trust の六つの節と 13 の行を、出典と参照日を付けて足した。
どの行も、実装済みの振る舞いを観測する既存の受け入れテストから引く。
未実装の要件（MFA のクレーム、端末のクレーム、Hybrid Join、証明書の自動の更新）は、担当の作業項目が実装するときに行を足す。
既存の WS-Federation、WS-Trust、WS-Security、WS-Addressing の行の ID を、作業項目の `affected_spec` から引ける英大文字の形に改めた。

全文は `git log -1 -- work-items/active/wi-96917-record-entra-and-ms-client-interop-requirements-in-ws-federation-standards.md` が示すコミットの親、またはそのコミットで読む。
