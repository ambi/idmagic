---
status: completed
---

# 監査イベントを SSF のストリームで継続配送する

監査イベントを SSF のストリームで継続配送できるようにした。
独自 Webhook との併存は、受信側が一つの方式に対応すれば済むよう検討の末に採らず、SET を受けられない具体的な受信側が確認できたときの再検討に残した。

`mise run spec-diff` の意味上の差分は次のとおりである。

- REQ-SHAREDSIGNALS-014: 送信する SET に SSF 1.0 必須の `sub_id` を加えた。既存の失効の SET にも効く。
- REQ-SHAREDSIGNALS-015、016: `audit_event_types`（REQ-SHAREDSIGNALS-009 と TypeSpec を変更）を購読するストリームへ、監査の記録と同じ Tx で配送を作り、`urn:idmagic:secevent:audit:<種類>` の SET を送信時に署名して送る。本文は許可した識別子だけである。
- REQ-SHAREDSIGNALS-017: 配送先を公開の HTTPS に限り、登録時と接続時に解決先を検査する。プロキシ、接続の再利用、リダイレクトを使わない。
- REQ-SHAREDSIGNALS-018: 連続 20 回かつ 1 時間以上の失敗で `disabled` にして `SsfStreamAutoDisabled` を発行し、リースとストリームごとの上限で配送を隔離する。終端の配送は 30 日で消す。
- 標準: SSF10-SUB-ID、SSF10-STATUS-NO-TRANSMIT を追加した。既存の worker が `disabled` のストリームへ送っていた点を直した。

後から読む人が知るべき判断: 配送を Jobs に載せないのは、`JobEnqueued` の監査が捕捉へ戻る増幅と、`active_jobs` の上限が監査を失敗させることを避けるためである。Audit と SharedSignals は互いを import せず、組み立て地点が捕捉の関数を渡す。

確認した範囲: 宛先検査、プロキシ、組み立ての配線、再配信の捕捉、配送の結果の除外、無効化の 6 種類の故障を注入し、対応するテストが失敗することを確かめた。

確認の限界と残るリスク:

- `assemblePostgres` の配線の行そのものは実行しておらず、`WithCapture(captureAuditRecordInTx, nil)` と同じ組み立てのテストで代替した。
- CSV の経路（`AuditInTx`）の配線の行も同様に実行していない。
- 実際の SSF 受信製品との相互運用は未確認である。Splunk、Sentinel、Datadog が SET を直接受ける公式の機能は確認できず、SIEM への取り込みには中継が要る想定である。
- 外向きの通信をプロキシに限る配置では配送が届かない。
- 無効化中のイベントは配送されず、エクスポートで補う。
- Discovery と Stream Management API を持たないので、SSF への適合は部分的である。
- 管理画面は wi-72447 の担当で未実装である。
