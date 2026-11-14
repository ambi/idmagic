# 監査の記録の保持

## 概要

この文書は、監査の記録を種類ごとの保持期間だけ残し、期間を過ぎた記録を一括で消す仕様である。
掃引は、外部のスケジューラーが起動する `idmagic-batch retention-sweep` が実行する。
保持期間はテナントによらず同じである。

| 扱わないもの | 担当 |
| --- | --- |
| 保持期間より長い保管 | 利用者。[監査エクスポート](event-export.md)で取り出し、利用者の環境へ保管する |
| 認証イベントの集計、ログインセッション、既知の端末の掃除 | [Authentication](../../authentication/observability.md#相関と保持) |
| 保持期間を配置やテナントの設定で変えること | 提供しない |

## モデル

保持期間は記録の種類で決まり、記録の発生時刻から数える。
どの種類も有限の保持期間を持ち、無期限に残す種類はない。

| 種類 | 保持期間 |
| --- | --- |
| `AuthenticationFailed` | 30 日 |
| `AuthenticationEventAggregated` | 90 日 |
| `MfaChallengeIssued`、`MfaChallengeSucceeded`、`MfaChallengeFailed`、`BackupCodeConsumed`、`MfaEnrollmentRequired`、`MfaEnrollmentCompleted`、`MfaEnrollmentBypassIssued`、`MfaEnrollmentBypassConsumed`、`MfaEnrollmentBypassRevoked`、`MfaEnrollmentBypassExpired` | 90 日 |
| `SessionEnded` | 90 日 |
| それ以外のすべての種類（なりすましの `SessionImpersonationStarted` と `SessionImpersonationEnded`、各モジュールの管理操作を含む） | 365 日 |

## 操作

### バッチによる保持期間の掃引

#### REQ-AUDIT-013 保持期間の掃引は、保持期間を過ぎた監査の記録だけを検索属性とともに消す

- バッチが保持期間の掃引を実行したとき、Audit は、発生時刻からモデルの表の保持期間を過ぎた監査の記録を、その検索属性とともに削除する。
- バッチが保持期間の掃引を実行したとき、Audit は、保持期間の内側の監査の記録を変えない。
- **判断**：[監査の記録を種類ごとの有限の期間だけ保持する](../decisions.md#監査の記録を種類ごとの有限の期間だけ保持する)。

## セキュリティ上の考慮

`AuthenticationFailed` は、実在しないアカウント名も追跡するため、平文のユーザー名を持つ（[Audit のデータ](../data.md#集約)）。
その保持期間を 30 日に限り、平文のユーザー名が残る期間を短くする。
なりすましの記録は、対象の利用者を守るために短い期間へ入れず、デフォルトの 365 日だけ残す。
