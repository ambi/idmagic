# wi-69290-bind-workload-token-exchange-to-the-bound-client

作業項目は `wi-69290-bind-workload-token-exchange-to-the-bound-client` である。
対象は [REQ-WORKLOADIDENTITY-001](../../modules/workloadidentity/features/attestation-exchange.md) と [RFC8693-SUBJECT-TOKEN](../../modules/oauth2/standards.md) である。

## Token Exchange のアテステーション

`subject_token_type` が `urn:ietf:params:oauth:token-type:jwt` の Token Exchange は、トークンエンドポイントで認証したクライアントが、アテステーションの対応先の Agent に束縛されている場合だけ成功する。
束縛されていないクライアントの要求は 400 と `invalid_grant` で拒否し、監査には理由 `client_not_bound`、用途 `token_exchange` の `WorkloadAttestationRejected` を残す。

以前は、Token Exchange を許可されたどのクライアントでも交換でき、発行したトークンの `sub` は Agent の最初の束縛先のクライアント、`client_id` は要求したクライアントになっていた。
仲介のクライアントでワークロードの代わりに交換していた構成は、次のどちらかへ移す。

- ワークロードが、Agent に束縛したクライアントとして認証して交換する。
- `spiffe` の信頼設定の JWT-SVID なら、Agent に束縛した `spiffe_jwt` のクライアントとして JWT-SVID で認証し、`client_credentials` でトークンを得る。

Agent が複数のクライアントに束縛されている場合、発行するトークンの `sub` と `client_id` は、束縛の順序によらず要求したクライアントになる。

後退するには、このリリースより前の版へ戻す。
永続データは変わらない。
