# 承認リクエスト

## 概要

この文書は、Agent の操作に対する人間の承認を、CIBA のバックチャネル認可要求として扱う仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | バックチャネル認可要求の受け付け、対象のユーザーによる承認と拒否、承認の成立の後のトークンの発行、`Supervised` な Agent の承認の要求 |
| 行為者 | Agent、登録済みのクライアント、ResourceOwner（承認する本人） |
| 扱わないもの | `supervised` の Agent に承認を義務付けるかの方針は、ガバナンスの層が決める |

## モデル

承認の判断の記録には、UUID を鍵とし通信の方式に依存しない `ApprovalRequest` を使う。
CIBA の検索のフィールドとポーリングのフィールドは通信の上の記録だが、ストアが判断とポーリングを不可分に直列化できるよう、同じ場所に置く。
承認済みの要求はストアの単位の Compare-and-Set でトークンに交換し、同時のポーリングによる二重の発行を防ぐ。
それ以外の状態は `/token` で拒否する。

CIBA の配信のモードは `poll` だけを実装する。
承認の画面はすでにユーザーのセッションとステップアップ認証で保護されているので、`user_code` は未対応として広告する。

- **判断**：承認を CIBA で実装する理由は、[Agent の操作に対する人間の承認を CIBA で実装する](../design/decisions.md#agent-の操作に対する人間の承認を-ciba-で実装する)。

## 状態遷移

### ApprovalRequestLifecycle

人間の承認を待つ ApprovalRequest のライフサイクル。Pending から Approved / Denied / Expired へ一方向に進み、Consumed へ到達できるのは Approved からだけである。Consume は保存層の CAS でちょうど一度だけ成立し、並行するポーリングが二重にトークンを得ることはない。

| State | Kind | Meaning |
|---|---|---|
| Pending | initial | 起票済み。人間の判断を待っている |
| Approved | — | 人間が承認した。1 回だけトークンへ引き換えられる |
| Denied | terminal | 人間が拒否した |
| Expired | terminal | 判断または引き換えの前に有効期間が切れた |
| Consumed | terminal | 承認をトークンへ引き換えた。CAS によりちょうど一度だけ成立する |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Pending | Approve | now() < expires_at | Approved |  |
| Pending | Deny | — | Denied |  |
| Pending | Expire | — | Expired |  |
| Approved | Consume | now() < expires_at | Consumed |  |
| Approved | Expire | — | Expired |  |

## 操作

### Agent によるバックチャネル認可要求

#### REQ-OAUTH2-041 バックチャネル認可要求は人間の承認が成立してからトークンを発行する

#### REQ-OAUTH2-050 `Supervised` な Agent は人間の承認を経ずに新しいトークンを得られない

#### REQ-OAUTH2-042 承認が成立していない承認リクエストはトークンを発行しない

### 本人による承認の判断

#### REQ-OAUTH2-043 承認リクエストを判断できるのは対象ユーザー本人のステップアップ認証済みセッションだけである
