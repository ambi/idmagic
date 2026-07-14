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

- authorization code は最大 60 秒かつ single-use、PAR request URI は 600 秒かつ single-use とする。
- access token は 600 秒、ID token は 3600 秒、refresh token は通常 14 日・絶対 30 日とする。
  refresh rotation は絶対期限を延長しない。
- device code と user code は 600 秒、既定 polling interval は 5 秒、`slow_down` ごとの増分は
  5 秒とする。
- client authentication failure は 1 分あたり 10 回、authorization code redemption failure は
  1 分あたり 5 回を上限とする。
- DPoP proof は過去方向 60 秒・未来方向 5 秒の clock skew を許容し、JTI replay window は 10 分、
  nonce lifetime は 60 秒とする。
- refresh token reuse alert window は 60 秒、Consent record retention は 7 年とする。
- 強制可能な値は `models` constraint、`states` guard、`interfaces` requires/ensures と scenario に
  置く。単一要素へ自然に所属しない運用設定は本 ADR を正本とする。

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
