---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-16
priority: p2
depends_on: [wi-583-normalize-design-document-terminology]
change_kind: docs
evidence_policy: risk-based-v3
spec_impact:
  kind: none
  reason: "シークレットと鍵の現行設計を書き下すだけで、鍵のライフサイクル、暗号方式、保存形式は変えない。"
documentation_impact:
  level: none
  reason: "変わるのは開発者と運用者が読む設計文書だけで、利用者が観測する振る舞い、API、設定キーはどれも変わらない。"
  references: []
initial_context:
  specification:
    - docs/domain/signing-keys/scenarios.feature.md#REQ-SIGNINGKEYS-001
    - docs/domain/signing-keys/scenarios.feature.md#REQ-SIGNINGKEYS-008
    - docs/domain/signing-keys/scenarios.feature.md#REQ-SIGNINGKEYS-012
    - docs/domain/oauth2/scenarios.feature.md#REQ-OAUTH2-036
    - docs/domain/oauth2/scenarios.feature.md#REQ-OAUTH2-037
  typespec: []
  source:
    - DOCUMENTATION_GUIDE.md
    - docs/design/security/secrets.md
    - docs/design/security/README.md
    - docs/design/security/threat-model.md
    - docs/design/data/database.md
    - docs/design/reliability/recovery.md
    - docs/design/infrastructure/platform.md
    - docs/design/infrastructure/network.md
    - docs/architecture/deployment.md
    - docs/runbooks/backup-restore-dr.md
    - docs/domain/data-keys/README.md
    - docs/domain/data-keys/states.md
    - docs/domain/data-keys/internals.md
    - docs/domain/data-keys/decisions.md
    - docs/domain/signing-keys/README.md
    - docs/domain/signing-keys/states.md
    - docs/domain/signing-keys/internals.md
    - docs/domain/signing-keys/decisions.md
    - backend/cmd/internal/bootstrap/config.go
    - backend/cmd/internal/bootstrap/sharedconfig.go
    - backend/cmd/internal/bootstrap/apiconfig.go
    - backend/cmd/internal/bootstrap/datakeys.go
    - backend/cmd/internal/bootstrap/keystore.go
    - backend/cmd/internal/bootstrap/postgres.go
    - backend/cmd/idmagic-batch/main.go
    - backend/oauth2/db_postgres/clients.sql
    - backend/authentication/totp/db_postgres/reencrypt.sql
    - backend/authentication/federation/db_postgres/federation.sql
    - infra/k8s/base/api.yaml
    - infra/k8s/base/batch-cronjobs.yaml
    - infra/docker/docker-compose.dev.yaml
  tests: []
  stop_before_reading:
    - frontend/
    - work-items/done/
    - spec/
---

# シークレットと鍵の設計に、種類ごとのライフサイクルと運用上の判断を書く

## Motivation

[シークレットと鍵の設計](../../docs/design/security/secrets.md)は 19 行である。
五種類の区分と所有者を表で示し、ライフサイクルを二段落で述べる。

区分表は良い骨格だが、各区分について次が書かれていない。

- どこで生成し、誰が生成の正しさを保証するか。
- ローテーションの契機と間隔。定期なのか、事象駆動なのか。
- 新旧を同時に受理する期間を何から決めるか。「署名または暗号化済みデータの寿命より短い切替にしない」とは書かれているが、その寿命がどこで決まるかは書かれていない。
- 失効と緊急ローテーションの手順の設計。漏洩が疑われたときに、何を、どの順で、どれだけの時間で入れ替えられるか。
- 廃棄したあとに何が読めなくなるか。

起動時シークレットの供給も、「配備基盤の Secret 管理」「Git へ保存せず、必要な実行単位だけへ注入する」の二文だけである。
デプロイプロファイルごとに何がその役を担うか（Docker Compose の環境変数、Kubernetes の Secret、Secret Manager と Cloud KMS）は書かれていない。

