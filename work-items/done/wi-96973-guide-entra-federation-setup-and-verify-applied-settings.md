---
status: completed
---

# Entra フェデレーションの導入を、設定の保存から反映確認まで案内する

REQ-WSFEDERATION-001 を変え、TypeSpec に `ListEntraFederationDomains`、`GetEntraFederationDomain`、`RecordEntraFederationObservation` と観測文書、確認、差分、証明書の要約、`not_found` と `observation_outdated` のモデルを加え、`ConfigureEntraFederation` と `EntraDomainFederation` に任意の `entra_tenant_id` を加えた。
Entra の構成は保存しただけでは `unverified` であり、Graph PowerShell の読み戻しの JSON を管理者が取り込むと、WsFederation が期待値と比べて `matched` か `mismatched` を記録し、構成や XML 署名鍵が変わると期待値の指紋の違いから `stale` を返す。
詳細は RP を変えずに再取得でき、接続（テナントを限る確認つき）、確認だけの接続、変更前の保存、差分、適用、読み戻しの手順とコマンドを返す。
観測記録は WsFederation が所有する `wsfed_entra_federation_observations` にドメインごとの最新の一件を置き、古い取得時刻の上書きを条件付き upsert で拒否する。新しいドメインの構成では前の記録を消す。
管理画面は一覧に確認の状態を示し、追加と編集の保存後にドメインの詳細へ移り、詳細で手順、証明書、取り込み、導入の前提と構成後の確認を示す。導入の runbook を追加した。
Entra のテナント ID は利用者の判断でドメインごとの任意の値とし、持たないドメインでは観測したテナントを記録するだけで比べない。
取り込んだ JSON が Graph 由来であることは確かめず、THREAT-104 として受容し、照合を自動の判断に使うときに見直す。
実 Entra での手順の実行は WI-79 の受入試験に残り、確認したのはテスト内の観測文書と実ブラウザーの経路までである。
