---
status: completed
---

# ワークロードのアテステーションの交換で、要求したクライアントと束縛先のクライアントの関係を決める

`spec-diff` は REQ-WORKLOADIDENTITY-001 と RFC8693-SUBJECT-TOKEN の変更を示す。
JWT のアテステーションによる Token Exchange は、トークンエンドポイントで認証したクライアントが対応先の Agent に束縛されている場合だけ成功し、そうでなければ理由 `client_not_bound`、用途 `token_exchange` の `WorkloadAttestationRejected` と 400 の `invalid_grant` で拒否する。
発行するトークンの `sub` と `client_id` は要求したクライアントになり、Agent の最初の束縛を選ぶ規則は取り除いた。
例 EX-WORKLOADIDENTITY-001-02 を足し、移行の案内を `docs/releases/upgrades/` に下書きした。

案 A を採った。
案 B は AIMS00-AGENT-CLIENT-AUTH と矛盾し、案 C は仲介者の登録がなければ案 A と同じなので、需要が出たら互換を保って足せる。
経路を廃止する案は、`oidc` の信頼設定のアテステーションがこの経路でしか使えないので退けた。
照合は REQ-WORKLOADIDENTITY-017 と同じく WorkloadIdentity が行い、OAuth2 のポートは認証済みのクライアントの ID を渡す。

確かめた範囲は、検証のユースケースの単体テスト（束縛の順序と拒否のイベント）と、管理 API で登録して `/token` へ署名付きで要求する HTTP 統合テストである。
実際の SPIRE が発行する JWT-SVID での確認は wi-76418 が扱う。
要求したクライアントの認証の強さは共通入口に依存し、この変更はそれを強めない。
