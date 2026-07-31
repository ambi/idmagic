---
status: accepted
authors: [tn]
created_at: 2026-07-25
supersedes: [ADR-128]
---

# ADR-141: 上流権威からの identity 取り込みを単一 `Sourcing` context に束ね、source 別 feature slice で拡張する (分類軸は runtime 形状でなく権威と source binding)

## コンテキスト

[[ADR-128-extract-provisioning-context-and-transactional-delivery-capture]] は outbound
provisioning を独立 `Provisioning` context として切り出し、inbound
(外部 → idmagic の identity 取り込み) の taxonomy 設計を
[[wi-258-inbound-integration-taxonomy]] へ申し送った。現状 inbound 相当の実装と計画は散在する:

- `Scim` context (`backend/scim/`) = SCIM 2.0 server。外部 IdP が我々の API を叩く。
- `backend/idmanagement/user/usecases/user_import.go` = 管理者の CSV user import。
- 計画: [[wi-95-ldap-ad-user-federation]] (LDAP/AD)、
  [[wi-30-inbound-federation-and-identity-broker]] (broker / JIT)、
  [[wi-156-orphan-account-discovery-and-reconciliation]] (orphan 照合)。

ADR-128 §コンテキスト (2) が仮に置いた 3 分類「受動 server 型 / upload 型 / 能動 pull 型」は
**起動契機 (runtime 形状) による分類**だが、実装と計画を精査すると境界の軸として成立しない。

- **transport の向きは context の軸にならない。** wi-95 の Directory Connector は AD と同じ閉域網に
  置かれ、idmagic へ **outbound-only + mTLS** で接続して差分を送る。つまり idmagic 側の runtime は
  SCIM server と同じ「受信 API」であり、「能動 pull」ではない。逆に将来 HR SaaS の API を idmagic から
  直接叩く source が来ればそれは能動 dial になる。誰が TCP を張るかは source 内部の transport 詳細で
  あって、ubiquitous language を割る線ではない。
- **管理者 CSV upload は「取り込み」ではなく管理者の一括編集である。** 外部の権威システムが存在せず、
  connection・credential・correlation identity・cursor・drift のいずれも無い。認可は既存の
  `UserDirectory` 権限で、`/api/admin/users/imports` は
  [[ADR-140-admin-data-csv-export]] 決定 1 が CSV export と**対称**に設計した admin data transfer の
  片翼であり、[[wi-284-improve-csv-import-export]] が group / member import でその対を拡張する。
  SCIM server と同じ語で束ねると、権威が反転したものが 1 context に同居する。
- **inbound と呼ばれていたもののうち 2 つは、そもそも別の族である。** wi-30 は自身の Plan で
  broker を Authentication 所有の upstream connection / linking capability と規定している
  ([[ADR-064-protocol-contexts-and-application-catalog]])。起動契機は対話的ログインで、対象は
  「今ログインしてきた 1 人」であり、集団の inventory も drift も持たない。wi-156 は我々が書いた
  **mirror 側 (downstream target)** の台帳を読んで照合するもので、上流権威の取り込みではない。

したがって「inbound」という方向語を context 名にも分類軸にも使えない。分類軸そのものを引き直す。

## 決定

### 1. 分類軸は「上流の外部権威 + durable な source binding」

context の線引きは方向 (inbound/outbound) でも runtime 形状 (server/upload/pull) でもなく、
**外部システムがある identity 集団に対して権威を持ち、その関係が持続的な binding として存在するか**
で行う。この条件を満たすものを **sourced identity** と呼び、次を伴う:

- source binding (どの外部システムか、credential / enrollment、有効・無効)
- 外部不変 ID ↔ 内部 principal の correlation link
- 取り込み実行の観測単位 (run、差分 cursor、失敗と再開)
- 属性 mapping と、削除・無効化を上流権威に従わせる規則
- 上流と内部の drift

この条件を満たすものだけを 1 つの context に束ねる。満たさないものは「取り込みに見えるが別の族」
として明示的に他 context へ帰属させる (決定 5、決定 6)。

