---
status: completed
archived_from: a8e03c729bc0cb0163e004b8abe40c9d40181425
---

# デプロイとインフラの設計を Docker Compose と Google Cloud の具体で書く

デプロイとインフラの設計が、三つのデプロイプロファイル（ローカル Docker Compose、汎用 Kubernetes、Google Cloud）ごとの具体で読めるようになった。
共通トポロジーは構成要素名（`idmagic-frontend`、`idmagic-api`、`idmagic-worker`、`idmagic-batch`、`idmagic-seed`、`psqldef`）で描かれ、プロファイルごとの配置先、構成図、プラットフォームとネットワークの設計項目を持つ。
Google Cloud プロファイルのコンピューティングは GKE Autopilot とし、エッジは GKE Ingress、静的アセットは `idmagic-frontend` のイメージと Cloud CDN とした。Cloud Run 案は採らなかった選択肢として比較とともに残り、`infra/deploy/gcp/` のひな型もその位置付けになった。
汎用 Kubernetes プロファイルには、単体で動くよう CloudNativePG の PostgreSQL と `psqldef` と `idmagic-seed` の Job が（仮）として加わった。構成ファイルへの反映はまだ無い。
ネットワーク設計は、ファイアウォールルールの表とプロファイルごとの実施の仕組み、Cloud CDN と Cloud Armor の評価順序、リスナーの公開範囲、管理アクセスを持つ。
Docker Compose の `idp` サービスは `api` になり、Caddy、Prometheus、Alloy の参照も追従した。
規範文書は語だけが変わった。`mise run spec-diff` は REQ-SYSTEM-001、016、017、REQ-IDMANAGEMENT-009、REQ-OAUTH2-048、REQ-PROVISIONING-005、REQ-TENANCY-021 と四つの標準要求を挙げるが、変わったのは「既定」から「デフォルト」への表記と、REQ-SYSTEM-001 の Given にあった実在しない「イベントリレー」を実在の実行単位名へ直したことだけで、条件も要求する結果も同じである。
文書全体で「資材」「既定」「コンピュート」「参照トポロジー」「参照プロファイル」を使わなくなり、`mise run check-terminology` が再発を拒否する。
現在状態の文書は work item を参照しなくなり、`mise run check-work-item-references` が再発を拒否する。
Google Cloud を構成ファイルから生成できる形にする作業は、別の記録 [[wi-592-generate-infrastructure-materials-from-parameter-definitions]] として起票した。

全文は `git show a8e03c729bc0cb0163e004b8abe40c9d40181425:<このファイルのパス>` で読む。
