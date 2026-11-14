---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-11
priority: p3
group: workload-identity
depends_on: [wi-25266-authenticate-agent-clients-with-jwt-svid]
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/standards.md, requirement: AIMS00-AGENT-CLIENT-AUTH }
  - { path: docs/modules/workloadidentity/features/attestation-exchange.md, requirement: REQ-WORKLOADIDENTITY-003 }
---

# WIT-SVID と WIMSE の Workload Proof Token に対応するかを判断して対応する

## 動機

JWT-SVID は Bearer であり、漏えいすれば有効期間の間に再生される。
IETF WIMSE は、鍵に束縛した Workload Identity Token（WIT、[draft-ietf-wimse-workload-creds](https://datatracker.ietf.org/doc/html/draft-ietf-wimse-workload-creds)）と、要求ごとに鍵の所有を示す Workload Proof Token（WPT、[draft-ietf-wimse-wpt-02](https://datatracker.ietf.org/doc/html/draft-ietf-wimse-wpt)）を定めている。
SPIFFE は WIT を WIT-SVID として規定し、[draft-ietf-oauth-spiffe-client-auth-03](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-spiffe-client-auth-03) の 3.4 節は WIT-SVID によるクライアント認証（`spiffe_wit`）を定める。
AIMS も、Agent の認証で WIT と WPT を挙げる。

IdMagic は、SPIFFE バンドルの鍵の `use` に `wit-svid` があることを認識するだけで、WIT-SVID を受け付けない。

## 対象範囲

- 判断を最初に行う。WIMSE のドラフトの状態、SPIRE の WIT-SVID の発行の対応状況、Attestation-Based Client Authentication のドラフトとの関係を調べる。
- 採ると判断したら、`spiffe_wit` によるクライアント認証の最小の範囲を仕様化して実装する。
- 見送ると判断したら、見直す条件を `## 要約` に書いて中止する。

## 対象外

- リソースサーバー側での WPT の検証。
- IdMagic が WIT を発行すること。

## 設計

判断の基準は次のとおりとする。

- WIMSE のドラフトが WG Last Call を過ぎているか。
- SPIRE の安定版が WIT-SVID を発行するか。
- 主要な IdP か、MCP の実装が WIT か WPT を受け付けているか。

二つ以上を満たさなければ見送る。
採る場合は、wi-25266 が足す WorkloadIdentity の検証の操作に WIT-SVID を足し、OAuth2 のクライアント認証で `spiffe_wit` として受ける。

## 計画

1. 調査して判断する。
2. 採る場合だけ、仕様と実装を進める。

## タスク

- [ ] T001 [Decision] 判断の基準に沿って調査し、採否を決める。
- [ ] T002 [Spec] 採る場合だけ、標準の表と WorkloadIdentity の仕様を更新する。
- [ ] T003 [Acceptance] 採る場合だけ、WIT-SVID と PoP でトークンを得る経路と、PoP の欠落による拒否を RED にする。
- [ ] T004 [App] 採る場合だけ実装する。

## 検証

- 採る場合は `mise run test-go-package -- ./backend/workloadidentity/...` と `./backend/oauth2/...`。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

- ドラフトの段階で公開契約に載せると、仕様の変更に追従する費用がかかる。判断の基準で抑える。
