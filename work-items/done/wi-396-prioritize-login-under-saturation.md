---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 容量が足りないときにログインが管理系トラフィックより先に生き残るようにする

`mise run spec-diff` が報告した規範的な差分は `added scenarios: REQ-SYSTEM-018, REQ-SYSTEM-019` の 2 件である。REQ-SYSTEM-019 は、経路の優先度クラスを実装を読まずに参照できることを、`CONFIGURATION.md` と同じ生成＋乖離検出の形で定める。飽和した API プロセスが優先度の低い要求から拒否すること、拒否がハンドラーへ到達しないこと、`infrastructure` の経路は拒否しないこと、飽和していないときは何も拒否しないことを、新しい振る舞いとして加えた。既存のシナリオは 1 つも変えていない。
契約側の差分は無い。飽和時の 503 は routing の後、どのハンドラーの手前でも同じに返り、呼び出し側の対応も `Retry-After` の秒数だけ待って再送する 1 つしかないので、`docs/api-rules.md` の Declared status codes に、ミドルウェアが返す 3 つ目の例外として記録した。TypeSpec の operation は 1 つも変えていない。
設定は 4 つ増えた（`ADMISSION_CONTROL_ENABLED`、`ADMISSION_MAX_CONCURRENT_REQUESTS`、`ADMISSION_MANAGEMENT_MAX_CONCURRENT_REQUESTS`、`ADMISSION_MANAGEMENT_BULK_MAX_CONCURRENT_REQUESTS`）。メトリクスは 2 つ増えた（`http_admission_decisions_total`、`http_admission_in_flight_requests`）。配備側では API の HorizontalPodAutoscaler が加わり、オーバーレイの固定 `replicas` が消えた。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
