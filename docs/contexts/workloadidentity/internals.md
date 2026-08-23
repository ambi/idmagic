# WorkloadIdentity Internals

## Attestation is verified, never trusted

この Context は SPIRE のサーバーやエージェントを動かさず、外部が発行したアテステーションを検証する側に立つ。OAuth2 の Token Exchange から渡された外部アテステーショントークンの署名、`iss`、`aud`、`exp`、TTL 上限を登録済みの `WorkloadTrustBundle` で検証する。次に、テナント内の `AgentWorkloadBinding` から対応する `Agent` を一意に特定し、その `Agent` が `Active` であることを確認する。いずれかの検証に失敗した場合はフェイルクローズで拒否する。