### 2. 目標構造は単一 bounded context `Sourcing` + source 別 feature slice

sourced identity の全メンバーを単一 bounded context `Sourcing`
(`spec/contexts/sourcing.yaml`) が所有し、source ごとに feature slice を並べる。現在および計画上の
メンバーは:

| source slice | 実体 | 状態 |
| --- | --- | --- |
| `scim` | SCIM 2.0 server (外部 IdP が push) | 実装済み (現 `Scim` context)。移設は [[wi-259-rename-scim-inbound-server-context]] |
| `directory` | 閉域 Directory Connector 経由の LDAP/AD | 計画 ([[wi-95-ldap-ad-user-federation]]) |
| (将来) `feed` | scheduled file feed (HR CSV / SFTP 等) | 未計画。決定 5 の受け皿 |

`Provisioning` (outbound = 下流 target への push) と対称の capability 名として `Sourcing` を採る。
IAM 慣習でも上流権威は "source" 語で呼ばれ (SailPoint の Sources、Okta の profile source)、既存の
命名様式 (`Provisioning` / `Seeding` / `ClaimMapping`) にも一致する。語彙は
`Sourcing`: source → correlation link → ingestion、`Provisioning`: connection → remote link →
delivery と、方向に応じて対称に反転する。

### 3. source 非依存コアは今作らない (thin root)

ADR-128 決定 3 が確立した非対称——**構造 (context / slice) は作り替えが高コストゆえ前もって決め、
共有コードは未成熟な二表現の早期結合が有害で後追い抽出が安価ゆえ on-demand で切り出す**——を
そのまま適用する。ADR-128 の `Provisioning` が「fat な protocol 非依存コア + protocol slice」に
なったのは、outbound では配送エンジン・delivery・remote link という共通振る舞いが**既に 1 実装
分存在していた**からである。inbound では事情が違う: 現 `Scim` は source binding も correlation
link も cursor も持たず、受け取った mutation を IdManagement へ直に適用するだけで、決定 1 が列挙した
共通機構の実体は**まだ 1 つも存在しない**。今コアを書けば、それは wi-95 の要求だけから逆算した
投機的抽象になる。

よって `Sourcing` の root は当面 thin (context の facade と composition だけ) とし、
2 つ目の source (wi-95) が着地した時点で実際に重複した部分——correlation link、ingestion run /
cursor、mapping、削除権威規則——を on-demand で root へ引き上げる。この抽出は wi-95 の作業範囲とし、
本 ADR は抽出の**タイミングと責務の置き場所**だけを決める。

### 4. Go 配置は最初から `backend/sourcing/<source>/` の slice にする

`backend/sourcing/scim/{domain,ports,usecases,handlers_http,db_memory,db_postgres}` の形で、
slice を最初から 1 段掘る。`module.go` は context ルートに 1 つ据える (ADR-128 決定 2 と同型)。
adapter package 名は `<役割>_<技術詳細>` ([[ADR-133-flat-wikipedia-architecture]])。

これは [[ADR-130-idmanagement-feature-vertical-slice]] の「単一 feature の context に feature slice を
導入しない (stutter を作らない)」に対する**明示的な例外**である。例外の根拠は、slice 軸 (source) が
決定 1・2 で確定済みであり、2 つ目のメンバーが計画済み WI (wi-95) として実在するため。flat root で
始めると、wi-259 で SCIM 実装全体を動かした後、wi-95 で**同じ実装を再び全部動かす**ことになり、
ADR-128 が避けたはずの「構造の後追い作り替え」を自ら招く。`sourcing/scim` は stutter でもない。

### 5. 管理者 CSV import は IdManagement に残す (ADR-128 §影響 の申し送りを覆す)

ADR-128 §コンテキスト (2)(b) / §影響 は CSV user import を「適所でない」とし
[[wi-260-relocate-csv-user-import-to-inbound]] へ申し送ったが、決定 1 の軸で判定すると
**CSV upload は sourced identity ではない**ため、本 ADR はこの申し送りを覆す。

