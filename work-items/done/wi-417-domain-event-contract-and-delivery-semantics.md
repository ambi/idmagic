---
depends_on: []
status: completed
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-08-27
priority: p1
change_kind: docs
evidence_policy: risk-based-v2
initial_context:
  specification:
    - docs/README.md
    - docs/structure.md
    - docs/deployment.md
    - docs/capacity.md
    - docs/threat-model.md
    - docs/contexts/system/decisions.md
    - docs/contexts/jobs/decisions.md
    - docs/contexts/audit/internals.md
    - docs/contexts/audit/scenarios.md#REQ-AUDIT-002
    - docs/contexts/provisioning/scenarios.md#REQ-PROVISIONING-003
  typespec:
    - IdMagic.Contract.AdminAuditEventResponse
    - IdMagic.Contract.AuditEventQuery
    - IdMagic.Contract.AuditEventSearchOptionsResponse
  source:
    - backend/shared/spec/events.go
    - backend/cmd/internal/bootstrap/audit_event_record.go
    - backend/cmd/internal/bootstrap/securitynotification.go
    - backend/audit/usecases/audit_search_extractor.go
    - backend/authentication/securitynotification/domain/catalog.go
    - backend/authentication/securitynotification/usecases/dispatch.go
    - backend/idmanagement/user/ports/provisioning_notify.go
    - backend/idgovernance/usecases/user_mutation_committer.go
    - backend/provisioning/ports/capture.go
  tests:
    - tools/check/src
  stop_before_reading: [frontend, infra, load]
affected_spec:
  - { path: docs/contexts/audit/scenarios.md, requirement: REQ-AUDIT-002 }
  - { path: docs/contexts/provisioning/scenarios.md, requirement: REQ-PROVISIONING-003 }
---

# ドメインイベントの契約と配信意味論を仕様として持つ

## Motivation

`docs/README.md` の Context Map は `IdManagement --Events: lifecycle--> IdGovernance`、`IdManagement --Events: lifecycle--> Provisioning`、`Authentication --Events: audit facts--> Audit` など、公開イベントによる関係を 6 本宣言している。ところが、その関係の契約はどこにも仕様化されていない。

モデルと API の契約は TypeSpec が持ち、`mise run check-api-compat` がリリース済み OpenAPI ベースラインとの互換性を検査する。イベントにはその対応物が無い。具象のドメインイベントは `backend/<context>/domain/events.go` にあり、`backend/shared/spec/events.go` がエンベロープのインターフェースとワイヤ表現への変換を持つ、と `docs/structure.md` が述べるだけで、イベントのスキーマ、フィールドの意味、バージョニング、互換性の保証はコードが唯一の正本になっている。Context Map が公開関係として宣言しているものの契約が仕様に無い、という状態である。

正本が失われた痕跡も残っている。`spec/contexts/audit/models.tsp` は `AdminAuditEventResponse.payload` を「the payload in the events section of the specification」と説明し、`AuditEventQuery.type` と `AuditEventSearchOptionsResponse.event_types` も同じ「events section」を指す。この節は SCL の廃止 (`1b7b2cef`、2026-08-11) とともに消えており、**外部に公開している API の説明が、存在しない仕様の節を指したまま残っている**。

配信の意味論も同じ穴を持つ。`docs/contexts/jobs/decisions.md` は「配送は少なくとも 1 回とし、冪等性はハンドラーの責務とする。ちょうど 1 回を基盤側で保証しようとすると、プロセスの異常終了と完了報告の競合を塞ぎきれないからである」という優れた判断を持つが、これは Jobs Context のローカルな判断であって、Context 間の `Events` 関係全体に効く規範ではない。順序、重複、遅延、再生について、製品全体としてどこまで保証しどこから保証しないかを述べた場所が無い。

