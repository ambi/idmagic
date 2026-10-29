---
status: completed
---

# トークン交換で元トークンの鍵所有を検証し、送信者制約を引き継ぐ

REQ-OAUTH2-056 を追加し、送信者制約付きの `subject_token` は同じ方式の同じ鍵の証明があるときだけ交換でき、同じ `cnf` を引き継ぐことを規範にした。
修正前は、K1 に束縛した token を K2 の Proof で交換すると K2 へ付け替わり、Proof を省くと Bearer になることを正式な `/token` で再現した。
判定は `token/domain` の `DecideExchangeBinding` 一つに置き、ユースケースはポリシーの評価の後、署名の前に呼ぶ。拒否は `invalid_grant` で、`TokenExchangeRejected` だけを発行する。
未束縛の subject と外部ワークロードの subject は従来どおり、要求の証明（両方なら DPoP）に結び付ける。
委譲先の鍵への再束縛（`may_act` を根拠にする案）と、束縛した `actor_token` の鍵の照合は対象外にした。前者は後から許可しても既存の交換を壊さないため、許可の発行元ができてから扱う。
確認した範囲は DPoP と mTLS の自社発行 token であり、`cnf` に未知のメンバーだけを持つ token は制約なしとして扱われるまま残る。