- 権威は idmagic 自身 (操作者は tenant 管理者) であり、外部 SoR・source binding・correlation
  identity・cursor・drift のいずれも無い。上流権威に削除を従わせる規則も存在しない。
- ADR-140 決定 1 が export を `/users/imports` と対称な per-type 表面として設計し、wi-284 が
  group / member import でその対称性を拡張する。import だけを別 context へ移すと、対称に設計された
  片翼だけが context 境界を越え、CSV codec・列 allowlist・formula injection エスケープ・job 配線が
  2 context に重複する。
- wi-284 が予定する属性拡充 (`attributes` の TenantUserAttributeSchema 検証、`required_actions`、
  dynamic group の CEL rule 式) は IdManagement domain の不変条件そのものであり、別 context から
  深く手を伸ばすことになる。

将来 **scheduled file feed** (HR システムが nightly に SFTP へ置く CSV 等) を実装する場合、それは
source binding と cursor と削除権威を持つので `Sourcing` の source slice (`feed`) に置く。その時点で
CSV parse / 列マッピングに実際の重複が出たら、共有コードは rule of three で on-demand に抽出する
(決定 3 と同じ判断)。**管理者の一回性 upload と scheduled feed は別物**であり、前者が後者に発展した時に
初めて Sourcing へ移す。

### 6. 「取り込みに見えるが別の族」の帰属を明記する

| もの | 帰属 | 理由 |
| --- | --- | --- |
| login-time federation / JIT provisioning / account linking ([[wi-30-inbound-federation-and-identity-broker]]) | `Authentication` (ADR-064、wi-30 Plan) | 起動契機は対話的ログイン、対象は認証してきた 1 人、集団 inventory と drift を持たない |
| downstream target の account 台帳照合 / orphan 検出 ([[wi-156-orphan-account-discovery-and-reconciliation]]) | `Application` / `Provisioning` 側 (wi-156 Scope) | 読む相手は我々が書いた mirror であり上流権威ではない |
| 管理者 CSV import / export | `IdManagement` (決定 5) | 権威は idmagic 自身、source binding 無し |

wi-30 の「外部 subject ↔ 内部 principal の link」は Sourcing の correlation link と概念的に重なるが、
共有は決定 3 と同じ理由で行わない (2 実装が出てから on-demand)。

### 7. Sourcing の依存方向は IdGovernance と同型にする

`Sourcing` は `Tenancy` / `IdManagement` / `ApiTokens` / `Jobs` を published language で参照する。
逆向き (IdManagement が Sourcing を知る) は作らない。取り込みは IdManagement の published な冪等
command surface を経由して principal を作成・更新・無効化する。record-of-truth 側に source 固有の
関心を持ち込まない、という [[ADR-117-extract-identity-governance-context]] と同じ形である。

ADR-128 §コンテキスト (2) の申し送り「client として外部 API を能動駆動する machinery (connection
登録・credential・スケジューリング・retry・remote 相関) は outbound push と active-pull inbound で
共通化し得る」については、**今は共通化しない**と判断する。決定 1 の通り現在の inbound メンバーは
いずれも idmagic 側が受信側であり、能動 dial する source は存在しない (wi-95 も connector からの
outbound-only)。再検討トリガは「idmagic 自身が外部へ dial する source が実装される」時点とし、その
WI が `Provisioning` 側の connection / credential / scheduling 実装との重複を測ってから抽出する。

## 却下した代替案

- **`Scim` を protocol context として維持し (rename のみ)、wi-95 を別 context にする**: 既存の
  protocol context 群 (OAuth2 / Saml / WsFederation) と揃う利点はあるが、SCIM server と directory
  source は決定 1 の共通機構 (correlation link、削除権威、drift、run) を同じ意味で必要とし、語彙も
  同じである。分けると同じ語彙が 2 context に二重化し、将来の feed source も置き場所を持たない。
  wire protocol の差は slice 軸で表せる (決定 2・4)。
- **runtime 形状ごとに 3 context (受動 server / upload / 能動 pull)**: 分類軸そのものが成立しない
  (§コンテキスト)。upload は sourced identity でなく、wi-95 は受信 API であって pull ではない。
  実メンバー 1 つずつの context が並ぶだけになる。