状態変更とイベント発行の原子性も未文書化である。リポジトリ全体を検索しても Outbox に相当する記述は 1 件も出てこない。監査を製品の重要な性質として扱い、`docs/standards.md` が `GDPR-PROCESSING-RECORDS` として監査記録の保持を約束している以上、「状態は変わったがイベントは出なかった」が起こりうるのかどうかは、全体規範として答えられていなければならない。`docs/capacity.md` の縮退順序も「監査イベントを黙って破棄することは縮退手段に含めない」と述べているが、破棄されないことを何が保証するかは書かれていない。

容量の面でも、`docs/capacity.md` はジョブレーンごとの実行枠の算出まではあるが、消費者の遅れ、再生、イベントの保持期間を扱っていない。

状態変更とイベント発行の原子性については、[docs/threat-model.md](../../docs/threat-model.md) の THREAT-074（状態は変わったのに、対応する監査イベントが残らない）が同じ欠落を脅威の側から指している。

最後に、`SPECIFICATION_FORMAT.md` は「意図的な非採用は、それを再開する条件とともに `decisions.md` に書く」と定めている。イベント駆動やストリーミングの基盤を採らないという判断は、この製品にとって明らかに意図的な非採用であるにもかかわらず、どこにも書かれていない。自らの規則が自らに適用されていない。

## Scope

- **イベント契約の正本**：ドメインイベントのエンベロープと、Context 間で公開されるイベントのスキーマの正本をどこに置くかを決め、置く。
- **公開と内部の区別**：Context Map の `Events` の辺に対応する公開イベントと、Context 内部でしか使わないイベントを区別する。公開したものだけが契約の対象になる。
- **バージョニングと互換性**：公開イベントの後方互換の規則を定め、`check-api-compat` に相当する検査を持つかどうかを判断する。
- **配信意味論の全体規範**：順序、重複、遅延、消失、再生について、製品全体として何を保証し何を保証しないかを `docs/` の該当ファイルに書く。
- **発行の原子性**：状態変更とイベント発行が同一トランザクションで捕捉されるかを調査し、現状を記述する。捕捉されていない経路は負債として記録する。
- **容量と保持**：消費者の遅れ、再生、イベントの保持期間を `docs/capacity.md` の枠組みへ加える。
- **非採用の明示**：専用のイベント基盤やストリーミング処理を採らない判断を、再開の条件とともに書く。

## Out of Scope

- Outbox パターンの実装。現状の記述と負債の把握までを担い、原子性が保証されていない経路の是正は経路ごとに別の work item が扱う。
- 専用のキューミドルウェアやイベントストアの導入。`docs/contexts/jobs/decisions.md` の「運用するデータストアを増やさない」という判断は本件では覆さない。
- イベントの外部公開。SharedSignals による SET の配信は既に独自の契約（RFC 8417）を持ち、本件の対象外である。
- Audit の Read Model の再設計。
- **214 件すべてのイベント payload の TypeSpec 宣言**。Design で述べるとおり、契約の対象は公開項目の語彙であって Context 内部の項目ではない。内部の項目まで宣言すると、検査されない写しが 214 個できる。
- **`auditEventCategoryTypes` の網羅性**。監査の種別絞り込みに載るイベント種別は 85 件で、発行される 214 件を覆っていない。`type` による絞り込みと検索属性は全件に効くので誤りではないが、管理画面の選択肢が全件を提示しないという別の問題であり、Audit Context の work item が扱う。
- **THREAT-074 の status の変更**。本 work item は保証されていないことを記述するのであって、保証を作るのではない。応える制御が無い状態は変わらないので `planned` のままとする。

## Design

### 調査結果：`Events` の 6 本の辺が実際に何であるか

Context Map の `Events` の辺は、まったく異なる 2 つの機構で実現されており、**どちらもイベントバスではない**。

