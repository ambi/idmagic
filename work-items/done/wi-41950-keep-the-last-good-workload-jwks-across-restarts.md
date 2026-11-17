---
status: completed
---

# 最後に取得できた信頼設定の鍵を、プロセスの再起動後も猶予の期間だけ使えるようにする

REQ-WORKLOADIDENTITY-018 を追加し、`jwks_uri` から最後に取得できた鍵の集合を信頼設定の行（`jwks_cached` と `jwks_cached_at`）に保存して、プロセスの再起動後も取得から 24 時間まで署名の検証に使えるようにした。
保存は鍵の集合が変わったときか、保存から 1 時間を過ぎたときだけ行い、保存の失敗は検証の結果を変えない。
REQ-WORKLOADIDENTITY-008 の再取得も同じ鍵として保存する。
`jwks_uri` を変えた更新は保存した鍵と `jwks_cached_at` を消し、取得元の変更と並行した古い取得や遅れた取得は条件付きの更新で書かない。
メモリの鍵も信頼設定と `jwks_uri` の組ごとに持つように直し、変更前の取得元の鍵を使わないようにした。
最後に取得できた鍵を使った検証は `idmagic_workload_jwks_fallback_total{source}` の `memory` と `stored` で区別する。
保存の失敗を数える指標は設けていない。
