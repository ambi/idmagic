---
depends_on: [wi-61-ws-federation-passive-requestor-idp, wi-62-ws-trust-active-sts, wi-63-federation-metadata-and-claims-mapping, wi-64-entra-domain-federation-m365-sso, wi-65-kerberos-spnego-inbound-silent-sso]
status: pending
authors: ["tn"]
risk: high
created_at: 2026-06-28
---

# Microsoft Entra ID 実テナントに対する WS-* / domain federation の end-to-end 検証

## Motivation
WS-Federation passive ([[wi-61-ws-federation-passive-requestor-idp]])、WS-Trust active STS
([[wi-62-ws-trust-active-sts]])、federation metadata / claim mapping
([[wi-63-federation-metadata-and-claims-mapping]])、Entra domain federation preset
([[wi-64-entra-domain-federation-m365-sso]])、Kerberos/SPNEGO 無音 SSO
([[wi-65-kerberos-spnego-inbound-silent-sso]]) は、いずれもローカル検証 (unit / integration /
curl) では完結しているが、Microsoft Entra ID / Microsoft 365 の実テナントに対する
end-to-end 接続検証は各 WI のローカルスライス外に残っている。

実テナント検証は検証用テナントの domain federation 切替 (`Update-MgDomainFederationConfiguration`)
という破壊的変更を伴い、claim 形状・issuer・署名要件の僅かな不一致でも無言のサインイン失敗や
sourceAnchor 不一致によるアカウント重複を招く。これは個別 WI のローカル完了とは性質が異なる
運用検証であり、複数 WI に分散させると重複し追跡しづらい。本 WI で Entra ID 周りの実テナント
検証を一箇所に集約し、各 WI からは「実テナント検証は本 WI が担う」として範囲外にする。

## Scope
- **verification**: 検証用 Entra テナントの検証済みドメインを idmagic に federation 設定 (managed → federated 切替) し、ブラウザサインイン (passive / wi-61) が成功し、発行 token に UPN / ImmutableID が 含まれることを確認する (wi-64)。, rich client / legacy active 認証経路 (WS-Trust usernamemixed / MEX、wi-62) で token が 発行され Microsoft 365 にサインインできることを確認する。, federation metadata (wi-63) を Entra が取り込み、issuer / passive / active endpoint / 署名証明書が一致することを確認する。, ドメイン参加 PC から Microsoft 365 への無音サインイン (wi-65) が成立することを確認する。, Hybrid Azure AD Join のデバイス登録が未提供である旨が設定時に診断・案内されることを実テナントで確認する (wi-64、ADR-065)。, 複数の検証済みドメインを 1 テナントで federation した場合に、ドメインごとに正しい profile / issuerUri へ解決されることを確認する。
- **documentation**: 実テナント検証の手順・結果・既知の落とし穴 (claim 形状不一致・sourceAnchor 不一致) を運用ドキュメントに残す。

## Out of Scope
- WS-Fed passive / WS-Trust active / metadata / Entra preset / 無音 SSO の実装 (wi-61〜65)。本 WI は実テナント検証のみ。
- Hybrid Azure AD Join のデバイス登録対応。Okta 同様の既知制約として未提供 (ADR-065)。
- Entra Connect (オンプレ同期) の同梱。sourceAnchor の供給はオンプレ側責務とする。

## Plan
- depends_on の wi-61〜65 が完了してから開始する純粋な外部検証 WI とし、製品コード変更を Tasks に含めない。未完の wi-65 が完了するまでは passive/active/metadata/domain profile だけを先行 rehearsal できるが、本 WI は完了にしない。
- 検証用 Entra tenant、verified test domain、test users、sourceAnchor、federation certificate を専用化し、Global Administrator 操作と domain federation 切替は時間窓・承認者・rollback command を記録する。
- Microsoft Graph/Entra の現在の federation 設定を開始前に export し、idmagic metadata/profile の issuer/passive/active/MEX/certificate と機械的に比較する。credential・domain・実 user PII は evidence 本文に埋めない。
- シナリオを passive browser、WS-Trust usernamemixed/MEX、metadata refresh/cert、複数domain routing、domain-joined SPNEGO に分け、各々 correlation ID、idmagic audit、Entra sign-in log、token claim 要約を証跡にする。
- 失敗は environment/configuration/product defect に triage し、製品修正は SCL-first の別 WI に切り出す。最後に元の managed/federated 設定へ必ず rollback して確認する。

## Tasks
- [ ] T001 [Gate] wi-61〜65 の completion/verification を確認し、未完機能を試験結果の代替で済ませない。
- [ ] T002 [Safety] Entra tenant/domain/user/sourceAnchor/certificate、管理権限、実施窓、連絡先、現在設定 export と rollback 手順を peer review する。
- [ ] T003 [Metadata] idmagic metadata/MEX と Entra 登録値の issuer、URL、token type、certificate を比較し、refresh/rotation を確認する。
- [ ] T004 [Passive] browser sign-in と wrong realm/reply/sourceAnchor negative case を実行し、UPN/ImmutableID/persistent NameID と両側 log を採取する。
- [ ] T005 [Active] WS-Trust usernamemixed/MEX から M365 token acquisition までを実行し、invalid To/Action/credential の fail-closed を確認する。
- [ ] T006 [Routing] 複数 verified domain が各 profile/issuerUriへ解決され、別 tenant/domain に混線しないことを確認する。
- [ ] T007 [Silent SSO] domain-joined test machine から SPNEGO→passive flow を実行し、fallback と未提供 Hybrid Join 診断も確認する。
- [ ] T008 [Evidence/Restore] version、時刻、主体、手順、要約値、artifact hash を記録し、設定を rollback して managed/federated 状態とサインインを再確認する。
- [ ] T009 [Triage] product defect は該当 SCL節・再現証跡を付けた別 WI として起票し、本 WI の手順書に既知の落とし穴を反映する。

## Verification
- 手動 (実テナント): test Entra テナントの検証済みドメインを idmagic に federation 設定し、 ブラウザサインイン (passive) が成功し、token に UPN / ImmutableID が含まれることを確認する。
- 手動 (実テナント): rich client / active 認証 (WS-Trust) で Microsoft 365 にサインインできることを確認する。
- 手動 (実テナント): ドメイン参加 PC から Microsoft 365 への無音サインインが成立することを確認する。
- 手動 (実テナント): Hybrid Join 設定を試みると未提供である旨が診断・案内されることを確認する。

## Risk Notes
実テナント検証は domain federation 切替という破壊的変更を伴うため、本番ではなく検証用テナントで
行う。federation を誤設定するとテナント配下ユーザーのサインインを止めうるので、切替前に managed へ
戻す手順を用意する。Entra は claim 形状・issuer・署名要件に厳格で、誤ると無言のサインイン失敗や
sourceAnchor 不一致によるアカウント重複を招くため、検証は段階的に行い各段で token を確認する。
