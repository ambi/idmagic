# wi-21671-react-to-revocation-events-emitted-by-the-worker

作業項目は `wi-21671-react-to-revocation-events-emitted-by-the-worker` である。
対象は [REQ-PLATFORM-001](../../requirements/scenarios.feature.md) と [REQ-SHAREDSIGNALS-007](../../modules/sharedsignals/revocation/acceptance.feature.md) である。

## `idmagic-worker` の `ISSUER`

`idmagic-worker` は、ライフサイクルワークフローで止めた User の Agent の失効を、SET に署名して外部の受信側へ送る。
SET の `iss` には `ISSUER` を使う。

`idmagic-worker` にも、`idmagic` と同じ `ISSUER` を設定する。
設定しないとデフォルト値の `http://localhost:8080` で署名され、受信側は発行者が一致しない SET として拒否する。
Kubernetes の構成で `ISSUER` を `idmagic-runtime` の ConfigMap に置いている場合は、`idmagic-worker` にも同じ値が届くので操作はいらない。
デフォルト値と値の検証（絶対 URL であること）は変わらない。
