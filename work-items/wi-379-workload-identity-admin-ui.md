---
status: pending
authors: [tn]
risk: low
created_at: 2026-08-16
priority: p2
depends_on: [wi-54-workload-identity-federation-spiffe]
change_kind: feature
affected_spec:
  - { path: spec/contexts/workloadidentity/SPECIFICATION.md, requirement: REQ-WORKLOADIDENTITY-008 }
---

# ワークロード ID の信頼設定とバインディングを管理コンソールから運用できるようにする

## Motivation

`WorkloadIdentity` は 13 本の管理 API を持ち、`WorkloadTrustBundle` の登録・更新・無効化・再有効化・削除・JWKS 再取得と、`AgentWorkloadBinding` の作成・無効化・再有効化・削除を提供する (REQ-WORKLOADIDENTITY-008)。しかし `frontend/src` に `workload` は **0 件**である。同じ管理対象である `admin-agents` と `admin-mcp-resource-servers` には画面があるのに、ワークロード ID だけが API 専用のまま残っている。

信頼バンドルは、外部の発行者が署名した JWT を IdMagic のトークンへ交換してよいと宣言する設定である。これは「長期シークレットを配布せずにエージェントへ資格情報を渡す」という設計の要であり、同時に**登録を誤れば任意の発行者がエージェントになりうる**設定でもある。`subject_pattern` はグロブで、曖昧な一致は拒否されるとはいえ、どのパターンがどのエージェントへ向いているかを一覧できなければ、意図しない広さの設定に気付けない。

運用事故を想定すると欠落がはっきりする。上流の鍵が漏れた疑いがあるとき、運用者はまず「どの信頼バンドルが有効か」を見て、当該のものを無効化し、JWKS を再取得する必要がある。現在それを行う手段は API を直接叩くことだけである。認証情報の配布経路を止める操作が、画面から行えないまま残っている。

`WorkloadTrustBundleJWKSRefreshed` や `WorkloadAttestationRejected` (11 種の理由コードを持つ) といったイベントも既にあるが、参照する場所が無い。アテステーションが拒否されている原因を運用者が特定できない。

## Scope

- 管理コンソールに `WorkloadIdentity` の機能を追加する。既存の `admin-agents` / `admin-mcp-resource-servers` と同じ構成・同じ命名規則に従う。
- 信頼バンドルの一覧・詳細・登録・更新・無効化・再有効化・削除・JWKS 再取得を画面から行えるようにする。一覧では `trust_domain` / `issuer` / 状態 / `jwks_cached_at` が読めること。
- バインディングの一覧・作成・無効化・再有効化・削除を、所属する信頼バンドルの詳細から行えるようにする。`subject_pattern` と対象 `Agent` の対応が一覧で読めること。
- 直近の `WorkloadAttestationRejected` を理由コードとともに提示し、設定の誤りを運用者が特定できるようにする。理由コードは英語の識別子のままにせず、UI 文言として `ja` / `en` の両方を辞書へ置く。
- 破壊的操作 (削除、無効化) には確認を挟む。削除はカスケードでバインディングを消すため、影響範囲を確認画面に示す。
- 既存の権限をそのまま使う。`workload-identity:read` / `workload-identity:write` は配線済みであり、新しい権限は作らない。

## Out of Scope

- 仕様と Go 実装の変更。API は 13 本すべて存在し、本 work item は画面を足すだけである。表示に必要な項目が API に無いことが判明した場合のみ、その差分を Scope へ繰り入れる。
- `WorkloadIdentity` の HTTP エラー応答の RFC 9457 移行。[[wi-338-workloadidentity-context-problem-details-migration]] が持つ。
- X.509-SVID と SPIRE サーバーの同梱。[[wi-54-workload-identity-federation-spiffe]] が将来送りと判断済みで、その判断は変えない。
- 複数の資格情報バインディングを持つ `Agent` に対する選択規則。現行実装は最初のバインディングを採るが、これは仕様に無い挙動であり、画面で見えるようにはするが規則の設計は行わない。

## Design

未定。着手時に、`admin-mcp-resource-servers` の構成をどこまで踏襲するかを確認して記録する。信頼バンドルとバインディングは親子関係にあり、MCP リソースサーバーの平坦な一覧とは形が違うため、詳細画面に子を埋め込むか別画面に分けるかを決める。

## Plan

- 既存の管理機能の構成をそのまま写す。フロントエンドの表示ロジック分離の方針 (提示ロジックを分離し、単体テスト可能にする) に従う。
- UI 文言は `*.i18n.ts` の辞書に `ja` / `en` の両方を置く。テストは辞書の値を参照し、翻訳済み文字列を直接書かない。
- アテステーション拒否の提示は、既存の監査検索を再利用できるかを最初に確認する。専用の読み取り経路を作るのは、再利用できないと分かってからにする。

## Tasks

- [ ] T001 [Design] 信頼バンドルとバインディングの画面構成を確定し `## Design` に記録する。
- [ ] T002 [UI] 信頼バンドルの一覧・詳細・作成・更新・状態変更・削除・JWKS 再取得を実装する。
- [ ] T003 [UI] バインディングの一覧・作成・状態変更・削除を実装する。
- [ ] T004 [UI] アテステーション拒否の提示と、理由コードの `ja` / `en` 辞書を追加する。
- [ ] T005 [Verify] 提示ロジックの単体テストと、権限不足時に操作が現れないことを検証する。

## Verification

- `just verify-ui`
- `just verify`
- 手動: 信頼バンドルを登録し、バインディングを作成し、K8s の ServiceAccount トークンでトークン交換が通ること、バンドルを無効化すると交換が拒否され、その拒否が理由コードとともに画面に現れることを確認する。
- 手動: `workload-identity:read` のみの権限で変更操作が現れないことを確認する。

## Risk Notes

リスクは low。既存 API に画面を足すだけで、仕様も認可も変えない。

ただし信頼バンドルの削除はバインディングをカスケードで消すため、確認画面で影響範囲を示さないと、一覧から軽い気持ちで消せてしまう。破壊的操作の確認は実装時に省略しない。