**`Events: lifecycle` の 2 本（IdManagement → IdGovernance、IdManagement → Provisioning）は、ドメインイベントを 1 件も運んでいない。** 実体は IdManagement が所有し下流の Context が実装する同期の Go のポート呼び出しである。IdGovernance 側は `userports.UserMutationCommitter` で、`igports.UserWorkflowCapture` が配線されていれば User の保存と派生する LifecycleWorkflow run を同一トランザクションで確定する。Provisioning 側は `userports.ProvisioningNotifier` で、呼び出し元のコミット後に別トランザクションで捕捉し、失敗しても記録するだけで呼び出し元へは伝播しない。`backend/provisioning/ports/capture.go` はこの残存する隙間を「2 つのコミットの間で異常終了すると捕捉が回復不能に失われる」と自ら書いている。どちらの辺も、上流が語彙を公開し下流が実装する形なので `Events` ではなく `OHS/PL` である。

**`Events: audit facts` は、`backend/cmd/internal/bootstrap` の `NewEmitFunc` が組み立てる閉包 1 つに集約されている。** 閉包は呼び出し元の goroutine の中で、呼び出し元自身の書き込みが済んだ後に、`context.Background()` から作り直した 2 秒のコンテキストで走り、`EventSink` への出力、セキュリティ通知のディスパッチ、`AuditEventRepo.Append` を順に行う。どの段の失敗もログに落とすだけで、呼び出し元の操作は成功したまま返る。キューも再試行も Outbox も無く、再配送の経路も無い。

Context Map はこの辺を 4 本しか宣言していないが、実際に閉包へ到達する Context は Application、Authentication、Authorization、DataKeys、IdGovernance、IdManagement、Jobs、OAuth2、Provisioning、Saml、SharedSignals、SigningKeys、Tenancy、WorkloadIdentity、WsFederation に及ぶ。**宣言が 4 本、実態が 15 本**である。

宣言に無い第 3 の消費者も見つかった。`backend/authentication/securitynotification` は監査の射影とまったく同じワイヤ表現を購読し、イベント種別名（カタログの 14 種別）と payload の項目名（`tenantId`、`userId`、`targetUserId`、`userAgent`、`reason`）で読む。Context Map にこの辺は無い。

### 契約の対象は、封筒と「公開項目の語彙」である

発行されるイベント型は 214 件ある。この全件を TypeSpec へ写すのは採らない。`SPECIFICATION_FORMAT.md` が禁じる「検査されない第二の写し」を 214 個作ることになり、`AdminAuditEventResponse.payload` が `Record<unknown>` として意図的に不透明であるという既存の契約とも噛み合わないからである。

代わりに、**公開イベントを「他の Context がイベント種別名または payload の項目名で読むもの」と定義する**。この定義で切ると、契約は次の 2 つになる。

- **エンベロープ**：`spec.MarshalDomainEvent` が必ず載せる `type` と `occurredAt`。監査の記録、管理 API の応答、セキュリティ通知のディスパッチがすべてこの形の上で動く。
- **公開項目の語彙**：`backend/audit/usecases/audit_search_extractor.go` が検索属性へ写す項目と、セキュリティ通知が宛先と条件の判定に読む項目。`tenantId`、`userId`、`actorUserId`、`targetUserId`、`agentId`、`clientId`、`sessionId`、`transactionId`、`correlationId`、`requestId`、`username`、`ip`、`workflowId`、`runId`、`stepIndex`、`actorChain`、`delegationDepth`、`actorChainDepth`、`delegationMode`、`userAgent`、`reason` の 21 項目である。

この語彙こそが、イベントにおける Published Language である。ある Context がこの名前で項目を載せれば、その値は監査の検索軸として引かれ、通知の宛先として使われる。名前を変えれば、コンパイルは通ったまま検索軸が黙って空になる。契約と呼ぶに値するのはここであって、Context 内部でしか読まれない項目ではない。