一方で、鍵を失ったときの結果は重い。
[バックアップと復旧の runbook](../../docs/runbooks/backup-restore-dr.md) は、OpenBao の Transit 鍵を失うと全テナントのラップ済み DEK を恒久的にアンラップできなくなり、`DestroyTenantDataKey` が意図して行う暗号学的消去と同じ結果が事故として起きると述べている。
この重さに対して、設計文書の側の記述が釣り合っていない。

## Scope

- [シークレットと鍵の設計](../../docs/design/security/secrets.md)に、区分ごとの生成、保存、配布、利用、ローテーション、失効、廃棄、復旧を書く。
- 起動時シークレットの供給を、デプロイプロファイルごとに書く。
- 緊急ローテーションで何がどの順に入れ替わるかの設計を書く。
- 鍵の喪失が何を意味するかを、復旧設計とバックアップの runbook から到達できる形で書く。

## Out of Scope

- 個別 Context の鍵ライフサイクルの詳細。[Data Keys](../../docs/domain/data-keys/README.md) と [Signing Keys](../../docs/domain/signing-keys/README.md) が正本である。
- 暗号方式の選択と実装。エンベロープ暗号の設計は [データベース設計](../../docs/design/data/database.md#可逆な秘密情報のエンベロープ暗号)が持つ。
- 鍵のローテーション運用の実装。[[wi-307-datakeys-rotation-lifecycle-operations]] が扱う。
- FIPS 準拠の暗号プロファイル。[[wi-296-fips-approved-cryptography-profile]] が扱う。
- 実作業の手順。[runbook](../../docs/runbooks/) が持つ。
- 脅威と制御の対応。[脅威モデル](../../docs/design/security/threat-model.md)が持つ。

## Design

現在の区分表を維持し、区分ごとに段階を追う形へ広げる。
段階は `DOCUMENTATION_GUIDE.md` §4.12 が挙げる八つ（生成、保存、配布、利用、ローテーション、失効、廃棄、復旧）に固定する。
区分を行、段階を列にすると表が大きくなりすぎるので、区分ごとの小節に段階の表を置く。

各段階で書くのは次の三つである。

| 列 | 内容 |
| --- | --- |
| 誰が | プロダクト、デプロイ基盤、外部鍵サービス、運用者のどれが担うか |
| 何を保証するか | その段階を通ったことで何が成り立つか |
| 失敗したとき | 到達できない、検証できない、期限切れのときの振る舞い。フェイルクローズかどうか |

「失敗したとき」の列を必ず置く。
鍵の設計で後から効くのは、正常時の流れではなく、鍵に到達できないときに何が起きるかである。
現在の文書は「アンラップに失敗したらフェイルクローズ」を `database.md` の側に書いており、鍵の設計から読めない。

新旧を同時に受理する期間は、**その鍵で保護されたものの最長寿命**から決まる。
署名鍵なら発行済みトークンの最長有効期限、暗号鍵なら再暗号化が完了するまでの時間、TLS 証明書なら更新間隔である。
この対応を表にする。値そのものは各 Context の仕様と設定が持ち、ここには「何から決まるか」を書く。

緊急ローテーションは、定期ローテーションとは別の設計として書く。
決めるのは、入れ替えの順序（何を先に止めるか）、入れ替え中に何が拒否されるか、入れ替え後に無効化すべき既存のセッションとトークン、そして入れ替えに要する時間の設計上の想定である。
これは手順ではなく設計であり、手順は runbook が持つ。

起動時シークレットの供給は、[[wi-584-ground-deployment-design-in-reference-profiles]] が定めるデプロイプロファイルごとに書く。
同じ work item がプラットフォーム設計の側にも供給元を書くため、片方が正本でもう片方が参照になる。
正本はこの文書とする。シークレットの扱いはデプロイ先の都合ではなくセキュリティ設計の判断であり、プロファイルが増えても判断は変わらないためである。

鍵の喪失については、失った鍵ごとに何が読めなくなり、何が残るかを書く。
マスター鍵、テナントの DEK、署名鍵の秘密鍵で結果が違う。
[復旧設計](../../docs/design/reliability/recovery.md)が鍵素材への到達性を復元の対象として扱っていることと、この節を相互に参照させる。

## Plan

1. 区分ごとに、現行の実装と Context 仕様から八段階を読み取る。
2. 区分ごとの小節へ段階の表を置き、「失敗したとき」を埋める。
3. 同時受理期間が何から決まるかの対応を書く。
4. 緊急ローテーションの設計を書く。
5. 起動時シークレットの供給をプロファイルごとに書き、プラットフォーム設計から参照させる。
6. 鍵の喪失の結果を書き、復旧設計と相互参照させる。

## Tasks

- [x] T001 [Design] 区分ごとの八段階を、実装と Context 仕様から読み取る。
- [x] T002 [Docs] 区分ごとの段階の表を書き、失敗時の振る舞いを埋める。
- [x] T003 [Docs] 同時受理期間の決まり方と、緊急ローテーションの設計を書く。
- [x] T004 [Docs] 起動時シークレットの供給をプロファイルごとに書く。
- [x] T005 [Docs] 鍵の喪失の結果を書き、復旧設計と相互参照させる。
- [x] T006 [Verify] リンク、仕様、全体検証を通す。

## Verification

- `mise run check-links`
- `mise run check-spec`
- `mise run verify`

## Risk Notes

ローテーションの間隔や有効期限の値を書くと、各 Context の仕様と設定リファレンスに続く三つ目の正本になる。この文書が持つのは「何から決まるか」であって値ではない。

緊急ローテーションの設計に、鍵の保存場所、到達手順、具体的な操作対象を書くと、攻撃者に有利な情報を仕様へ置くことになる。書くのは何が起こりうるかと何を守るかであり、どうやるかは runbook が持つ（`DOCUMENTATION_GUIDE.md` §4.11）。

「配備基盤が所有する」と書いた段階は、実際には誰も所有していないことがある。所有者の欄に基盤と書くときは、デプロイプロファイルのどれで実際にその機能が構成されているかを確かめ、構成されていないものは未確定と書く。

## Completion

- **Completed At**: 2026-09-19
- **Summary**:
  `mise run spec-diff` は main に対して規範仕様の差分を報告せず、`spec_impact: none` と一致する。
  意味の差はシークレットと鍵の設計文書にある。

  五つの区分それぞれが、生成、保存、配布、利用、ローテーション、失効、廃棄、復旧の八段階を表として持つようになった。
  各段階は担い手、その段階を通って成立すること、失敗したときの動作の三つを持ち、「失敗時の動作」の列は空にしない。
  これにより、鍵へ到達できないときの振る舞いが鍵の設計から読めるようになった。
  これまで「アンラップに失敗したらフェイルクローズ」はデータベース設計の側にしかなく、`provider_healthy` を観測するだけで発行を止めるのは OAuth2 側だという分担も SigningKeys の内部設計にしかなかった。

  実装を読み直したことで、設計として書かれていなかった現在状態が三つ表に現れた。
  DataKeys の DEK は、ローテーションを本番で起動する経路を持たない。管理 API も定期実行のジョブもなく、構成ファイルが宣言する CronJob は保持期限による削除と署名鍵のライフサイクルの二つだけである。
  署名鍵の緊急ローテーション後にテナントの全セッションとリフレッシュトークンをまとめて失効させる経路はなく、利用者ごとの失効を繰り返すしかない。
  クライアント資格情報も、クライアント単位でまとめて失効させる経路を持たない。
  どれも「未実装」と行に書き、是正を担う work item は名指さない。

  クライアント資格情報の保存は、これまで「比較可能な表現または封筒暗号で保存する」という二択の書き方だった。
  実装は 32 バイトの乱数を base64url で表した値を SHA-256 でハッシュし、定数時間で照合している。
  利用者が選んだ値ではないため鍵導出関数による伸長を行わない、という判断を理由とともに書いた。

  新旧を同時に受理する期間は、区分ごとに何がその期間を決めるかを表にした。
  値そのものは書かず、正本の場所だけを示す。
  署名鍵とクライアント資格情報は旧を残したまま新を有効にし、データ暗号鍵は新を有効にしてから旧の参照を消していく。
  この向きの違いが、検証する側が外部にあるかどうかから来ることを書いた。

  緊急ローテーションは定期ローテーションと別の設計として独立した。
  順序を決める三つの原則と、漏洩したものごとに「先に止めるもの」「入れ替え中に拒否されるもの」「入れ替え後に無効化するもの」を表にした。
  所要時間は未確定と書いた。署名鍵と DEK のローテーションをローカルでしか動かしておらず、本番規模での実測がないためである。

  起動時シークレットの供給元は、三つのデプロイプロファイルごとに供給元、プロセスへの渡り方、保存時の保護、実装状況を持つ表になった。
  正本をこの文書へ移し、[プラットフォーム設計](../../docs/design/infrastructure/platform.md)からはプロファイルの比較表の該当行を外して、参照だけを残した。

  鍵とシークレットの喪失は、失ったものごとに読めなくなるものと残るものを書いた。
  [リカバリ設計](../../docs/design/reliability/recovery.md)は復旧の経路を持ち、この文書は喪失の意味を持つ、という分担を両側から明示して相互参照させた。
- **Acceptance RED Evidence**:
  - **Test**: `N/A: この変更は設計文書だけを変え、観測可能なプロダクトの境界を持たない。`
  - **Requirement**: N/A: spec_impact は none であり、規範シナリオも標準 id も変えていない。
  - **Observed Failure**: `mise run check-links` は変更前も変更後も GREEN であり、何も壊れていなかった。
    文書が薄いことを測る検査は存在しない。
  - **Detection Reason**: この変更を所有する検査は `mise run check-links` である。
    起動時シークレットの供給元の正本をこの文書へ移した結果、プラットフォーム設計はアンカー越しにこの文書の節を名指す。
    `platform.md` の参照先を `#起動時シークレットの供給元` から `#起動時シークレットの供給` へ、`secrets.md` の参照先を `recovery.md#暗号鍵の喪失` から `#鍵素材の喪失` へ差し替えると、`Markdown anchor does not exist` を両方とも行番号つきで報告して exit 1 になった。
    「正本がそこにある」と書いた行が、実在する節に対応していることをこの検査が固定する。
- **Unit RED Evidence**:
  - **Test**: `N/A: 変更したのは Markdown だけで、単体の境界を持つコードを変えていない。`
  - **Requirement**: N/A: 同上。
  - **Observed Failure**: `mise run check-terminology` は初稿から GREEN であった。
    採らない表記を混入させると、`docs/design/security/secrets.md:47:135: 「既定」は採らない表記。「デフォルト」 を使う。` を報告して exit 1 になった。
  - **Detection Reason**: 用語検査はこの変更の単体の境界にあたる。
    文書全体ではなく行と列の単位で判定するため、書き換えた三文書のうち一文書の一行の一語だけを指せる。
- **Change-Resistance Results**:
  `risk` は `low` で、Go と TypeScript は変えていないため `mise run test-go-mutation` は適用されない。
  変異器が表現できない故障を二つ手で注入した。
  一つは、存在しない節へのリンクの差し替えである。`check-links` が `platform.md:1` と `secrets.md:201` の両方を報告し、戻すと `ok Markdown links (865 document(s))` に復帰した。
  もう一つは、採らない表記の混入である。`check-terminology` が位置つきで検出し、戻すと `ok terminology (226 document(s))` に復帰した。
  どちらも注入前後で `diff` によりファイルが元どおりであることを確かめている。
- **Verification Results**:
  - `mise run check-links` - passed（865 文書）
  - `mise run check-terminology` - passed（226 文書）
  - `mise run check-work-item-references` - passed（184 文書）
  - `mise run check-spec` - passed
  - `mise run spec-diff` - no normative specification change against main
  - `mise run check-work-items` - passed
  - `mise run verify` - passed
