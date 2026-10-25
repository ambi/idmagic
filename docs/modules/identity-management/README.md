# IdManagement

## 責務と境界

テナント単位の `User`、`Group`、`Agent` の台帳を扱う。
`User` は人間、`Group` は `User` を束ねてロールをまとめて付与する単位、`Agent` は非人間の行為者である。
このモジュールは、それぞれのプロフィール、ロール、ライフサイクル、管理 API、セルフサービス API、CSV による一括の取り込みと書き出しを扱う。
User と Group の属性スキーマは、定義の型、組み込み属性、管理 HTTP、更新の検証、動的 Group の参照保護と保存をこのモジュールに集める。
公開 API の仕様、規範 ID とイベント名は、既存の [Tenancy の属性スキーマ](../tenancy/features/attribute-schema.md)を使う。

| 扱わないもの | 担当 |
| --- | --- |
| 資格情報の検証、多要素認証、ログインセッション | `Authentication` |
| OAuth2 クライアントの資格情報とトークンの発行 | `OAuth2` |
| ライフサイクルワークフローによる自動化 | `IdGovernance`。このモジュールは、そこから呼ばれる冪等なコマンドの側であり、誰がいつ変更したかの記録はこのモジュールに残る |
| 外部の権威からの取り込み（SCIM など） | `Sourcing` |
| 外部の宛先への送り出し | `Provisioning` |

User と Group の詳細の SCIM 由来は、公開 `user/ports` の `SCIMUserLookup` と `group/ports` の `SCIMGroupLookup` で照会する。
Sourcing が対応レコードから有無へ変換し、組み立て地点が渡すため、IdManagement は取り込み元の保存型に依存しない。

SCIM の User と Group の作成、属性変更、所属変更と削除は `SourceCommands` で受ける。
取り込み元の投影済みの属性だけを受け取り、既存の作成と管理用ユースケースの検証、使用量、イベントと通知を共有する。
User の作成と属性変更では、管理操作と同じワークフロー捕捉も使う。
上流の属性は値の保存前に書き込み権威を検査し、ローカル専用属性の追加、変更と削除を拒否する。
JIT も同じ検査を使い、取り込み元の保存型を属性の分類へ持ち込まない。
SCIM の対応表と一緒に確定するトランザクションは組み立て地点が束縛する。

資格情報は保存型を受け取らず、初回のパスワードの検証とハッシュ化、履歴の書き込み、完全削除と端末失効の操作を要求する。
完全削除の認証情報の区間は Authentication の一操作に委ね、前後のセッションと OAuth2 の削除、Tombstone の進捗とイベントの編成をこのモジュールに残す。
CSV の履歴は行と同じ Tx の writer を使い、通常の作成の履歴と区別して組み立てる。
本人のメール変更の再認証と認証エラーの表示も所有者の操作へ委ね、認証の時間窓と sentinel を複製しない。

プロトコル記録の削除は `user/ports` の `UserRecordsPurger`、Agent の束縛先の存在照会は `agent/ports` の `ClientLookup` を要求し、OAuth2 の保存先が直接満たす。
本人データの同意は、OAuth2 の本人向け照会から `user/ports` の表示値へ組み立て地点で変換して受け取る。
同意の保存型とクライアントの設定を受け取らず、User の削除順と Agent の束縛の判断はこのモジュールに残す。

