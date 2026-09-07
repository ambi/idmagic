# 実行時アーキテクチャ

## 目的

本書は、論理アーキテクチャを実行時のプロセスと通信へ写像する。レプリカ数、配置先、ネットワーク制御、可用性、容量はそれぞれの設計文書が所有し、本書は値を再掲しない。

## 実行単位

| 実行単位 | 入口 | 責務 | 状態 |
| --- | --- | --- | --- |
| API | `backend/cmd/idmagic` | OAuth2/OIDC、管理、アカウント、SCIM、Shared Signals などの HTTP リクエストを処理する | 正となる業務状態をプロセス内に保持しない |
| Worker | `backend/cmd/idmagic-worker` | キューに記録された遅延・再試行可能なジョブを処理する | PostgreSQL のジョブ状態を取得して確定する |
| Batch | `backend/cmd/idmagic-batch` | 保持期限に基づく削除など、横断的な保守処理を実行する | PostgreSQL の対象状態を更新する |
| Seed | `backend/cmd/idmagic-seed` | 初期データを投入する一回限りの管理実行単位 | PostgreSQL に結果を確定する |
| Frontend gateway | `frontend` | ブラウザー向け画面と同一オリジンの API 中継を提供する | ブラウザーセッション以外の正となる業務状態を持たない |

API は通常、複数の Bounded Context を一つのプロセスに組み立てる。ジョブとバッチは処理時間と再試行の性質が HTTP リクエストと異なるため、別の実行単位にする。API の用途別分割は [System Context の判断](../contexts/system/decisions.md#no-api-plane-separation) の再検討条件を満たすまで採らない。

## 通信

ブラウザーは Frontend gateway と通信し、gateway が API へ HTTP リクエストを中継する。ブラウザー Cookie を用いる画面と API は同一オリジンで公開する。外部クライアントと上流の IdP は公開 HTTP エンドポイントへ到達し、API と Worker は PostgreSQL を共有する。署名鍵や可逆な秘密情報の保護に外部提供元を選ぶ場合、対象の実行単位だけがその提供元へ接続する。

Context 間の同期処理は、公開されたポートを `backend/cmd/internal/bootstrap` の組み立て地点で接続する。監査とセキュリティ通知に渡すドメインイベントは同じ組み立て地点の単一の配信点を通り、発行側と消費側を直接依存させない。公開するイベント語彙と互換性は [構造](../structure.md#context-間イベント) が所有する。

## 実行時の規則

- すべてのバックエンド実行単位は `backend/cmd/internal/bootstrap` の共通設定と依存注入を使い、その外で環境変数を直接読まない。
- API のレプリカ間で正となる状態を共有メモリに置かない。認証セッション、認可コード、ジョブ、スロットルなどの共有状態は PostgreSQL に置く。
- HTTP の生存性と準備状態は異なるプローブで表す。データベースへ到達できず新しいリクエストを安全に処理できない API は準備完了にしない。
- 停止時は新規要求の受け入れを止め、処理中のリクエストまたはジョブへ定めた猶予を与えてから終了する。

## 関連文書

- 物理的な配備構成と環境差は [配備アーキテクチャ](deployment.md) が所有する。
- 基盤資源と Kubernetes の方針は [基盤設計](../design/infrastructure/platform.md) が所有する。
- 通信経路とネットワーク境界は [ネットワーク設計](../design/infrastructure/network.md) が所有する。
- レプリカ配置と障害耐性は [可用性設計](../design/reliability/availability.md) が所有する。
- 負荷、入場制御、縮退は [性能設計](../design/performance/README.md) が所有する。
- 信頼境界と攻撃者モデルは [脅威モデル](../design/security/threat-model.md) が所有する。
