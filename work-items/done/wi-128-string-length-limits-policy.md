---
status: completed
archived_from: 610341bde014725da511dbb909e29fdcfbeaa085
---

# 文字列入力値の最大文字数ルールを定義し必要な DB 制約へ反映する

文字列長の上限に、単位（Unicode コードポイント）、既定の区分、4 つの境界それぞれの役割を与え、境界ごとにばらばらだった実際の上限を 1 つに揃えた。数値そのものはほぼ据え置きで、意味上の差分は「同じ数がどこでも同じ意味を持つようになったこと」と「違反が利用者に伝わるようになったこと」の 2 点である。着手時の調査で見つかった 2 つの不具合を実際に直した。
- **マルチバイト入力が上限の 1/3 で拒否されていた**。zog の `String().Max(n)` は `len(string)`、すなわち UTF-8 バイト数を数える（ドキュメントコメントは "at most n characters long" と書いてある）。上限 100 文字と宣言したグループ名に日本語を入れると 34 文字で拒否されていた。`backend/shared/spec/length.go` に `Chars` / `CharsAtMost` を追加し、`utf8.RuneCountInString` で数えるようにしたうえで、10 ファイル・30 か所の `.Max(` を置き換えた。`authorization/domain/tuple.go` の「256 bytes」判定と、CIBA `binding_message` の usecase（コードポイント）と domain（バイト）の食い違いも同じ数え方に寄せた。
- **上限超過が HTTP 500 になっていた**。domain の `Validate()` が返す素の error が各 handler のエラー写像の `default:` を素通りし、`{"message":"Internal Server Error"}` だけが返っていた。長さ違反だけを型付きの `*spec.LengthError` にし、`support_http.ErrorHandler` が既存の `quotaExceeded` と同じ構造的インターフェースで受けて `422` / `field_length_exceeded` の Problem Details に写像する。`detail` には違反したフィールドの wire 名と上限が載る（`name: must be at most 100 characters`）。長さ以外の検証失敗は素の error のまま残し、保存済みデータの破損がサーバの不具合として扱われる余地を保った。
- このエラー写像は `backend/cmd/idmagic/server.go` だけで組み立てられていたため、ハンドラのテストは本番と別のエラー経路を通っていた。既定版を `server_http.Register` へ移し、ログとメトリクスを持つ版は起動時に cmd が差し替える形にした。分類漏れがテストから見えるようになる。
- **仕様**：`docs/SPECIFICATION.md` の Cross-cutting Concerns に `String length limits` を追加。単位、既定の 9 区分、外部の標準や固定の表示面から決まる 7 つの例外、4 つの境界の役割、上限を置かない値、違反時の 422 を記述した。Database design policy §1 の「`TEXT` + `CHECK` か `varchar(N)` のいずれか」という未決を `TEXT` + `CHECK` に確定した。
- **契約の穴埋め**：`User.name` / `given_name` / `family_name` / `email`、`Group.email`、`GroupAttributeDef.label`、CIBA `binding_message` に `@maxLength` を追加した。メールは RFC 5321 の 254。いずれも従来 Go 側だけが持っていた上限か、どこにも無かった上限である。
- **DB の最後の防壁**：`users`、`groups`、`agents`、`tenants`、`tenant_brandings`、`oauth2_clients`、`mcp_resource_servers`、`authorization_detail_types`、`workload_trust_bundles`、`agent_workload_bindings` に `CHECK (char_length(col) ...)` を追加した。通知テンプレートは DB と TypeSpec に上限があるのに Go 側の検査が無く、超過が制約違反として返って 500 になっていたので、`template.ValidateDefinition` に同じ上限を入れた。
- **UI**：`frontend/src/lib/lengthLimits.ts` に同じ区分を置き、group / user / agent の作成・編集、テナント設定、通知テンプレート、ライフサイクルワークフローの各入力欄の `maxLength` をそこから引くようにした。既存の直書きの数値も同じ表に寄せた。パスワード欄には付けない（貼り付けを黙って切り詰めるため）。
- **上限値の変更**：`UserAttributeDef.OIDCScope` の 60 を Handle 区分の 64 へ広げた 1 件のみ。他はすべて据え置きで、引き下げは行っていない。

全文は `git show 610341bde014725da511dbb909e29fdcfbeaa085:<このファイルのパス>` で読む。