## 主な流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| 管理 API とセルフサービス API | HTTP の要求 | `api` が、ハンドラーからユースケースを同期的に呼ぶ | 各機能の設計 |
| CSV のプレビューと適用 | 管理者の要求で `Jobs` のジョブを作る | `worker` | [CSV の転送の実現方式](features/csv-transfer.md#実現方式) |
| データエクスポート | 管理者の要求で、種類 `data_export` のジョブを作る | `worker` | [データエクスポート](features/data-export.md) |
| ドメインイベントの発行 | 状態を変えた操作 | ユースケースが発行し、監査と下流へ渡す | [イベントと監査の記録](observability.md) |
| CSV の成果物の削除 | 外部のスケジューラーが Batch の `retention-sweep` を起動する | `batch` が、作成から 30 日を過ぎた成果物を消す | [CSV の転送](features/csv-transfer.md#成果物の保持) |
| 猶予期間を過ぎた User の完全削除 | 外部のスケジューラーが Batch の `retention-sweep` を起動する | `batch` が、各テナントで猶予期間を過ぎた削除予約の User と、途中で失敗した完全削除を完全削除する | [ユーザーの設計](features/user.md#信頼性) |
| ほかのモジュールからの User の停止 | ライフサイクルワークフローの手順、SCIM の取り込み | `worker` と `api` が、User のユースケースを呼ぶ | [ほかのモジュールからの User の停止](#ほかのモジュールからの-user-の停止) |

### ほかのモジュールからの User の停止

User の無効化、再有効化、削除の予約は、管理 API の外からも起きる。
どの経路でも User のユースケースを通し、User の状態を直接保存させない。
止めることに伴うイベント、記憶済みの端末の失効、動的グループの再評価、下流への通知、所有する Agent の無効化、削除予約中の User の拒否を、管理 API と揃えるためである。

| 経路 | 呼ぶ側のポート | 操作 | 監査イベントの actor |
| --- | --- | --- | --- |
| ライフサイクルワークフローの `disable_user` と `enable_user` | `IdGovernance` の `UserLifecycle` | 無効化と再有効化 | `lifecycle-workflow` |
| SCIM の `active` の切り替え | `Sourcing` の `UserLifecycle` | 無効化と再有効化 | `scim` |
| SCIM の `DELETE` | `Sourcing` の `UserLifecycle` | 削除の予約 | `scim` |

`user` スライスの `UserLifecycleCommands` が両方のポートを満たし、組み立ての地点（`worker` と `api` の起動）が依存を管理 API と同じ集合で組み立てて渡す。
`UserLifecycleCommands` は、User の変更からライフサイクルワークフローの実行を作る仕組みを渡さない。
ワークフローの手順による User の変更から、さらにワークフローの実行が連鎖しないようにするためである。
削除予約中の User への無効化と再有効化は、ワークフローの手順にとっては変更なしとして扱う。

## 設計の要点

| 方針 | 理由 |
| --- | --- |
| `User`、`Group`、`Agent` をそれぞれ機能スライスに分ける | 一つの Aggregate の仕組みを一つのディレクトリの下で読めるようにする。詳細は[重要な設計判断](decisions.md)の「User、Group、Agent を機能ごとの縦割りのスライスで構成する」 |
| CSV のインポートを、プレビューと適用の二つの非同期ジョブにする | 適用の前に全行の検証結果を管理者に見せ、適用ではプレビューしたペイロードだけを実行する。詳細は[CSV の転送の実現方式](features/csv-transfer.md#実現方式) |
| User の削除を Tombstone にする | 追記専用の記録が参照する `sub` を壊さない。詳細は[判断](decisions.md#user-の削除を物理削除ではなく-tombstone-で行う) |
| 他のモジュールとの協調をポートの注入で行う | このモジュールから下流のモジュールへの依存を作らない |

## 機能一覧

| 機能 | 内容 |
| --- | --- |
| [ユーザー](features/user.md) | 管理者による `User` の作成、一覧、更新、無効化、削除、属性 |
| [アカウントのセルフサービス](features/account.md) | 本人によるプロフィール、メールアドレス、データエクスポートの操作 |
| [エージェント](features/agent.md) | `Agent` の登録、資格情報のバインド、ライフサイクル |
| [グループ](features/group.md) | `Group` の作成と更新、連絡先とカスタム属性、手動のメンバーシップ、実効ロール |
| [属性スキーマ](../tenancy/features/attribute-schema.md) | User と Group の定義の管理。公開 API の分類は Tenancy に維持し、管理 HTTP をモジュールのルート、更新を各機能スライスに置く |
| [動的グループ](features/dynamic-group.md) | CEL の規則の保存、プレビュー、全件の再評価による所属 |
| [CSV の転送](features/csv-transfer.md) | User と Group の CSV が共有する、見出し、行、セルの読み書きと転送の上限 |
| [ユーザー CSV](features/user-csv.md) | User の CSV のインポートとエクスポート |
| [グループ CSV](features/group-csv.md) | Group と、一つの Group のメンバーシップの CSV のインポートとエクスポート |
| [データエクスポート](features/data-export.md) | 非同期のエクスポートのジョブの開始、一覧、ダウンロード、取り消し、保持期限 |
| [管理 API の認可](features/admin-access.md) | 管理者のセッションと API アクセストークンによる管理 API の呼び出しを認める条件 |
| [ロール](features/roles.md) | ロールの正規化と、予約ロール `system_admin` を割り当てられる対象 |

データエクスポートとロールは機能スライスを持たず、モジュールのルートが CSV の転送ポリシーと解析器、成果物ストアとともに実装する。

## 文書

| 設計領域 | 文書 | 内容 |
| --- | --- | --- |
| データ設計 | [IdManagement のデータ](data.md#集約) | Aggregate とその関係、永続化、保持と削除、排他と一意性、テナントの属性スキーマ |
| セキュリティ設計 | [管理 API の認可](features/admin-access.md#概要)、[認可設計](../../design/security/authorization.md#主体の種類) | 管理 API の認可は機能 管理 API の認可 が、システム全体の認可は認可設計が扱う |
| 信頼性設計 | [IdManagement の信頼性](reliability.md#idmanagement-の信頼性) | 障害の症状、原因、直し方 |
| 性能設計 | [IdManagement の性能](performance.md#実効ポリシーを超えるエクスポート) | 実効ポリシーを超えるエクスポート |
| オブザーバビリティ設計 | [IdManagement のオブザーバビリティ](observability.md#経路) | ドメインイベント、監査の記録、下流への通知の経路と、確定との関係 |
| 検証設計 | [性能](performance.md#実効ポリシーを超えるエクスポート) | 品質要件の確かめ方は性能に置き、それ以外はシステムの検証設計に従う |

| 文書 | 内容 |
| --- | --- |
| [IdManagement の用語集](glossary.md) | このモジュールでの語義 |
| [IdManagement の品質要件](quality.md) | このモジュールに割り当てた品質要件 |
| [設計判断](decisions.md) | 代替案を比べた重要な判断 |
| [リスク](risks.md) | 既知の欠陥、技術的負債、受容したリスク |
