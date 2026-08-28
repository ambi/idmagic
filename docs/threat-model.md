# Threat Model

製品が何から守られているかを、脅威の側から書く。ほかの正本文書は製品が何をするかを書くので、実装された制御が正しく働いているかは照合できる。しかし**応えるべき制御がそもそも無い**とき、その欠落はどの記述とも矛盾しないので見過ごされる。この文書はその一段上を引き受ける。

[SPECIFICATION_FORMAT.md](../SPECIFICATION_FORMAT.md) は「拒否が書かれていない制御は、照合すべき記述を持たない」と述べる。同じ形が一段上にも当てはまる。**脅威が書かれていなければ、制御の欠落は何の記述とも矛盾しない。** `mise run check-security-controls` は宣言された拒否がテストされていることを確かめるが、宣言そのものが無い制御については何も言わない。

**この一覧は網羅ではない。** 現時点で識別した脅威であり、書かれていないことは検討して問題なしと判断した意味を持たない。網羅を主張しないことと、見直しの契機を定めておくことが、この文書が静的な保証と誤読されないための条件である。

**攻撃の手順は書かない。** 各行が述べるのは何が起こりうるかであって、どうやるかではない。再現手順、具体的なパラメーター、未対応の経路の詳細はここに置かない。

## How to read

分類は STRIDE を用いる。脅威の見落としを防ぐ網としては十分に粗く、この製品の関心を素直に覆うからである。攻撃木と攻撃ライブラリは、網羅性の主張が弱く維持の費用が高いので採らない。`Category` 列の値は次の 6 つに限る。

| Category | Meaning |
|---|---|
| `Spoofing` | 別の主体になりすます |
| `Tampering` | 内容または状態を改竄する |
| `Repudiation` | 行った操作を後から否認できてしまう |
| `Information disclosure` | 見えてはならないものが見える |
| `Denial of service` | 正規の利用を妨げる |
| `Elevation of privilege` | 与えられていない権限を得る |

LINDDUN は併用しない。7 分類のうちこの製品で意味を持つ Identifying と Data disclosure は STRIDE の情報漏洩と重なり、Non-compliance は [standards.md](standards.md) の GDPR 3 行が既に持つ。残る Linking、Detecting、Unawareness を脅威として立てると、主体を識別し関連付けるという IdP の職務そのものを脅威と呼ぶことになる。Non-repudiation は符号が逆で、LINDDUN では否認できないことが脅威、STRIDE では否認できることが脅威であり、監査を保証する製品としては後者を採る。個人データの観点は情報漏洩の下位として扱い、該当する行を GDPR の規範 ID へ結びつける。**再検討の条件**は、同意管理または目的制限をテナント向けの機能として提供したときである。

`THREAT-NNN` は不変であり、一度参照されたら変更しない。境界や分類を ID に埋めないのは、再分類のたびに ID が嘘になるからである。分類は列で持てば、再分類は列の変更で済む。脅威が当てはまらなくなったときは行を削除せず、`Threat` 列の末尾に後継の ID または当てはまらなくなった理由を書き、`Status` を `retired` にする。ID を消すと、その脅威を検討したという事実まで消える。

`Controls` が指すのは次の 4 つに限る。`REQ-<CONTEXT>-NNN` は規範シナリオ、大文字の識別子は [standards.md](standards.md) または各 Context の `standards.md` の規範、`<file>.md: <節>` はその文書の節、`<file>.md` は節に切り出せない文書全体の判断である。応える規範が 1 つも無い行は `—` とする。**制御の側に新しい ID 体系を作らず、計画中のものも指さない。** ここは現在存在する保護だけを並べる列であり、これから作るものを混ぜると、読み手は表を見て何が守られているかを判断できなくなる。

`runbooks/` を指す行は必ず `planned` になる。runbook は事象の最中に読む手順であって製品が従う規範ではないので、対応するテストを持たず、検査の対象にもならない。手順としての保護が実在することと、それが規範として書かれていることは別である。

`Status` は 4 値である。

