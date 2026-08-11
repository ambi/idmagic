# Seeding Requirements

> This Markdown file is the normative, language-independent home for product requirements. Models and API contracts live in the adjacent TypeSpec source.

## Requirements

### REQ-SEEDING-001: 環境別の明示profileが選択される
- Actor: SeedOperator
- Given: SeedOperator が environment と profile を明示している
- Then: SeedOperator が SeedData を dry_run で呼ぶ
- Then: planner は environment policy に許可された manifest だけを選ぶ
- Then: 応答は redacted な SeedPlan を返し永続状態を変更しない

### REQ-SEEDING-002: 明示manifestまたはprofile既定manifestが選択される
- Actor: SeedOperator
- Given: SeedOperator が environment と profile を明示している
- Then: SeedOperator が明示 manifest path を指定して SeedData を呼ぶ
- Then: loader は指定 path の manifest と contained include を strict decode する
- Then: planner は manifest の typed desired resource を計画する
- Alternative (manifest path が未指定である): loader は profile ごとの repository default manifest を選ぶ

### REQ-SEEDING-003: manifestとrequestのprofile不一致は拒否される
- Actor: SeedOperator
- Given: SeedOperator が request と異なる profile の manifest を指定している
- Then: SeedOperator が SeedData を呼ぶ
- Then: SeedData は secret 解決と書き込みの前に SeedRejectedError で拒否する

### REQ-SEEDING-004: 不正manifestは書き込み前に拒否される
- Actor: SeedOperator
- Given: manifest に未知 key、重複 logical key、未対応 schema version、include cycle、または root 外 path がある
- Then: SeedOperator が SeedData を呼ぶ
- Then: loader は secret 解決と書き込みの前に SeedRejectedError で拒否する
- Then: 診断には秘密値を含めない

### REQ-SEEDING-005: productionではenv secret providerを拒否する
- Actor: SeedOperator
- Given: environment が production である
- Given: manifest が env secret provider を参照している
- Then: SeedOperator が SeedData を dry_run または apply で呼ぶ
- Then: SeedData は secret 解決と書き込みの前に SeedRejectedError で拒否する
- Then: 永続状態は変更されない

### REQ-SEEDING-006: 同一seedの再適用はno-opになる
- Actor: SeedOperator
- Given: 同じ manifest、generator seed、secret version で seed が apply 済みである
- Then: SeedOperator が同じ SeedRequest を再度 apply する
- Then: SeedPlan の全 operation は noop である
- Then: password history と created_at / updated_at は変更されない

### REQ-SEEDING-007: productionでdemoまたはperformance profileは拒否される
- Actor: SeedOperator
- Given: environment が production である
- Then: SeedOperator が development または performance profile を指定する
- Then: SeedData は書き込み前に SeedRejectedError で拒否する
- Then: 既知のデモ資格情報は作成されない

### REQ-SEEDING-008: production bootstrapには明示redirect URIが必要である
- Actor: SeedOperator
- Given: environment が production である
- Given: profile が bootstrap である
- Then: SeedOperator が first_party_redirect_uris を指定して SeedData を apply する
- Then: first-party client は指定 URI だけを redirect URI として持つ
- Alternative (redirect URI が未指定、localhost、または http URI である): SeedData は書き込み前に SeedRejectedError で拒否する

### REQ-SEEDING-009: manual driftは上書きせずconflictになる
- Actor: SeedOperator
- Given: seed 管理対象の logical key が手動変更されている
- Then: SeedOperator が対応する profile を apply する
- Then: SeedData は SeedConflictError を返す
- Then: 手動変更は維持される

### REQ-SEEDING-010: 部分失敗後に同じrequestを再実行すると収束する
- Actor: SeedOperator
- Given: SeedData の apply が一部の operation を完了した後に失敗している
- Then: SeedOperator が同じ SeedRequest を再度 apply する
- Then: 完了済み logical key は no-op と判定される
- Then: 未完了の logical key だけが適用され、重複なく目的状態へ収束する

### REQ-SEEDING-011: SeedData
seed を同一の決定的 planner で計画し適用する内部運用 interface。apply は各 record context の published command surface だけを呼び、直接 SQL fixture で不変条件を迂回しない。
- Precondition: manifest_schema_supported(input.request)
- Precondition: manifest_profile_matches_request(input.request)
- Precondition: manifest_paths_are_local_and_contained(input.request)
- Precondition: input.request.environment in ['staging', 'production'] implies manifest_secret_providers(input.request) == ['file']
- Postcondition: input.request.mode == 'dry_run' implies persistent_state_unchanged()
- Postcondition: reapply_same_request_is_noop(input.request)
- Postcondition: input.request.environment == 'production' && input.request.profile == 'bootstrap' implies production_safe_redirect_uris(input.request.first_party_redirect_uris)
- Postcondition: seed_plan_and_diagnostics_exclude_secret_values(output.plan)

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| SeedProfile | seed の内容と生成規則を表す明示選択の profile。bootstrap は稼働に必要な最小データだけ、development/test は既知のサンプル、performance は非機密の合成データを表す。環境名から暗黙に選ばない。 | seed profile |
| SeedPlan | 現在状態と profile manifest を比較して作る、redacted な変更計画。dry-run と apply は同じ計画規則を使う。 | seed plan |
| SeedDrift | seed 管理対象の logical key に対する現在値が manifest の canonical value と異なる状態。既定では手動変更を上書きせず conflict として停止する。 | drift |
| BootstrapSeed | first-party client など、サービス稼働に必要な最小データ。デモ資格情報やサンプル tenant data を含まない。 |  |
| SeedOperator | 明示した profile を plan または apply するローカル運用者または自動化主体。 |  |
| SeedManifest | seed resource と決定的 generator を宣言する versioned YAML desired state。DB fixture ではなく各 record context の公開 command surface への入力となる。 | seed manifest |
| SeedSecretReference | manifest が秘密値そのものの代わりに保持する provider、locator、version の組。解決値は plan、log、error に現れない。 | secret reference |
