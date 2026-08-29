# WorkloadIdentity Glossary

| Term | Definition | Aliases |
|---|---|---|
| WorkloadTrustBundle | テナントが登録する外部アテステーション発行者の信頼設定。トラストドメイン、発行者、JWKS の取得元またはインライン JWKS、受理する audience、外部 SVID の最大 TTL をまとめる。事前に登録した発行者だけを信頼し、Trust On First Use は許可しない。 |  |
| AgentWorkloadBinding | `WorkloadTrustBundle` の配下で、外部主体に対する glob パターンを同じテナントの既存 `Agent` に対応付けるレコード。パターンに一致しない主体や、複数の有効な関連付けに一致するため対象を一意に決められない主体は、フェイルクローズで拒否する。 |  |
| JwtSvid | OIDC 互換 JWT を通信形式とする外部アテステーショントークン。Kubernetes の投影 ServiceAccount トークン、クラウドのインスタンスアイデンティティトークン、SPIFFE JWT-SVID を包含する、主要な `subject_token` 種別。X.509-SVID（mTLS）は将来の拡張とする。 | JWT-SVID |
| TrustDomain | `WorkloadTrustBundle` がまとめる論理グループ名。SPIFFE トラストドメイン、または運用者が定める発行者グループのラベル。 |  |
| FailClosed | 未登録の発行者、署名不正、期限切れ、パターン不一致、一意に決められない一致、対応先の `Agent` が `Active` でない場合のいずれでも、交換を拒否する方針。判定できない場合も「交換しない」側に倒す。 |  |
| System | `WorkloadIdentity` の検証ユースケースそのものを指す、人間の操作者を伴わない技術的な主体。 |  |

この Context の Aggregate root は `WorkloadTrustBundle` と `AgentWorkloadBinding` である。関連付けは信頼束の配下にあるが独立して有効・無効を切り替えるため、信頼束の内側には置かない。
