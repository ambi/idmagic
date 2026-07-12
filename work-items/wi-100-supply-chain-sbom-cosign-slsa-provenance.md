---
depends_on: []
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-04
---

# ADR-020 を実装し SBOM 同梱・cosign keyless 署名・SLSA provenance をリリース成果物に付与する

## Motivation
ADR-020 はサプライチェーン保護として SLSA Level 3・Sigstore(cosign) 署名・
CycloneDX SBOM 同梱・再現性検証を「採用」と決定しているが、現行 CI
(idmagic-ci.yaml) が実装しているのは Trivy スキャン・CodeQL・frozen-lockfile・
no-push のイメージビルドまでで、SBOM 生成、cosign 署名、SLSA provenance、
Rekor 透明性ログ、semgrep OAuth ルールは未実装。OAuth2/OIDC IdP は他システムが
認証を委ねる pivot 攻撃の起点であり、成果物が改ざん不能で内容を即答できる体制は
本番前提である。ADR は在るが実装が無い状態を WI として着地させる。

Kubernetes リリースや Sigstore が示すとおり、keyless 署名（GitHub OIDC）+ SBOM
attestation + 検証可能な provenance を「リリースと atomic に」生成することが要点で、
リリース後生成では攻撃時に稼働版と SBOM が乖離する（ADR-020 が却下した案）。

## Scope
- **ci_release**:
  - リリース時にコンテナイメージへ Syft で CycloneDX SBOM を生成し、 `cosign attest` で attestation として署名する。
  - `cosign sign`（GitHub OIDC keyless）でイメージ署名し、Rekor に記録する。 検証手順（`cosign verify` / `cosign verify-attestation`）を文書化する。
  - SLSA provenance generator でビルド provenance を生成し non-falsifiable に保つ。 到達可能な SLSA レベルを明記する（GitHub-hosted runner 前提）。
  - semgrep で OAuth 2.0 Security BCP 由来の禁止パターン ruleset を CI に追加する （ADR-020 の依存スキャン節）。
  - WI-99 の version stamp と、署名対象イメージ tag・SBOM を対応付ける。
- **documentation**:
  - README に成果物検証手順（cosign verify、SBOM 取得、provenance 確認）を書く。
  - Kubernetes 側で未署名イメージを拒否する admission（Policy Controller 等）の前提を記す。

## Out of Scope
- Kubernetes admission controller 本体のデプロイ・運用（前提として記述のみ）。
- SLSA L4（two-person review）— ADR-020 が out of scope とする。
- アプリケーションコード・HTTP API の変更。
- UI バンドルの本番堅牢化（別途 WI-104 系の範囲）。

## Plan
- [[ADR-020-supply-chain-protection]] の対象を現行成果物（Go binaries、frontend assetsを含むcontainer image）へ具体化する。現在は `.github/workflows/idmagic-ci.yaml` だけなので、PR CIを変更せずtag/手動dispatch専用のrelease workflowを追加する。
- buildは既存`just build-go`/`just build-ui`と`infra/docker/Dockerfile`を正本にし、version/commit/dateを固定したartifactとimage digestを一度だけ生成する。SBOM/署名のために別buildしてdigestをずらさない。
- CycloneDX SBOMはGo module、frontend package、container filesystemを対象にartifact/image digestへ紐付け、release artifactとOCI attestationの双方に格納する。
- cosign keyless署名はGitHub Actions OIDC、SLSA provenanceは公式generator/reusable workflowを用いる。長寿命signing keyをrepository secretに置かず、workflow permissionを最小化・actionをcommit SHA pinする。
- release publish前にidentity issuer/workflow ref、signature、provenance subject digest、SBOM schemaを別verify jobで検査する。READMEには利用者が同じpolicyで検証するcommandとexpected identityを記載する。

## Tasks
- [ ] T001 [Inventory] release対象、artifact名/platform、container registry、tag→version/commit mappingとADR-020未実装差分を確定する。
- [ ] T002 [Build] release用just recipe/workflowでGo/UI/containerを一度だけbuildし、checksumsとimmutable digestをjob output/artifactへ渡す。
- [ ] T003 [SBOM] pinned generatorでbinary/module/frontend/containerのCycloneDX SBOMを生成・validateし、checksum/OCI subjectへ関連付ける。
- [ ] T004 [Sign] GitHub OIDC permissionを持つ隔離jobでartifact/imageをcosign keyless署名し、transparency log bundleを保存する。
- [ ] T005 [Provenance] SLSA generatorでbuild inputs/subject digestをattestし、release/imageへ付与する。
- [ ] T006 [Verify/Publish] expected issuer/repository/workflow ref、全subject digest、SBOM/provenance schemaを検証した後だけreleaseをpublishするgateを追加する。
- [ ] T007 [Hardening/Docs] action SHA pin、least permissions、artifact retentionと利用者検証/runbookを記載し、test tagでend-to-end検証する。

## Verification
- CI: リリースワークフローで SBOM・署名・provenance が生成され、`cosign verify` と `cosign verify-attestation` が成功する.
- CI: semgrep OAuth ruleset がパイプラインで実行される。
- 手動: 生成されたイメージに対し外部から cosign verify が透明性ログ込みで通ることを確認する。
- `just build-go`

## Risk Notes
keyless 署名は GitHub OIDC 権限（id-token: write）とワークフロー分離に依存する。
権限過多や pull_request からの署名を避け、署名・attest は push/tag トリガに限定する。
SLSA generator の版差でワークフロー構文が変わりやすいので固定版で導入する。
