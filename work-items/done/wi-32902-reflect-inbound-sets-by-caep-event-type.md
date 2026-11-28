---
status: completed
---

# 受信した SET の反映を CAEP のイベントの種類と購読で決める

受信した SET の反映を、イベントの種類と受信側のストリームの購読で決めるようにした（REQ-SHAREDSIGNALS-019 を追加し、REQ-SHAREDSIGNALS-010 を変更した）。
失効させてセッションを終わらせるのは、購読する `session-revoked` だけである。
購読する `token-claims-change`、`credential-change`、`assurance-level-change` は、主体を解決し、`reflected_at` のない受理を記録するだけにした。
購読しない種類と `CaepEventType` にない種類は、署名の検証の後に 202 で受理して無視する。`jti`、主体、記録は扱わない。
これで、未知の種類で 500 を返していた欠陥もなくなった。

拒否ではなく無視を選んだのは、RFC 8935 §3 が無視を認めるためである。
また、go-ssf、SSFgo、IdMagic 自身の送信側はどれも 400 を配送の失敗として数え、送信側によってはストリームを止める。
credential-change は、Okta の送信側が要素の無効化やパスワードのリセットでも送るので、`change_type` によらず失効させない。
資格情報の取り消しで失効させたい送信側は、`session-revoked` を送る必要がある。

CAEP 1.0 の 13 例はすべて再生し、`caepNotReplayed` をなくした。
risk-level-change の例は未知の種類として無視するので、RFC9493-SUBID-ISS-SUB はこのテストでは観測しなくなった。この標準は、既存の usecases と HTTP のテストが引き続き検査する。
記録だけの種類を受けたときのドメインイベントと、`events` が複数のメンバーを持つ SET の扱いは対象外として残した。
