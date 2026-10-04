# wi-21671-react-to-revocation-events-emitted-by-the-worker

管理 API の外から User を止めたときにも、所有する Agent の失効エポックが進み、外部の受信側へ失効が伝わるようになった。
対象はライフサイクルワークフローの `disable_user` の手順と、SCIM の取り込みによる無効化である。
これまでは、両者が Agent を無効化しても発行済みのトークンはイントロスペクションで有効のまま残り、SET の配送も作られなかった。
対象は [REQ-PLATFORM-001](../../domain/scenarios.feature.md) と [REQ-SHAREDSIGNALS-007](../../domain/sharedsignals/revocation/acceptance.feature.md) である。

`ISSUER` は `idmagic` だけでなく `idmagic-worker` も読むようになった。
設定方法は[アップグレードノート](../upgrades/wi-21671-react-to-revocation-events-emitted-by-the-worker.md)を参照する。
