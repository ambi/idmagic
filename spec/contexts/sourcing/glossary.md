# Sourcing Glossary

| Term | Definition | Aliases |
|---|---|---|
| IdentitySource | あるアイデンティティ集団について IdMagic の外部で権威を持つシステムと、その関係を表すテナント単位の関連付け。取り込み元の種類、資格情報と登録情報、有効・無効、属性の対応付け、削除・無効化を上流の権威にどこまで従わせるかをまとめる。取り込み元との関連付けを持たない経路は本 Context の対象外とする。 | source, アイデンティティソース, 取り込み元 |
| SourceCorrelation | 外部の取り込み元が持つ不変 ID と、IdMagic 内部のプリンシパル（User / Group）を結ぶ関連付け。取り込みを冪等にし、名前や属性が変わっても同一性を失わないための基準となる。`scim` 機能では ScimUserRef / ScimGroupRef が該当する。 | correlation link, external identity link, 相関 |
| Ingestion | IdentitySource が権威を持つ状態を IdMagic 内部のプリンシパルへ反映する処理。作成、更新、無効化、削除は IdManagement が公開する冪等なコマンドインターフェースを通じて適用し、記録元へ取り込み元固有の関心事を持ち込まない。 | ingest, 取り込み |
| IngestionRun | 1 回の取り込みを観測する単位。対象の取り込み元、開始と終了、適用件数、失敗、再開位置を持つ。実行は Jobs の永続ジョブに委ね、失敗後に再開できる粒度で観測する。`scim` 機能は外部 IdP からのリクエスト単位で適用するため IngestionRun を持たず、`directory` 以降の機能で実体を追加する。 | ingestion run, 取り込み実行 |
| SourceCursor | 前回の取り込みがどこまで進んだかを表す、取り込み元ごとの位置情報。差分取り込みと再同期の境界を決める。完全再同期は、カーソルを破棄して全件を読み直す操作として定義する。 | sync cursor, カーソル |
| SourceDrift | 上流の取り込み元が持つ正しい状態と IdMagic 内部の状態との乖離。取り込みの失敗、取り込み元での直接変更、相関情報の欠落などで生じ、検出と是正は取り込み元ごとの権威規則に従う。 | drift, 乖離 |
| ScimClient | テナント単位の Bearer トークンを提示して SCIM プロビジョニング API を呼び出す外部エージェント。`scim` 機能において IdentitySource を駆動する側である。 |  |