- **`Inbound` 命名**: ADR-128 §コンテキスト (2) 自身が「inbound 一語では 3 shape を判別できない」と
  批判した名前で、決定 6 で 3 つの「inbound に見えるが別の族」を他 context へ出した後は特に誤解を招く。
  方向語は境界の理由 (権威) を語らない。
- **`IdentitySync` 命名**: 機構名なので責務が同期に固定され、決定 1 の軸 (権威) が名前から消える。
  SCIM server の push も「同期」と読めるが、将来 source が同期以外の形を取る余地を狭める。
- **`Provisioning` を `Outbound` に改名して `Inbound` / `Outbound` の対称にする**: ADR-128 決定 1 が
  capability 名を採った判断を、方向語という劣った軸へ後退させることになる。対称性は
  `Sourcing` / `Provisioning` という capability 名の対で足りる。
- **CSV user import を upload slice として `Sourcing` へ移す (ADR-128 の当初申し送り)**: 決定 5 の通り
  権威条件を満たさず、ADR-140 が設計した export との対称性を壊し、wi-284 の属性拡充で IdManagement
  domain へ深く手を伸ばす。
- **source 非依存コアを今から作る**: 決定 3 の通り、共通機構の実体がまだ 1 つも存在せず、wi-95 の
  要求だけから逆算した投機的抽象になる。ADR-128 決定 3 の非対称に反する。
- **flat root で始め wi-95 到来時に slice 化する**: 決定 4 の通り、SCIM 実装全体を 2 度動かすことに
  なり、構造の後追い作り替えという最も高いコストを自ら招く。

## 影響

- `spec/scl.yaml` `context_map`: `Sourcing` エントリを新設 (`depends_on` =
  Tenancy / IdManagement / ApiTokens / Jobs、すべて `via: published_language`。決定 7)。
- `spec/contexts/sourcing.yaml` (新規): 決定 1 の語彙を glossary として scaffold する
  (`IdentitySource` / `SourceCorrelation` / `Ingestion` / `IngestionRun` / `SourceCursor` /
  `SourceDrift`)。`not_to_confuse_with` で `ProvisioningConnection`・`RemoteResourceLink`・管理者
  CSV import・JIT provisioning・target 側 reconciliation との混同を封じる。models / interfaces /
  scenarios の実体は wi-259 (SCIM 移設) と wi-95 (directory) が入れる。
- `spec/contexts/scim.yaml` / `context_map` の `Scim`: 本 ADR では触らない。`Sourcing` の `scim`
  slice への移設と canonical ref namespace 変更 (`Scim/...` → `Sourcing/...`)、`ScimUserRef` /
  `ScimGroupRef` の rename 判断は wi-259 が行う。移設完了時に `Scim` エントリは無くなる。
- `ARCHITECTURE.md`: `contexts` 台帳に `Sourcing` を追加し、Context Map と Structural Decisions に
  目標構造 (決定 2・4) と thin root 方針 (決定 3) を記す。`backend/sourcing/**` の module 台帳は
  実体が入る wi-259 で追加する。
- `backend/idmanagement/user/usecases/user_import.go` / `admin_user_import_handler.go`: 移設しない
  (決定 5)。wi-260 は前提が消滅するため不要になる (受け皿は将来の scheduled feed に限定)。
- 下流 WI への指示: wi-259 = `backend/scim` → `backend/sourcing/scim` の slice 移設 + SCL context
  rename、wi-95 = `sourcing/directory` slice + 決定 3 のコア抽出、wi-30 = Authentication 所有のまま、
  wi-156 = target 側のまま (決定 6)。
- ADR-128 との関係: 本 ADR は ADR-128 の**申し送り部分だけ**を上書きする (§コンテキスト (2)(b) の
  「CSV import は適所でない」と、inbound taxonomy 未決という前提)。ADR-128 の決定 1〜5
  (`Provisioning` 境界・protocol slice・共有 kernel 不作成・same-Tx capture) は有効のまま。