置き場所は `spec/contexts/system/models.tsp` とする。配信点 `bootstrap.NewEmitFunc` を所有するのは System Context（`docs/README.md` の索引が `backend/cmd/internal/bootstrap` を System に割り当てている）だからである。Audit へ置く案は却下した。Audit はこの語彙の消費者であって、全 Context が満たすべき形を Audit が所有すると、供給側が下流の Context の契約に従う倒立が起きる。

却下した第二案（各 Context の `internals.md` に散文で書く）と第三案（AsyncAPI や JSON Schema の独立した木を持つ）は、元の記述のとおり採らない。前者は差分も検査も取れず、後者は `docs/README.md` の「機械が食う契約だけが `spec/` にあり、2 つの木は `contexts/<context>/` で対応する」という構成を崩す。

### 互換性の検査は、OpenAPI 相当ではなく語彙の一致を見る

`check-api-compat` に相当する検査、すなわちリリース済みベースラインとの後方互換の判定は持たない。公開イベントは HTTP の契約と違い、消費者がこのリポジトリの中にしかいない。外部の消費者がいない契約にリリースベースラインを敷いても、守る相手がいない。

代わりに置くのは `mise run check-event-contract` である。TypeSpec が宣言する公開項目の語彙と、Go の消費者が実際に読んでいる項目名が一致することを検査する。宣言だけを置いて検査を置かなければ、それこそが `SPECIFICATION_FORMAT.md` の言う「検査されない写し」になる。検査があってはじめて、TypeSpec の宣言が飾りではなく荷重を持つ。

検査の抽出点は 2 つに固定する。`backend/audit/usecases/audit_search_extractor.go` の `payloadString` / `payloadStrings` / `payloadNumberString` の第 2 引数と、`backend/authentication/securitynotification/` の `stringField(payload, …)`、`payload["…"]`、`RecipientField:` の値である。抽出は純粋な関数（ソース文字列 → 項目名の集合）として書き、ファイルの読み取りは呼び出し側に置く。

### 配信意味論と実現方法は、別々のファイルが持つ

**実現方法**（どの辺がどの機構で実現されているか、公開項目の語彙が何であるか）は `docs/structure.md` に置く。この文書は依存の向き、層、アーキテクチャスタイルを持ち、既に `domain/events.go` の配置規則と Anti-Corruption Layer の置き方を持っている。辺の実体はその隣にある事実である。

**保証**（何を保証し何を保証しないか）は `docs/deployment.md` に置く。この文書は実行単位と可用性を持ち、「Availability and shared state」で既に、何が永続で何が一時か、再試行が安全か、障害時にフェイルクローズするかを述べている。イベントが失われうるかどうかは同じ種類の問いである。

**非採用の判断**は `docs/contexts/system/decisions.md` に置く。配信点を所有する Context の判断だからである。

### 効果の境界

本 work item は製品コードを変更しない。追加する検査は次の形を取る。時刻、乱数、識別子生成、設定、永続化はいずれも関与しない。

```text
collectDeclaredEventFields(tspSource: string): Set<string>
collectConsumedEventFields(goSources: Map<path, string>): Set<string>
diffEventFieldVocabulary(declared, consumed): { missing: string[]; undeclared: string[] }
```

ファイルシステムの読み取りは `check-event-contract.ts` の入口だけが行い、上の 3 つは文字列と集合の上の純粋な計算とする。

## Plan

1. TypeSpec に封筒と公開項目の語彙を宣言し、`spec/contexts/audit/models.tsp` の「events section」への 3 箇所の参照をこの宣言へ向け直す。
2. `docs/structure.md` に辺の実現方法と公開項目の語彙を書く。
3. `docs/README.md` の Context Map を実態へ合わせる。`Events: lifecycle` の 2 本を `OHS/PL` へ改め、`Events: audit facts` を実際に発行する Context へ広げ、セキュリティ通知の辺を足す。
4. `docs/deployment.md` に配信意味論の全体規範を書く。守られている保証だけを保証として書き、原子性の欠落は保証しないと明記する。
5. `docs/capacity.md` に消費者の遅れ、再生、保持を Planning assumption として加える。
6. `docs/contexts/system/decisions.md` にストリーミング基盤の非採用判断を、再開の条件とともに書く。
7. `check-event-contract` の RED を観測してから実装する。

