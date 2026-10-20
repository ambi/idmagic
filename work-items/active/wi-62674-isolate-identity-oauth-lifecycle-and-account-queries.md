---
status: in_progress
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-18
priority: p3
depends_on: [wi-21934-decide-module-dependency-order-and-break-cycles]
change_kind: refactor
spec_impact: { kind: none, reason: "要求側のポートと境界の変換で IdManagement → OAuth2 を切り離し、HTTP の応答、認証と認可の判定、永続状態、イベントと既存の確定順を保つ。" }
---

# IdManagement の OAuth2 操作と照会を要求側のポートで分離する

## 動機

[依存の向きの決定](../done/wi-21934-decide-module-dependency-order-and-break-cycles.md)で、`IdManagement → OAuth2` を除く向きを選んだ。
公開パッケージの利用でもモジュールの循環は残るため、この組を独立して解消する。

## 対象範囲

User の同意、リフレッシュトークン、device code、承認要求の完全削除、Agent とクライアントの束縛と照会、本人データのエクスポートを対象にする。
モジュールのルート、公開型、HTTP、アダプターを含め、この辺を生む本番 import をすべて対象にする。
実装後に循環を再計測し、消えた違反 ID だけを台帳から除き、依存の図を生成し直す。

## 対象外

使用量再集計、削除の原子性、Agent の資格情報の意味と HTTP エクスポート項目の変更。
ほかの組の循環の解消は、この項目へ混ぜない。

## 設計

[ユーザーのライフサイクル](../../docs/modules/identity-management/features/user.lifecycle.md)の REQ-IDMANAGEMENT-050、[アカウント](../../docs/modules/identity-management/features/account.md)の REQ-IDMANAGEMENT-018、[Agent](../../docs/modules/identity-management/features/agent.md)の REQ-IDMANAGEMENT-074、078 と、`backend/idmanagement/internal/user/usecases/admin_users.go`、`internal/agent/usecases/admin_agents.go`、`internal/user/handlers_http/account_data_handler.go` を起点にする。

D2、D4、D6、D7 を適用する。
IdManagement に削除、Agent のクライアント操作、本人同意照会のポートと必要な値を定義し、OAuth2 の公開操作へ組み立て地点で接続する。
OAuth2 → IdManagement の主体照会は残す。
同意とトークンの保存型を IdManagement へ渡さず、状態所有者の検証と書き込みは OAuth2 に残す。

完全削除は同意、refresh、セッション、認証情報、device code、承認要求の既存の順を崩さないよう、Authentication 向けの別項目とも境界を照合する。
PendingPurge の進行、各保存先の確定、失敗後の再実行を維持する。
すべてを一操作の Tx にする案と非同期イベントに変える案は、現行の確定と完了の条件を変えるため採らない。
Agent の管理を OAuth2 へ移す案は主体の責務を分散させる。
トークンの保存方式を変える場合は OAuth2 の削除操作の内部へ閉じる。

束縛時の同一テナント確認と一つの Agent への束縛、本人以外をエクスポートできない条件を保つ。
[wi-83071](wi-83071-make-user-deletion-rerunnable-and-recount-quota-usage.md) の使用量再集計は混ぜない。
2つの逆向きの辺は別々に解消できるが、削除の順は両項目で共有する既存の操作列を基準にする。

### 着手時に確定した境界

入力は REQ-IDMANAGEMENT-050、018、074、078 と対応する TypeSpec 操作である。
D2、D4、D6、D7 を適用する。
利用者の指示に従い、循環を消すためだけの中継層、汎用アダプター、保存型の複製は増やさない。

| 用途 | 型と操作 | 隠す知識と接続 |
| --- | --- | --- |
| User に属するプロトコル記録の削除 | `user/ports.UserRecordsPurger.DeleteAllForSub(context.Context, string) error` | 同意、refresh、device、承認の保存先が直接満たす。4 保存先を受ける既存のフィールドと呼び出し順を保ち、ラッパーを作らない |
| 束縛先の存在確認 | `agent/ports.ClientLookup.ClientExists(context.Context, string, string) (bool, error)` | メモリと PostgreSQL のクライアント保存先が既存の `FindByID` で存在を調べる。テナントの選択と読み取り時のエラーを保ち、保存型を渡さない |
| 本人データの同意 | `func(context.Context, string) ([]user/ports.AccountConsent, error)` | OAuth2 の既存の `ListGranted` を組み立て地点で呼び、client_id、scopes、state、granted_at、expires_at だけへ写す。HTTP 応答の型をこの値へ移し、二段階の変換を作らない |

