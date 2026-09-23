---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-23
priority: p2
depends_on: [wi-552-back-signing-keys-examples-with-tests]
change_kind: feature
affected_spec:
  - { path: docs/domain/signing-keys/scenarios.feature.md, requirement: REQ-SIGNINGKEYS-002 }
  - { path: docs/domain/signing-keys/scenarios.feature.md, requirement: REQ-SIGNINGKEYS-006 }
---

# ライフサイクルバッチが XmlFederationSigning 鍵も周期でローテートし、アーカイブする

## 動機

[[wi-552-back-signing-keys-examples-with-tests]] で SigningKeys の具体例を実装と照合したところ、`XmlFederationSigning` 鍵をローテートする入口が製品に 1 つも無かった。
wi-552 は管理 API `RotateTenantSigningKey` に `usage` を足し、管理者がローテートできるようにした。
一方、バッチ `idmagic-batch signing-key-lifecycle` はテナントごとの文脈に用途を足さないため、`Signing` 鍵だけを周期でローテートし、アーカイブも `Signing` 鍵だけに行う。

`XmlFederationSigning` 鍵は、管理者が手でローテートしない限り、初回の参照で遅延生成された鍵が使われ続ける。
猶予期間を過ぎた XML 鍵にはアーカイブの記録（`SigningKeyArchived`）も残らない。
SAML の IdP プロファイルごとのスコープを持つ鍵も同じである。

## 対象範囲

- `signing-key-lifecycle` が、各テナントの `XmlFederationSigning` 鍵を `Signing` 鍵と同じ周期と猶予期間でローテートし、猶予期間を過ぎた鍵をアーカイブする。
- SAML の専用 IdP プロファイルのスコープを持つ鍵も対象にするかを Design で決める。

## 対象外

- 鍵の用途、スコープ、猶予期間の規則そのものの変更。
- 管理 API と管理画面の操作。

## 検証

- 周期を過ぎた `XmlFederationSigning` 鍵がバッチの実行でローテートし、猶予期間を過ぎた旧鍵が `SigningKeyArchived` を残してメタデータから外れることを、バッチの入口から観測する。
- `mise run verify`

## リスク

- **既存の SAML 連携の信頼を切る。** ローテーション後に旧証明書をメタデータから早く落とすと、旧鍵で署名済みのメッセージを受け取った SP が検証できない。猶予期間の公開は既存の振る舞いを保つ。
- **プロファイルごとのスコープを取りこぼす。** SAML IdP プロファイルごとの鍵は用途に加えてスコープで分かれる。デフォルトのスコープだけを回すと、専用プロファイルの鍵が回らないまま残る。