| Status | Meaning |
|---|---|
| `covered` | 応える制御があり、規範 ID または正本文書の規則として名指しできる |
| `planned` | 応える制御が無いか、あっても規範として書かれていない |
| `accepted` | 制御を持たないことを意図して選んだ。理由と再検討の条件を [Accepted residual risk](#accepted-residual-risk) に書く |
| `retired` | この脅威はもう当てはまらない。行は残し、`Threat` 列に後継または理由を書く |

**`planned` は「何も無い」と「あるが規範化されていない」の両方を含む。** どちらであるかは `Controls` を見れば分かる。`—` の行には応える保護が無く、識別子が並ぶ行には規範として書かれていない部分的な保護がある。この区別を落とすと、runbook の指示やゲートウェイへの要求として実在する保護が、表の上では存在しないものとして読まれる。

**是正を担う work item はこの表に書かない。** 対応は work item の側が `THREAT-NNN` を名指しすることで持つ。向きを一方向に保つのは、正本文書が現在の状態を書く場所であり、まだ存在しない制御への前方参照を持つと、work item が完了して移動した時点で参照が黙って腐るからである。どの work item がある脅威を担っているかは `mise run spec-where THREAT-NNN` で引ける。

`planned` と `accepted` の行こそがこの文書の主な出力である。だから負債を別ファイルへ切り出さない。切り出すと、一覧を読んだ人が欠落に気づかずに読み終えてしまう。

## Trust boundaries

境界は、そこで**何を信用しないか**とともに書く。信用しないものが書かれていない境界は線でしかなく、越えるときに何を検査すべきかを言えない。

| Boundary | Not trusted here |
|---|---|
| ブラウザーとゲートウェイ | リクエストの出所。セッション Cookie の存在は、そのリクエストが製品自身の UI から出たことを証明しない |
| ゲートウェイと API プロセス | 受信ヘッダー。`X-Request-ID` と転送系ヘッダーはクライアントが制御できる |
| テナント境界 | URL、本文、クエリ文字列で与えられた `tenant_id` と `sub`。別テナントで発行されたページングのカーソル |
| 主体と権限 | ロールを持つことだけ。UI での非表示。権限が下流の呼び出しへ自動的に伝播すること |
| 資格情報と認証 | 呼び出し元が本人であるという主張。第二要素を省略できるという端末側の記憶 |
| プロトコルとトークン | 提示されたトークンが提示者のものであること。リダイレクト先。クライアントが送るメタデータ URL |
| 鍵と可逆なシークレット | マスターキー提供元への到達性。暗号文が置かれていた場所 |
| 上流の外部権威 | 外部 IdP、SCIM の取り込み元、アテステーション発行者が主張する属性と主体 |
| 下流の外部受信者 | 配信先が正しいこと。受信側が署名を検証すること |
| 永続化と非同期実行 | ジョブの `params` と `result` の中身。ハンドラーが 1 回だけ動くこと |
| 運用者と制御面 | 実行環境を取れる者。起動時設定の値 |

## Assets

| Asset | Consequence of loss | Source of truth |
|---|---|---|
| パスワード、TOTP シード、WebAuthn 資格情報、復旧コード | 任意のユーザーへのなりすまし | Authentication、可逆なものは DataKeys の DEK で保護 |
| 署名鍵 | 任意のトークン、アサーション、SET の偽造 | SigningKeys |
| DEK とマスターキー | 可逆なシークレット全体の復号 | DataKeys、マスターキーは OpenBao |
| ログインセッション | 本人としての操作の継続 | `authentication_sessions` |
| 認可コード、`request_uri`、デバイスコード、リフレッシュトークン | 認可の横取りと権限の継続的な取得 | OAuth2 |
| API アクセストークン | テナント運用権限での機械アクセス | ApiTokens |
| 個人データ（User 属性、監査イベントの PII） | 法的義務の違反と当事者への直接の害 | IdManagement、Audit |
| 監査記録 | 何が起きたかを後から言えなくなる | Audit |
| テナントの信頼設定（外部 IdP、RP、SP、`WorkloadTrustBundle`） | サインイン経路の乗っ取り | Tenancy、Authentication、Saml、WsFederation、WorkloadIdentity |
| 認可モデルと関係タプル | 細粒度認可の判定の書き換え | Authorization |
| 起動時設定 | 全体の構成の掌握 | `backend/cmd/internal/bootstrap` |

## Browser and gateway

| ID | Category | Threat | Contexts | Controls | Status |
|---|---|---|---|---|---|
| THREAT-001 | Spoofing | 攻撃者のサイトが利用者のセッション Cookie に便乗して状態変更操作を呼ぶ | System, Authentication | authorization.md: その他の境界の規則、REQ-AUTHENTICATION-005 | `covered` |
| THREAT-002 | Tampering | ログイン、同意、ポータルの画面を埋め込み、利用者の操作を別の意味に変える | System | deployment.md: Security response headers | `covered` |
| THREAT-003 | Information disclosure | 注入したスクリプトがセッションとトークンを持ち出す | System | deployment.md: Security response headers | `covered` |
| THREAT-004 | Information disclosure | 単一ページアプリがブラウザーに保持するアクセストークンが、スクリプト実行の成立時にそのまま持ち出される | System | contexts/system/decisions.md | `accepted` |
| THREAT-005 | Information disclosure | 認可コードやトークンを含む URL が Referer で外部へ渡る | System, OAuth2 | deployment.md: Security response headers | `covered` |
| THREAT-006 | Elevation of privilege | アカウントポータルのトークンで管理 API へ到達する | System, ApiTokens | authorization.md: スコープの語彙、REQ-APITOKENS-004 | `covered` |
| THREAT-007 | Denial of service | 低速な接続と過大な本体で接続枠とメモリを枯渇させる | System | deployment.md: HTTP server hardening | `covered` |
| THREAT-008 | Spoofing | ログイン画面を模した別のサイトが資格情報を受け取る | Authentication | WEBAUTHN3-AUTHENTICATION | `accepted` |

## Gateway and API process

| ID | Category | Threat | Contexts | Controls | Status |
|---|---|---|---|---|---|
| THREAT-009 | Repudiation | クライアントが `X-Request-ID` を偽装し、相関を壊して追跡を妨げる | System | observability.md: Request correlation | `covered` |
| THREAT-010 | Tampering | 受信ヘッダーの制御文字がログとレスポンスヘッダーへ注入される | System | observability.md: Request correlation | `covered` |
| THREAT-011 | Information disclosure | 平文への降格により、Cookie とトークンが経路上で読まれる | System | deployment.md: Security response headers | `accepted` |
| THREAT-012 | Spoofing | 同一オリジンでない構成で配備され、Cookie のスコープと `Origin` 検証の前提が崩れる | System | deployment.md: Runtime units | `planned` |

## Tenant boundary

| ID | Category | Threat | Contexts | Controls | Status |
|---|---|---|---|---|---|
| THREAT-013 | Elevation of privilege | 要求に含めた `tenant_id` や `sub` を信じさせ、別テナントを操作する | Tenancy | authorization.md: テナント境界、REQ-TENANCY-009 | `covered` |
| THREAT-014 | Information disclosure | 未知のサブドメインが既定テナントへ落ち、既定テナントの情報が見える | Tenancy | REQ-TENANCY-008 | `covered` |
| THREAT-015 | Information disclosure | 正規ロケーション以外の経路からテナントへ到達する | Tenancy | REQ-TENANCY-009、REQ-TENANCY-010 | `covered` |
| THREAT-016 | Information disclosure | 別テナントで発行されたページングのカーソルを再利用する | Tenancy, Audit | authorization.md: テナント境界、REQ-AUDIT-004 | `covered` |
| THREAT-017 | Information disclosure | 権限のない対象と存在しない対象の応答差から、他テナントの資源の存在を推測する | Tenancy, Authorization | authorization.md: その他の境界の規則 | `covered` |
| THREAT-018 | Elevation of privilege | 他テナントの関係タプルが細粒度認可の判定に寄与する | Authorization | REQ-AUTHORIZATION-006 | `covered` |
| THREAT-019 | Elevation of privilege | 非同期の実行が処理の途中で別テナントの範囲へ到達する | Jobs | REQ-JOBS-006、authorization.md: テナント境界 | `covered` |
| THREAT-020 | Spoofing | 他テナントの署名鍵で発行したトークンが受理される | SigningKeys, OAuth2 | REQ-SIGNINGKEYS-004、REQ-OAUTH2-034 | `covered` |
| THREAT-021 | Elevation of privilege | 他テナントのワークロード信頼設定を使って資格情報を得る | WorkloadIdentity | REQ-WORKLOADIDENTITY-007、REQ-WORKLOADIDENTITY-009 | `covered` |
| THREAT-022 | Denial of service | 単一テナントの投入がレーンを占め、他テナントの非同期処理が滞る | Jobs, Tenancy | REQ-JOBS-009 | `planned` |
| THREAT-023 | Information disclosure | 管理者が別テナントの監査記録を検索できる | Audit | REQ-AUDIT-001、authorization.md: テナント境界 | `covered` |

## Principals and permissions

| ID | Category | Threat | Contexts | Controls | Status |
|---|---|---|---|---|---|
| THREAT-024 | Elevation of privilege | 新設した管理 API がスコープ宣言を持たず、あらゆるスコープから到達できる | ApiTokens | authorization.md: 対話セッション限定の操作、REQ-APITOKENS-004 | `covered` |
| THREAT-025 | Elevation of privilege | トークン発行の操作を通じて、任意のスコープ集合を持つ資格情報を作る | ApiTokens, OAuth2 | authorization.md: 対話セッション限定の操作、REQ-AUTHENTICATION-004 | `covered` |
| THREAT-026 | Elevation of privilege | 外部 IdP の登録を通じて、任意のユーザーとしてサインインする経路を作る | Authentication | REQ-AUTHENTICATION-025 | `covered` |
| THREAT-027 | Elevation of privilege | 代行または委譲が、元の主体の権限を超える | OAuth2, Authorization | RFC8693-DELEGATION-DEFAULT、RFC8693-DELEGATION-DEPTH、REQ-OAUTH2-048、REQ-TENANCY-021、REQ-AUTHORIZATION-004 | `covered` |
| THREAT-028 | Repudiation | 代行による操作を本人の操作と区別できず、誰が行ったかを争える | Audit, OAuth2 | REQ-AUDIT-005、REQ-AUDIT-006、REQ-OAUTH2-049 | `covered` |
| THREAT-029 | Spoofing | 無効化または削除した主体のセッションとトークンが生き残る | IdManagement, Authentication, OAuth2 | REQ-PLATFORM-001、REQ-PLATFORM-002、REQ-AUTHENTICATION-009、REQ-OAUTH2-046、REQ-OAUTH2-012 | `covered` |
| THREAT-030 | Elevation of privilege | 通常のテナント管理者が制御面の操作へ到達する | Tenancy, SigningKeys | REQ-TENANCY-014、REQ-SIGNINGKEYS-009、REQ-DATAKEYS-006 | `covered` |
| THREAT-031 | Elevation of privilege | 判定に必要な事実が得られない状況が、許可として扱われる | Authorization | REQ-AUTHORIZATION-005、AUTHZEN-FGA-FAIL-CLOSED、authorization.md: その他の境界の規則 | `covered` |
| THREAT-032 | Information disclosure | 列挙が打ち切りを隠し、権限のない対象の存在を推測させる | Authorization | REQ-AUTHORIZATION-007 | `covered` |

## Credentials and authentication

| ID | Category | Threat | Contexts | Controls | Status |
|---|---|---|---|---|---|
| THREAT-033 | Spoofing | 資格情報の総当たりでアカウントへ到達する | Authentication | REQ-AUTHENTICATION-008、deployment.md: Availability and shared state | `covered` |
| THREAT-034 | Spoofing | 他所で漏洩したパスワードの使い回しでアカウントへ到達する | Authentication | NIST63B4-PASSWORD-MINIMUM、contexts/authentication/decisions.md | `accepted` |
| THREAT-035 | Information disclosure | データベースの流出からパスワードが復元される | Authentication | NIST63B4-PASSWORD-STORAGE | `covered` |
| THREAT-036 | Spoofing | 端末の記憶を悪用して第二要素を省略する | Authentication | REQ-AUTHENTICATION-027、REQ-AUTHENTICATION-028、REQ-AUTHENTICATION-029 | `covered` |
| THREAT-037 | Spoofing | パスワード再設定の導線を使ってアカウントを乗っ取る | Authentication | REQ-AUTHENTICATION-016、REQ-AUTHENTICATION-008 | `covered` |
| THREAT-038 | Information disclosure | 認証と復旧の応答差から、利用者名の存在を暴く | Authentication | REQ-AUTHENTICATION-016、contexts/authentication/decisions.md | `covered` |
| THREAT-039 | Spoofing | 外部 IdP が主張するメールアドレスを信じ、既存アカウントへ結び付ける | Authentication | REQ-AUTHENTICATION-001、REQ-AUTHENTICATION-002 | `covered` |
| THREAT-040 | Repudiation | 資格情報の変更が本人に知られないまま行われる | Authentication | REQ-AUTHENTICATION-030、REQ-AUTHENTICATION-031、REQ-AUTHENTICATION-032、REQ-AUTHENTICATION-033 | `covered` |
| THREAT-041 | Spoofing | 承認要求を繰り返し送り、利用者が誤って承認する | OAuth2 | CIBA-CORE-BINDING-MESSAGE、REQ-OAUTH2-043、REQ-OAUTH2-040 | `covered` |

## Protocol and tokens

| ID | Category | Threat | Contexts | Controls | Status |
|---|---|---|---|---|---|
| THREAT-042 | Tampering | 横取りした認可コードを交換する、または二重に交換する | OAuth2 | RFC7636-S256、RFC7636-VERIFY、RFC9700-AUTHORIZATION-CODE、REQ-OAUTH2-015 | `covered` |
| THREAT-043 | Tampering | 緩いリダイレクト先の照合により、認可の結果が別の宛先へ渡る | OAuth2 | RFC9700-REDIRECT-MATCH、RFC7591-REDIRECT-URI、REQ-OAUTH2-023 | `covered` |
| THREAT-044 | Spoofing | どの発行者からの応答かを区別できず、別の発行者の応答を受け入れる | OAuth2 | RFC9207-ISS | `covered` |
| THREAT-045 | Tampering | 使用済みのリフレッシュトークンを再生する | OAuth2 | RFC9700-REFRESH-REPLAY、REQ-OAUTH2-006、REQ-OAUTH2-018 | `covered` |
| THREAT-046 | Spoofing | 持ち出したアクセストークンを、発行時とは別の提示者が使う | OAuth2 | RFC9449-TOKEN-BINDING、RFC9449-ATH、RFC8705-CERT-BOUND、REQ-OAUTH2-029、REQ-OAUTH2-045 | `covered` |
| THREAT-047 | Tampering | PAR の `request_uri` を再利用する | OAuth2 | RFC9126-SINGLE-USE、REQ-OAUTH2-009 | `covered` |
| THREAT-048 | Spoofing | クライアントアサーションを改竄または再生してクライアントになりすます | OAuth2 | RFC7523-CLIENT-ASSERTION、REQ-OAUTH2-028、REQ-OAUTH2-007 | `covered` |
| THREAT-049 | Information disclosure | クライアント認証の失敗理由の差から、登録済みクライアントの存在を見分ける | OAuth2 | REQ-OAUTH2-007 | `covered` |
| THREAT-050 | Information disclosure | クライアントメタデータの取得を通じて、内部ネットワークへ到達させる | OAuth2 | REQ-OAUTH2-017、CIMD00-URL-SHAPE、CIMD00-FETCH | `covered` |
| THREAT-051 | Denial of service | プロトコルエンドポイントへの大量要求で正規の利用を妨げる | OAuth2 | REQ-OAUTH2-040、deployment.md: Endpoint rate limiting、capacity.md: Degradation order | `covered` |
| THREAT-052 | Tampering | 署名アルゴリズムの取り違えを突いて署名検証を回避する | OAuth2, SigningKeys | RFC7518-SIGNATURE-ALGORITHMS、RFC9068-ASYMMETRIC-SIGNATURE | `covered` |
| THREAT-053 | Tampering | XML 署名の構造を組み替え、検証を通したまま別の内容を主張する | Saml, WsFederation | contexts/saml/decisions.md、contexts/saml/internals.md | `covered` |
| THREAT-054 | Denial of service | 圧縮された受信要求の展開でメモリを枯渇させる | Saml | contexts/saml/internals.md | `covered` |

## Keys and reversible secrets

| ID | Category | Threat | Contexts | Controls | Status |
|---|---|---|---|---|---|
| THREAT-055 | Information disclosure | 暗号文を別テナント、別テーブル、別フィールドへ複製して復号する | DataKeys | database.md: Envelope encryption for reversible secrets | `covered` |
| THREAT-056 | Elevation of privilege | マスターキー提供元へ到達できないときに平文へ退避する | DataKeys | database.md: Envelope encryption for reversible secrets | `covered` |
| THREAT-057 | Information disclosure | 破棄したはずの鍵で、残った暗号文が復号できる | DataKeys | REQ-DATAKEYS-003、REQ-DATAKEYS-005 | `covered` |
| THREAT-058 | Information disclosure | 管理 API またはエラー応答から鍵素材が出る | DataKeys, SigningKeys | REQ-DATAKEYS-006、authorization.md: 応答が決して含まないもの | `covered` |
| THREAT-059 | Denial of service | 鍵提供元の障害が、発行と検証の両方を止める | SigningKeys, OAuth2 | REQ-OAUTH2-039、REQ-SIGNINGKEYS-008、REQ-SIGNINGKEYS-001 | `covered` |
| THREAT-060 | Tampering | 攻撃者の鍵を JWKS へ紛れ込ませ、偽造したトークンを信頼させる | SigningKeys | REQ-SIGNINGKEYS-004、REQ-SIGNINGKEYS-010、REQ-SIGNINGKEYS-011 | `covered` |
| THREAT-061 | Information disclosure | 起動時設定のシークレットが、生成した設定リファレンスやログへ出る | System | REQ-SYSTEM-016、REQ-SYSTEM-017、glossary.md: 外部契約 | `covered` |
| THREAT-062 | Information disclosure | 平文の鍵を含むバックアップが、保存先の権限から持ち出される | SigningKeys, DataKeys | runbooks/backup-restore-dr.md | `planned` |

## Upstream external authorities

| ID | Category | Threat | Contexts | Controls | Status |
|---|---|---|---|---|---|
| THREAT-063 | Spoofing | 未登録の発行者によるアテステーションが受理される | WorkloadIdentity | REQ-WORKLOADIDENTITY-002、REQ-WORKLOADIDENTITY-008 | `covered` |
| THREAT-064 | Spoofing | 署名が不正、または期限切れのアテステーションが受理される | WorkloadIdentity | REQ-WORKLOADIDENTITY-003、REQ-WORKLOADIDENTITY-004 | `covered` |
| THREAT-065 | Elevation of privilege | 主体が複数の関連付けに一致し、意図しない Agent として扱われる | WorkloadIdentity | REQ-WORKLOADIDENTITY-005、REQ-WORKLOADIDENTITY-006 | `covered` |
| THREAT-066 | Tampering | 取り込みが読み取り専用属性や未対応の操作で内部状態を壊す | Sourcing | REQ-SOURCING-004、RFC7644-PATCH | `covered` |
| THREAT-067 | Elevation of privilege | 上流が書き込んだ属性が動的な所属を通じて実効ロールへ波及する | Sourcing, IdManagement | — | `planned` |
| THREAT-083 | Elevation of privilege | 上流が同期したグループ所属が、動的な規則を経ずに実効ロールを直接動かす | Sourcing, IdManagement | REQ-SOURCING-005 | `planned` |

## Downstream external receivers

| ID | Category | Threat | Contexts | Controls | Status |
|---|---|---|---|---|---|
| THREAT-068 | Information disclosure | 配信が誤った接続先へ個人データを送る | Provisioning | REQ-PROVISIONING-002、REQ-PROVISIONING-015、REQ-PROVISIONING-018 | `covered` |
| THREAT-069 | Tampering | ログアウトトークンを再生し、任意のセッションを落とす | OAuth2 | OIDC-BACKCHANNEL-REPLAY、OIDC-BACKCHANNEL-LOGOUT-TOKEN、REQ-OAUTH2-025 | `covered` |
| THREAT-070 | Tampering | 署名の無い、または検証されないセキュリティイベントが受信側で信じられる | SharedSignals | RFC8417-SET-SIGNED、RFC8417-SET-VERIFY | `covered` |
| THREAT-071 | Information disclosure | ジョブの入出力に混ざった個人データやシークレットが管理 API から出る | Jobs | REQ-JOBS-014、contexts/jobs/decisions.md | `covered` |
| THREAT-072 | Denial of service | 下流の障害が実行枠を占有し、同じレーンの他の非同期処理を止める | Jobs, Provisioning | REQ-JOBS-005、REQ-JOBS-009 | `planned` |

## Persistence and asynchronous execution

| ID | Category | Threat | Contexts | Controls | Status |
|---|---|---|---|---|---|
| THREAT-073 | Tampering | 入力が問い合わせの構造として解釈される | 全 Context | database.md: Ports and adapters | `covered` |
| THREAT-074 | Repudiation | 状態は変わったのに、対応する監査イベントが残らない | Audit | — | `planned` |
| THREAT-075 | Tampering | 再試行と再取得によって副作用が重複して起きる | Jobs | REQ-JOBS-003、REQ-JOBS-004、REQ-JOBS-007 | `covered` |
| THREAT-076 | Denial of service | 有効期限を過ぎた一時データの滞留が容量を圧迫する | Jobs, OAuth2, Authentication | deployment.md: Availability and shared state | `covered` |

## Operator and control plane

| ID | Category | Threat | Contexts | Controls | Status |
|---|---|---|---|---|---|
| THREAT-077 | Elevation of privilege | 実行環境を取得した者が、HTTP を経ずに任意の状態を作る | Seeding | authorization.md: テナント境界 | `accepted` |
| THREAT-078 | Tampering | 本番で開発用の構成や環境変数由来のシークレットが使われる | Seeding | REQ-SEEDING-005、REQ-SEEDING-007、REQ-SEEDING-008 | `covered` |
| THREAT-079 | Tampering | 適用が、運用中に加えた変更を黙って上書きする | Seeding | REQ-SEEDING-009、REQ-SEEDING-010 | `covered` |
| THREAT-080 | Information disclosure | 指標の公開先からテナントの活動が推測される | System | observability.md: Metrics | `covered` |
| THREAT-081 | Tampering | 一部のプロセスだけが検証されない設定値で起動する | System | REQ-SYSTEM-016、structure.md: Context internals | `covered` |
| THREAT-082 | Tampering | 取り込んだ依存物と配布する成果物の来歴を確かめられず、差し替えを識別できない | 全 Context | structure.md: Stack | `planned` |

## Accepted residual risk

`accepted` は、制御を持たないことを意図して選んだ行である。それぞれについて、選んだ理由と、その選択を見直すべき条件を書く。**条件が書かれていない受容は、受容ではなく放置である。**

**THREAT-004（ブラウザーが保持するアクセストークン）** — 純粋な単一ページアプリの RP として、アクセストークンをブラウザーに保持することを受け入れている。BFF を挟むとセッション状態を持つ層が増え、IdP 自身の可用性と復旧の経路が複雑になるからである。影響は短命なトークンと `no-store` で限定する。**再検討の条件**は、より長命なトークンをブラウザーへ渡す必要が生じたとき、または管理コンソールが単一ページアプリ以外の形態を持つときである。

**THREAT-008（ログイン画面の模倣）** — IdP 単独では、利用者が別のサイトへ資格情報を入力することを防げない。フィッシングに耐える要素として WebAuthn を提供するが、その利用はテナントの選択である。**再検討の条件**は、フィッシング耐性のある要素をテナントへ強制できるポリシーを持ったときである。

**THREAT-011（平文への降格）** — `Strict-Transport-Security` は既定で無効であり、TLS を終端する側が設定する。平文の `http` を使う開発環境に影響させないためである。**再検討の条件**は、製品が TLS 終端を自ら担う配備形態を支援するときである。

**THREAT-034（漏洩パスワードの使い回し）** — 漏洩との照合は `BreachedPasswordChecker` として存在するが、既定は何もしないアダプターである。有効にした場合、同梱辞書との照合は確実に働き、外部の HIBP を使う追加の照合だけが障害時にフェイルオープンする。外部への依存を既定で持ち込まないという判断の帰結として、**既定の構成では漏洩照合が働かない**ことを受け入れている。**再検討の条件**は、同梱辞書だけで既定を有効にできると判断したとき、または規制がこの照合を要求するときである。

**THREAT-077（実行環境を取得した者）** — `Seeding` は権限ではなく実行環境そのものを境界とする。プロセスを起動できる者は既にデータベースへ到達できるので、この経路を塞いでも実効的な防御にならない。**再検討の条件**は、`Seeding` に HTTP の入口を与えるときである。

## When to revisit

この文書は書いた直後から古くなる。次のいずれかが起きた変更は、脅威モデルの見直しを伴う。

- **新しい信頼境界が増える。** 新しい実行単位、新しい外部依存、新しいネットワーク経路。
- **主体の種類が増えるか、既存の主体が新しい境界へ到達できるようになる。** [authorization.md](authorization.md) の主体の表かスコープの語彙が変わるとき。
- **新しい外部連携が加わる。** 上流の権威、下流の受信者、新しいプロトコルのバインディング。
- **資産が増える。** 新しい種類の秘密、個人データ、または後から証明を求められる記録を持つとき。
- **`accepted` の再検討の条件が満たされる。**

この義務は [仕様先行の開発ワークフロー](development/specification-first-workflow.md) の現在状態の同期に含まれる。`planned` の行は、応える規範が書かれた時点で `covered` へ移し、`Controls` にその規範 ID を入れる。`—` のまま `covered` になる行は無い。
