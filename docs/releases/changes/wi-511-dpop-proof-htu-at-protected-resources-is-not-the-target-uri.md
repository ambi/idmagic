# wi-511-dpop-proof-htu-at-protected-resources-is-not-the-target-uri

DPoP で送信者制約した API アクセストークンを、RFC 9449 に従うクライアントが保護リソースで使えるようになった（[`RFC9449-API-TOKEN-DPOP`](../../domain/api-tokens/standards.md)）。

- 保護リソースは、DPoP 証明の `htu` をリクエストの絶対 URL と照合する。これまではパスだけ（例: `/realms/default/api/admin/v1/users`）を期待していたので、絶対 URL を送る適合クライアントは必ず拒否されていた。
- パスだけの `htu` を送るクライアントは、保護リソースで拒否されるようになる。
- `htu` の期待値は、トークンエンドポイント、`/userinfo`、保護リソースのどれでも、テナントの正規ロケーションの origin にリクエストのパスを継いだ URL になる。subdomain style のテナントでは、トークンエンドポイントと `/userinfo` も `https://{realm}.{base}/...` を期待するようになった。これまでは基底の issuer の host を期待していた。
- `htu` を持たない DPoP 証明は、どのエンドポイントでも拒否する。