## Tasks

- [x] T001 [Baseline] `Events` の 6 本の辺の実現方法、運ばれるイベント、発行の原子性を経路ごとに調査する。結果は Design に記録した。
- [x] T002 [Spec] 封筒と公開項目の語彙を `spec/contexts/system/models.tsp` へ宣言し、`spec/contexts/audit/models.tsp` の宙に浮いた参照を向け直す。`mise run check-spec` と `mise run check-api-compat`。
- [x] T003 [Spec] `docs/structure.md` に辺の実現方法、公開項目の語彙、後方互換の規則を書く。
- [x] T004 [Spec] `docs/README.md` の Context Map を実態へ合わせる。
- [x] T005 [Spec] `docs/deployment.md` に配信意味論の全体規範と、原子性が保証されていない経路を書く。
- [x] T006 [Spec] `docs/capacity.md` に消費者の遅れ、再生、保持を Planning assumption として加える。
- [x] T007 [Spec] `docs/contexts/system/decisions.md` に非採用判断を書く。
- [x] T008 [Acceptance] `mise run check-event-contract` が語彙の不一致で落ちることを、宣言を入れる前に観測する。
- [x] T009 [App] Unit RED を確認してから `diffEventFieldVocabulary` と 2 つの抽出関数を実装し、`mise run check-event-contract` を GREEN にする。
- [x] T010 [Verify] `mise run verify`。

## Verification

- `mise run check-spec` が新しいモデルを含めて成功する。
- `mise run check-api-compat` が、HTTP 操作から到達しないモデルの追加によって互換性判定を変えない。
- `mise run check-event-contract` が、宣言された語彙と Go の読み取り点の一致を判定する。
- `mise run check-links` が、`docs/` に足したリンクとアンカーで落ちない。
- `mise run verify`

## Risk Notes

イベントを TypeSpec へ宣言すると、生成された OpenAPI に意図せず現れるおそれがある。HTTP 操作から到達しないモデルの扱いを T002 で確認し、公開契約の表面積を広げないことを `check-api-compat` で確かめる。

原子性の調査で「保証されていない経路」が多数見つかった場合、本 work item の範囲では是正できない。記述と負債の記録にとどめ、是正は経路ごとに切り出す。ここで是正まで抱え込むと、仕様が書かれないまま実装が長引き、最も価値のある成果（契約が仕様に載ること）が最後まで出てこない。

配信意味論を「保証したいこと」として書いてしまうと、Context Map と同じく実装と乖離した宣言になる。現在守られている保証だけを書き、それ以外は保証しないと明記する。

Context Map の辺を書き換える影響は本 work item に閉じない。[[wi-414-boundary-fitness-functions]] は Context Map を機械可読な正本として読み、宣言に無い辺を持つ import を拒否する検査を作る。`Events: lifecycle` を `OHS/PL` へ改める判断と、`Events: audit facts` を 15 本へ広げる判断は、その検査が許す import の集合を直接決める。結論はここに固定し、wi-414 はこれを前提にする。

`reversibility: irreversible` とするのは、公開項目の語彙が名前の集合として固定されるためである。ある項目名を語彙から外すと、その名前で検索していた監査の絞り込みが黙って空を返すようになり、既に記録されたイベントは書き戻せない。追記のみの監査記録に対しては、この決定を後から取り消せない。

## Completion

