# wi-92540-guarantee-provisioning-deliveries-by-reconciliation

外向きのプロビジョニングは、書き込み時の捕捉に加えて、`worker` の定期的な照合で下流への反映を保証するようになった。
LifecycleWorkflow、SCIM の取り込み、CSV インポートのように、書き込み時の捕捉を呼ばない経路で User が変わった場合や、捕捉が失敗した場合も、次の照合がプロビジョニングタスクを作る。
照合は、有効で隔離されていない接続ごとに、User のあるべき状態と下流へ反映済みの状態を突き合わせる。
対象は [REQ-PLATFORM-003](../../domain/scenarios.feature.md) と [REQ-PROVISIONING-019](../../domain/provisioning/scenarios.feature.md) である。

照合の周期は `PROVISIONING_RECONCILE_INTERVAL` で設定する。デフォルトは 5 分である。
1 回の照合で 1 接続に作るプロビジョニングタスクは 500 件までで、残りは次の照合が拾う。
照合の対象は User だけで、Group は含まない。
