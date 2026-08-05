---
status: accepted
authors: [tn]
created_at: 2026-07-15
---

# ADR-109: OAuth2 の lifetime・security・retention 設定を SLO から分離する

## コンテキスト

SCL 3.0 の `objectives` は、boolean indicator、numeric target、window、budgeting で測定できる
SLO に限定する。旧 OAuth2 context の `objectives` には、token/code/PAR の TTL、rate limit、
DPoP clock skew・replay window、device polling interval、Consent retention が混在していた。
これらは可用性やレイテンシの達成率ではなく protocol・security・運用設定であるため、値を
変更せず所有先を分離する必要がある。

## 決定

token/code/PAR の TTL、rate limit、DPoP clock skew・replay window、device polling interval、
Consent retention の具体的な値は現在 [`backend/oauth2/ARCHITECTURE.md`](../backend/oauth2/ARCHITECTURE.md)
の「Lifetime, security, and retention configuration」に記載する。強制可能な値は `models`
constraint、`states` guard、`interfaces` requires/ensures と scenario に置く。単一要素へ自然に
所属しない運用設定は本 ADR を正本とする。

## 却下した代替案

- 旧 `objectives` の kind をそのまま SCL 3.0 に持ち込む: SLO と設定値が再び混在し、objective の
  error budget semantics が成立しない。
- すべてを model field constraint にする: rate limit、clock skew、retention は単一 aggregate の
  妥当性ではなく、複数 request・時間窓・運用 lifecycle にまたがる。
- 値を runtime 実装へだけ残す: specification core から security boundary が失われる。

## 影響

- `spec/contexts/oauth2.yaml` の `objectives` には latency、error rate、availability、throughput の
  測定可能な SLO だけが残る。
- authorization code / refresh token / PAR の局所条件は同文書の model constraint、state guard、
  interface contract、scenario に反映する。
- 本決定は仕様表現の移行であり、protocol wire behavior、runtime 設定値、token format を変更しない。
