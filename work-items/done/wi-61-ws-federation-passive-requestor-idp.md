---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# WS-Federation Passive Requestor Profile の IP-STS としてブラウザ SSO に対応する

WsFederation bounded context (ADR-064) と宣言的 claim 発行 (ADR-059) に沿って、WS-Federation passive requestor
profile の IP-STS を実装した。`/{realm}/wsfed` で `wa=wsignin1.0` /
`wsignout1.0` / `wsignoutcleanup1.0` を受理する。未認証は既存ログイン UI へ
return_to つきで誘導し、認証済みなら RP の宣言的 claim policy で claim を発行し、
goxmldsig (ADR-060) で署名した SAML assertion を WS-Trust RSTR に包んで自動 POST
する。token は既定で SAML 1.1 (Entra / AD FS の WS-Fed 互換)、RP 設定で SAML 2.0
も選択可。`wfresh` を尊重して古い認証は再認証へ誘導し (ログイン往復を吸収する短い
猶予つき)、`wauth` を尊重して満たせる password 系は反映、満たせない統合 Windows
認証等は fail-closed で拒否する (無音サインインは wi-65)。relying party は
`/api/admin/wsfed/relying-parties` と管理 UI 「WS-Federation 連携先」で CRUD でき、
許可 wreply 閉集合・audience・token type・claim mapping を編集する。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
