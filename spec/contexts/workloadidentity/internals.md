# WorkloadIdentity Internals

## VerifyWorkloadAttestation
OAuth2 Token Exchange グラント（`subject_token_type` は JWT-SVID URN）から呼び出す公開インターフェースである。外部アテステーショントークンの署名、`iss`、`aud`、`exp`、TTL 上限を登録済みの `WorkloadTrustBundle` で検証する。次に、テナント内の `AgentWorkloadBinding` から対応する `Agent` を一意に特定し、その `Agent` が `Active` であることを確認する。いずれかの検証に失敗した場合はフェイルクローズで拒否する。
- Input invariant: trust_bundle_registered_for_issuer(context.tenant_id, input.attestation.subject_token)
- Input invariant: workload_attestation_signature_and_claims_valid(context.tenant_id, input.attestation.subject_token)
- Input invariant: !ambiguous_binding_match(context.tenant_id, input.attestation.subject_token)
- Input invariant: bound_agent_is_active(context.tenant_id, input.attestation.subject_token)
- Result invariant: output.grant == null || output.grant.client_id != ''

