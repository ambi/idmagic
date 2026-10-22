---
status: completed
---

# Entra のドメインフェデレーションに、Entra が送る realm で応じ、ドメインの IssuerUri を発行者にする

REQ-WSFEDERATION-001、002、004 と、`ConfigureEntraFederation`、`EntraFederationProfile` を改め、`EntraDomainFederation` を足した。
Entra の定型設定は、テナントに一つの `urn:federation:MicrosoftOnline` の RP にドメインと IssuerUri の組を持ち、利用者の UPN のドメインに最も長く一致する組の IssuerUri を発行者にする。
受動的な発行と能動的な発行は同じ規則を使い、どのドメインにも属さない利用者には 500 で発行を拒否する。
ImmutableID は ImmutableID のクレームでも発行し、合成の属性は sourceAnchor の属性を公開できるときだけ公開できる。
着手時に、現在の定型設定が公開の下限でどの経路でも発行できなかったことと、同じ IssuerUri の別のドメインの構成が既存の RP を置き換えることを確かめ、両方を直した。
`issuerid` のクレームは発行しない。Microsoft の文書はトークンの発行者がドメインの IssuerUri と一致することを求めるからである。
ドメインを一つずつ外す操作は対象外とし、実テナントでの確認は wi-79 に残る。

全文は `git log -1 -- work-items/active/wi-97369-issue-entra-federation-tokens-for-the-realm-and-issuer-entra-expects.md` が示すコミットの親、またはそのコミットで読む。
