---
status: pending
authors: [tn]
risk: medium
reversibility: irreversible
created_at: 2026-11-11
priority: p2
depends_on: [wi-25266-authenticate-agent-clients-with-jwt-svid]
change_kind: feature
affected_spec:
  - { path: docs/modules/workloadidentity/features/attestation-exchange.md, requirement: REQ-WORKLOADIDENTITY-001 }
  - { path: spec/modules/oauth2/models.tsp, symbol: IdMagic.Contract.AccessTokenClaims }
  - { path: spec/modules/oauth2/models.tsp, symbol: IdMagic.Contract.IntrospectionResponse }
---

# 検証した SPIFFE ID を、発行するトークン、イントロスペクション、監査に載せる

## 動機

ワークロードのアテステーションで発行するトークンは、Agent を `agent_id` と `agent_chain` で表すだけで、提示された SPIFFE ID を持たない。
`WorkloadTokenExchanged` も、信頼設定、関連付け、Agent、audience を載せるが、SPIFFE ID を載せない。

関連付けはパターン（例：`spiffe://example.org/ns/agents/*`）で多くのワークロードを一つの Agent にまとめるので、後から「どのワークロードのインスタンスが要求したか」を辿れない。
リソースサーバーと IdMagic の認可ポリシーも、SPIFFE ID のトラストドメインやパスを条件に使えない。
Google Cloud の Workload Identity Federation が属性の対応付けと CEL の条件を持つように、外部の主体の属性を保持して使うことは主要な IdP の共通の機能である。

## 対象範囲

- 検証した SPIFFE ID（`oidc` の信頼設定では `iss` と `sub`）を、`WorkloadAttestationRejected` を除く監査イベント（`WorkloadTokenExchanged`、wi-25266 の認証成功のイベント）へ載せる。拒否のイベントには、署名を検証できた場合に限って載せる。
- 発行するアクセストークンとイントロスペクションの応答に、外部の主体を表すクレームを載せる。
- 交換したトークンを Token Exchange でさらに交換するとき、このクレームを引き継ぐ。

## 対象外

- SPIFFE ID の属性を条件とする認可ポリシーの記述。クレームを載せた後で、[[wi-80811-decide-authzen-compatible-evaluation]] の評価の入力として扱うかを判断する。
- 関連付けの属性条件（CEL など）の導入。

## 設計

クレームの名前と形は着手時に決める。

| 案 | 内容 | 評価 |
| --- | --- | --- |
| A | 独自のクレーム（例：`workload` に `{ "iss": ..., "sub": ..., "trust_domain": ... }`）を載せる | 意味を IdMagic が定められ、OIDC 形式の主体も同じ形で表せる |
| B | RFC 8693 の `act` に外部の主体を入れる | `act` は現在の行為者の連鎖であり、交換を要求した主体の意味と食い違う |
| C | WIMSE の識別子の表現に合わせる | 仕様がまだ固まっていない |

案 A を第一候補とし、名前は IANA の JWT クレームの登録と、WIMSE の動向を確かめてから決める。
クレームは署名の検証の後の値だけから作り、選択の段で読んだ未検証の値を使わない（attestation-exchange.md の実現方式の規則）。

## 計画

1. クレームの名前と形を決める。
2. TypeSpec と仕様を更新し、監査、トークン、イントロスペクションの順に実装する。

## タスク

- [ ] T001 [Design] クレームの名前と形を決める。
- [ ] T002 [Spec] TypeSpec と仕様を更新する。
- [ ] T003 [Acceptance] 交換したトークンのイントロスペクションに SPIFFE ID が現れることを RED にする。
- [ ] T004 [App] 監査イベント、トークン、イントロスペクション、交換時の引き継ぎを実装する。
- [ ] T005 [Verify] 検証を実行する。

## 検証

- `mise run test-go-package -- ./backend/oauth2/...` と `./backend/workloadidentity/...`。
- `mise run check-schema`（localhost の待受けを使うため、サンドボックスの外で実行する）。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

- トークンに載せた値は外部へ出るため、トラストドメインやパスの命名に含まれる内部の構成情報が、リソースサーバーへ漏れる。テナントの設定で載せるかを選べるようにするかを T001 で決める。
- 公開したクレームの名前は後から変えにくい。