- **Completed At**: 2026-08-29
- **Summary**:
  `mise run spec-diff` が出したのは `spec/contexts/system/models.tsp` への `DomainEventEnvelope` と `DomainEventPayload` の追加 2 件だけである。規範シナリオ、状態遷移の行、規範 ID はいずれも増減していない。ドメインイベントの契約が、今回はじめて機械が読める形で仕様に載った。
  この出力が成果物の全体を映していないことは、そのまま [[wi-442-spec-diff-does-not-see-standards-rows]] の Motivation の裏付けになる。`docs/structure.md` の Cross-context events、`docs/deployment.md` の Domain event delivery、`docs/contexts/system/decisions.md` の非採用判断、`docs/README.md` の Context Map の 3 辺の書き換えは、いずれも `spec-diff` の視野に入らない。
- **Acceptance RED Evidence**:
  - **Test**: `bun run check/src/check-event-contract.ts`（`mise run check-event-contract` として登録する前の直接実行）。
  - **Requirement**: N/A: 製品の振る舞いを変えない。REQ-AUDIT-002 と REQ-PROVISIONING-003 は本 work item が記述した配信経路が満たしている既存の規範であり、変更の対象ではない。
  - **Observed Failure**: `DomainEventPayload does not declare the consumed field …` が 22 行、終了コード 1。宣言側が 0 項目、読み取り側が 22 項目という差である。
  - **Detection Reason**: 検査は宣言と読み取りの両側を独立に集めて突き合わせる。TypeSpec に何も書かない実装、書いたが読み取り点と名前が食い違う実装、読み取り点を増やして宣言を忘れた実装のいずれも、この差として現れる。宣言だけを置いて満足する（もっともらしい間違い）が通らないことが、この境界を選んだ理由である。
  - **Alternate check**: 上記が観測可能な境界そのものなので、代替は要さない。
- **Unit RED Evidence**:
  - **Test**: `bun test check/src/event-contract.test.ts`
  - **Requirement**: N/A: 検査器の内部関数であり、対応する製品の規範シナリオを持たない。
  - **Observed Failure**: `error: Cannot find module './event-contract.ts'`、0 pass / 1 fail。
  - **Detection Reason**: 3 つの関数を、ファイルシステムに触れない文字列と集合の上の計算として切り出したので、宣言の走査、読み取り点の抽出、差の算出をそれぞれ独立に落とせる。検査器全体の成否だけを見ていると、どの段が壊れたのかが分からない。
- **Change-Resistance Results**:
  変更した純粋ロジックは `tools/check/src/event-contract.ts` の 3 関数だけである。その全体に対して系統的に変異を入れ、単体テストと実データに対する検査の両方で殺せるかを見た。等価変異は無く、9 件すべてを殺した。

  | # | Mutation | Result |
  | --- | --- | --- |
  | M1 | 閉じ括弧での打ち切りを `continue` にする | KILLED（当初 SURVIVED。次のモデルを飲み込む fixture を足して塞いだ） |
  | M2 | 省略可能を表す `?` を property の正規表現から落とす | KILLED |
  | M3 | 対象を `DomainEventPayload` から任意のモデルへ広げる | KILLED |
  | M4 | `RecipientField` の抽出点を無効化する | KILLED |
  | M5 | `payload["…"]` の抽出点を無効化する | KILLED |
  | M6 | `stringField(payload, …)` を任意の 2 引数呼び出しへ緩める | KILLED |
  | M7 | `missing` を反対側の集合から計算する | KILLED |
  | M8 | 出力の整列をやめる | KILLED |
  | M9 | `undeclared` を常に空にする | KILLED |

  **手法の限界**：変異は抽出と差分の計算だけを対象にしており、抽出点の一覧そのもの（`check-event-contract.ts` の `CONSUMERS`）は変異させていない。Context をまたいで payload を読む第 4 の箇所が将来加わっても、この検査は気付かない。一覧が実態を覆っているかを守るのは、`docs/structure.md` の Cross-context events を読んだ人間のレビューであって、この検査ではない。
- **Verification Results**:
  - `mise run verify` - passed
