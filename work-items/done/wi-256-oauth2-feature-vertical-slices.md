---
status: completed
archived_from: 807f8be3571cfedcc7b1cee5025d26606f55350a
---

# oauth2 を client/consent/authorization/token/device の feature 垂直スライスへ再配置する

OAuth2 の feature 固有 domain / port / usecase / persistence adapter を client / consent /
authorization / token / device の垂直スライスへ再配置した。feature 横断 helper と公開 import
互換 facade は root に維持し、HTTP adapter は ADR-131 に従って feature usecase を dispatch する
context root shared protocol adapter とした。`ARCHITECTURE.md` と traceability manifest の
module / test path を新配置へ同期した。

全文は `git show 807f8be3571cfedcc7b1cee5025d26606f55350a:<このファイルのパス>` で読む。