現状維持は保存先の全操作と保存型への依存が残るため不採用とする。
削除ごとの具象ラッパー、汎用のプロトコルサービス、新しいパッケージは、同じ操作を委譲するだけなので採らない。
クライアントの存在確認を組み立て地点のアダプターにする案より、既存の両保存先が小さな照会契約を直接満たす案を選ぶ。
同意は既存の応答型を要求側の公開値へ移すので、保存型と応答型の二重定義を増やさない。
OAuth2 の保存表現が変わる場合、削除と存在照会は所有者の内部、同意の表示変換は一つの組み立て地点だけが追従する。

削除は同意、refresh、セッション、認証情報、device、承認の順を維持する。
各保存先が独立に確定し、途中失敗を返し、Tombstone の `PendingPurge` から再実行する。
Agent の束縛とイベントは引き続き IdManagement が所有する。
新しい Tx、SQL、イベント、再入は加えない。
セキュリティ上は、テナント付きのクライアント照会と、認証済み本人からの同意照会を維持する。
互換性は HTTP の項目、空配列、スコープの複製、エラー伝播を保つ。
スキーマ移行はなく、差分の取り消しで戻せる。
未決事項はない。

既存の Agent の規範テストと本人データの HTTP テストを着手前に実行する。
User の削除と再実行も既存の規範テストで確かめる。
構成入口を含む事前照合の未解析操作は 282 件、テスト参照の不足は 0 件であり、実行の証拠とは区別する。
テストのパスを変更しない構造変更として、既存テストと境界検査の成功で確かめる。
この条件を維持する間は、受け入れ RED、単体 RED、ミューテーション検査は「振る舞いを保つ変更」の条件により適用しない。

## 計画

上記の小さな要求契約で依存を置き換え、既存の成功、拒否、途中失敗と再実行のテストを変更せず実行する。
境界検査で観測されなくなった循環だけを台帳から除き、現在状態の設計と依存図を同期する。
保証の追加や変更が必要になれば、構造変更と分けた仕様変更として扱う。

## タスク

- [x] T001 [Design] import の全入口と作用を再調査し、境界の型と操作、変換、確定順を設計へ記す。
- [x] T002 [App] 要求側のポートと所有側の公開操作を結び、逆向きの辺を解消する。User の削除と再実行、Agent の束縛と削除、本人データ取得の既存の規範テストを変更せず使う。
- [x] T003 [Docs] 公開範囲と現在の設計を同期し、実測で消えた負債だけを削除して依存図を再生成する。既存の公開パッケージを使い、台帳にあった 13 件の循環を削除する。
- [x] T004 [Verify] 既存の保証、正式な配線と境界を検査し、仕様差分を確認する。七つの視点で差分を読み、既存の保存型を複製せず、新しいパッケージ、DB の確定境界、任意の依存を増やしていないことを確認する。

## 検証

- `mise run test-go-package -- ./backend/idmanagement/internal/user/usecases` の削除と再実行の検査、`./backend/idmanagement/internal/agent/usecases` の束縛検査、本人データの HTTP 検査を起点にし、OAuth2 と正式な配線を `mise run test-go-changed` で確認する。
- `mise run lint-go`：変更した Go の静的検査を行う。
- `mise run check-boundaries` と `mise run check-boundary-debt-ratchet`：辺の解消、公開範囲と負債の非増加を確認する。
- `mise run check-contract-drift`、`mise run check-status-drift` と `mise run check-published-api-compat`：HTTP の表現と応答を変えていないことを確認する。
- `mise run check-work-items`、`mise run check-links`、`mise run spec-diff`：記録と参照、規範への影響を確認する。
- `mise run check-spec`：更新した現在状態の設計と公開範囲を検査する。
- `mise run test-go-changed -- main`：変更したパッケージと正式な HTTP 配線を含む逆依存を race 付きで検査する。DB モードと未実行の検査をレポートから確認する。

テストを変更しない構造変更は既存テストの成功を証拠にする。
保証を表すテストが欠けている場合は、公開状態に応じた特性化または要件のテストを実装より先に用意し、その段階の証拠と変更耐性の手順を選ぶ。

## リスク

import を消しただけで実行時の再入と必須の作用を失わないよう、変換前後の公開経路と各確定点を比較する。
必須の作用を省略できる未配線経路を増やさず、既存の構築条件とテナント境界を保つ。

## 要約

`spec-diff -- main` に規範仕様の変更はない。
IdManagement が求める削除とクライアントの存在照会へ契約を絞り、既存の保存先を直接渡す。
本人データの同意は既存の応答型を要求側の表示値へ移し、組み立て地点で一度だけ変換する。
User の削除順、途中失敗からの再実行、Agent のテナント境界と束縛の判断、HTTP の項目とイベントは保つ。
新しい中継層とパッケージを増やさず、IdManagement から OAuth2 への本番 import を除き、観測されなくなった 13 件の循環を台帳から削除した。
テストを変更しない構造変更なので、既存テストと配置の検査を使い、RED と変更耐性の検査は適用しない。
使用量の再集計、削除の原子性、Agent の資格情報とエクスポート項目の変更は対象外に残す。
