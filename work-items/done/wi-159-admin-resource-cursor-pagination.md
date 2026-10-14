---
status: completed
archived_from: 61f5d0fe18c17eadc4a23f1b9048389f098c55ca
---

# 管理対象リソース一覧をカーソルページネーション対応にする

RFC 8288 `Link` ヘッダによる keyset pagination (ADR-158) を対象 10 interface すべてに Go 層
(handler/usecase/repository、memory + Postgres) で実装した。SCL コア言語に response header
宣言語彙 (`bindings.http.response_headers`) を追加。監査イベント/ユーザー/グループ/エージェント/
アプリケーションの5画面 (Plan が優先指定した順) を「さらに読み込む」UI へ移行し、共有基盤
(`requestPage`/`usePaginatedList`/`LoadMoreButton`) を新設した。全 10 interface に handler レベルの
cursor 境界テスト (Link header 有無、改ざん cursor 拒否、他テナント cursor 拒否、次ページ重複なし)
を追加した。
この WI の範囲内で意図的に対応しなかったこと (すべて Tasks に開示済み、隠れた欠落ではない):
Consents・ApplicationAssignments・ProvisioningDeliveries の一覧画面、および sign-in-policy 画面の
application テーブルは「さらに読み込む」UI へ未移行 (`limit=200` capped の単一ページ表示のまま)。
「戻る」の URL 履歴 stack 方式は未実装 (前方の「さらに読み込む」のみ、Design が明示的に許容する
代替方式)。一覧 API を picker/dropdown/id→name lookup に流用している約12箇所は `limit=200` に
capped したのみ (全件検索可能な picker は wi-161 の範囲)。Dashboard の
`activeUserCount`/`disabledUserCount`/`grantedConsentCount` と Users 一覧画面自身の概要タイルは
breakdown 用の summary endpoint が無いため `limit=200` capped な近似値のまま
(`userCount`/`clientCount` のみ正確な tenant usage summary に切り替えた)。e2e は既存 smoke test に
Groups の到達性チェックを追加したのみで、「さらに読み込む」クリック→次ページ表示の専用シナリオ
(50件超のデータ seed が必要) は未追加。いずれも該当リソースが 200 件を超える大規模テナントでのみ
顕在化する制約で、別 follow-up とする。

全文は `git show 61f5d0fe18c17eadc4a23f1b9048389f098c55ca:<このファイルのパス>` で読む。
