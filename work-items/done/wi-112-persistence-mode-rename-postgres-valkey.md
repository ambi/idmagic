---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-07-04
---

# PERSISTENCE モードの値を postgres から postgres_valkey に改名し durable/volatile 双方のバックエンドを名前へ反映する

## Motivation
`PERSISTENCE=postgres` は名前上は Postgres 単独を連想させるが、実装 (assemblePostgres)
は durable を Postgres、volatile を Valkey に振り分け両方のバックエンドを組み立てる。
`VALKEY_URL` が必須であり、複数レプリカ運用では Valkey 上の共有ストアが前提
(SharedEphemeralStateHA) であるにもかかわらず、その事実がモード名から読み取れない。
`internal/bootstrap/postgres.go` というファイル名も片側のみを名乗っていた。

値を `postgres_valkey` とし durable/volatile 双方を名前へ含めることで、実態と名前の
乖離を解消する。加えて将来 durable バックエンドが増えた場合 (例 `mysql_valkey`) にも
組み合わせを一意に区別できる。`distributed` 等の抽象名は durable 側の別を表せないため
採らず、durable/volatile を独立軸に割る案は現状 memory / postgres+valkey の2択しか
実在せず過剰なため見送った (ADR-016 に記録)。

## Scope
- **scl**: spec/contexts/system.yaml の objectives SharedEphemeralStateHA description 内の `PERSISTENCE=postgres` / `postgres ランタイム` を `postgres_valkey` に更新する。派生 HTML/JSON を再生成する。
- **go**: internal/bootstrap の PERSISTENCE スイッチ値・エラーメッセージ・コメントを `postgres_valkey` に更新し、assemblePostgres を assemblePostgresValkey へ、ファイル postgres.go を postgres_valkey.go へ改名する。durable アダプタパッケージ internal/shared/adapters/persistence/postgres は Postgres 実装そのものであり据え置く。
- **docs**: README (env 表・HA 節)、ARCHITECTURE、deploy/docker/docker-compose.dev.yaml、decisions/ADR-016 (選択子の定義と改名理由) を更新する。

## Out of Scope
- durable/volatile を独立の環境変数へ分離する2軸化 (現状 2 択のため将来課題)。
- 過去の履歴記録 (ADR-024 / ADR-077、work-items/done/*) 内の `PERSISTENCE=postgres` 表記の遡及修正。その時点の記録として据え置く。

## Verification
- just yaml-check-scl / just scl-render
- go build ./... / go vet ./internal/bootstrap/... / go test ./internal/bootstrap/...

## Risk Notes
環境変数の値の破壊的リネームであり、既存の起動構成 (docker-compose・運用の env) が
旧値 `postgres` のままだと default の `memory` に落ちる。リポジトリ内の全参照を
更新済みだが、外部の運用 manifest がある場合は追随が必要。

## Completion
- **Completed At**: 2026-07-04
- **Summary**:
  system.yaml の SharedEphemeralStateHA description を `postgres_valkey` に更新し、
  just scl-render で spec/idmagic.html を再生成した (JSON schema/openapi はこの prose を
  含まないため差分なし、drift なし)。internal/bootstrap では assemble の switch 値・
  default エラー文言・構造体コメントを `postgres_valkey` に更新し、assemblePostgres を
  assemblePostgresValkey にリネーム、postgres.go を git mv で postgres_valkey.go へ改名、
  `PERSISTENCE=postgres_valkey requires ...` のエラー文言も更新した。durable アダプタ
  パッケージ persistence/postgres は Postgres 実装そのものとして据え置いた。README の
  env 表・HA 節、ARCHITECTURE の runtime 選択と assemble 関数名、docker-compose.dev.yaml の
  PERSISTENCE 値、ADR-016 の選択子定義 (memory | postgres_valkey) を更新し、ADR-016 には
  postgres 単独名が Valkey 必須の事実を隠す点・将来 mysql_valkey とも区別できる点という
  改名理由を追記した。過去の履歴記録は据え置いた。
- **Verification Results**:
  - just yaml-check-scl (全 11 ファイル OK)
  - just scl-render (idmagic.html 等を再生成)
  - go build ./... (成功)、go vet ./internal/bootstrap/... (指摘なし)
  - go test ./internal/bootstrap/... (ok)
  - リポジトリ内に旧値 `PERSISTENCE=postgres` / `assemblePostgres` の live 参照が
    残っていないことを rg で確認 (dev.sh・e2e fixtures は memory のため対象外)。
- **Affected Guarantees State**: SharedEphemeralStateHA の内容 (共有対象・fail_closed 縮退) は
  不変。モードを指す識別子のみ改名しており、保証義務の実体に変更はない。
