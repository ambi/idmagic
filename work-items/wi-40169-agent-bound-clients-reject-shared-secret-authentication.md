---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-23
priority: p2
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/domain/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-009 }
  - { path: docs/domain/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-046 }
  - { path: spec/contexts/identity-management/main.tsp, symbol: BindAgentCredential }
---

# Agent に束縛したクライアントは共有シークレットで認証できないようにする

## 動機

IETF WIMSE WG が `draft-ietf-wimse-aims-00`（AI Identity Management System、2026-09-15、Informational）を公開した。
この draft は新しいプロトコルを定義せず、既存の WIMSE、SPIFFE、OAuth、OpenID SSF をエージェントの認証と認可へどう組み合わせるかを示す。
IdMagic が関わるのは、エージェントが OAuth クライアントとして認可サーバーからトークンを得る部分（§7、§10）と、監視と是正（§11）である。

draft は、エージェントが認可サーバーへ自身を認証するとき、静的で長期の共有シークレットを使わないことを三か所で求める。

- §7 の注記は、静的な API キーを、暗号的に束縛されず識別も伝えず、長期でローテーションしにくいベアラー資格情報として、エージェント ID の反パターンと位置づける。
- §10.4.1（利用者からの委譲、Authorization Code Grant）は、トークン要求でのクライアント認証を RFC 7523、RFC 8705、`draft-ietf-oauth-spiffe-client-auth` のいずれかで行い、"not with the use of static, long-lived client secrets" と書く。
- §10.4.2（エージェント自身の認可、Client Credentials Grant）も同じ制約を置く。

IdMagic では、`Agent` は自身の資格情報を持たず、`AgentCredentialBinding` で束縛した `OAuth2Client` の資格情報でトークンを得る。
ところが、束縛の作成もトークン発行も、そのクライアントの `token_endpoint_auth_method` を見ていない。
`client_secret_basic` や `client_secret_post` のクライアントを Agent に束縛でき、そのシークレットだけで Agent としてのトークンを取得できる。
Agent を第三の主体として扱い、長期シークレットを配らずに実行環境を認証するという WorkloadIdentity の主張と、この経路は矛盾する。

## 対象範囲

- Agent に束縛できる `OAuth2Client` を、非対称鍵で自身を証明する認証方式（`private_key_jwt`、`tls_client_auth`）のクライアントに限る。
  束縛の作成時に、それ以外の方式（`client_secret_basic`、`client_secret_post`、`none`）のクライアントを拒否する。
- トークンエンドポイントで認証したクライアントが Agent に束縛されていれば、そのクライアントの認証方式を発行の前提条件として検査する。
  共有シークレットまたは `none` のクライアントには、グラントの種類によらず Agent としてのトークンを発行しない。
  束縛後にクライアントの認証方式が変更された場合も、この検査で拒否される。
- 規範シナリオを追加する。
  束縛の拒否は IdManagement、発行の拒否は OAuth2 の `scenarios.feature.md` に置く。
- `docs/domain/oauth2/standards.md` へ `draft-ietf-wimse-aims-00` の節を追加し、この変更が満たす要件行を宣言する。
- `BindAgentCredential` の 400 応答へ、拒否理由を区別できるエラーを追加する。
- 管理コンソールのエージェント画面で、束縛できないクライアントを選んだときの拒否理由を表示する。

## 対象外

- `draft-ietf-oauth-spiffe-client-auth`（JWT-SVID や X.509-SVID をクライアント認証に直接使う方式）への対応。
  WorkloadIdentity は JWT-SVID を Token Exchange の `subject_token` として受ける経路をすでに持ち、長期シークレットを配らない要件はその経路で満たせる。
  draft の改訂が進み、Token Exchange を経由しないクライアント認証を求める利用者が現れた時点で別の work item にする。
- Transaction Tokens（§10.5、`draft-ietf-oauth-transaction-tokens-11`）。
  Transaction Token の発行者はリソースサーバー側の信頼ドメインの内部に置かれる構成が基本であり、IdMagic が担う範囲を別に検討する必要がある。
- Identity Assertion JWT Authorization Grant とドメイン間の連鎖（§10.6）。
  既存の Cross-App Access の work item が扱う。
- 改ざん検知可能な監査ログ（§11 の "tamper-evident"）。
  既存の改ざん検知監査ログの work item が扱う。
- WIT、WPT、HTTP Message Signatures によるアプリケーション層のワークロード認証（§9.2）。
  ワークロード間の認証であり、認可サーバーの責務ではない。
- 大規模言語モデルに資格情報を渡さない要件（§8 末尾）。
  エージェント実装の責務であり、IdP が強制する手段を持たない。
- Agent に束縛していないクライアントの共有シークレット認証。
  人間の利用者のためのアプリケーションは AIMS の対象外であり、従来どおり受け付ける。

## 設計

### AIMS の各節と IdMagic の対応状況

| AIMS の節 | 求める内容 | IdMagic の状況 | この work item での扱い |
| --- | --- | --- | --- |
| §6 Agent Identifier | エージェントごとに一意で安定した識別子 | `Agent` を主体として持ち、`AgentWorkloadBinding` で SPIFFE ID などの外部主体を対応付ける | 変更しない |
| §7 Agent Credentials | 識別子へ暗号的に束縛した短期の資格情報、静的 API キーを使わない | Agent に束縛したクライアントが共有シークレットで認証できる | 対象（束縛と発行で拒否する） |
| §8 Credential Provisioning | 実行時の自動発行と姿勢評価 | JWT-SVID などの外部アテステーションを検証して短期トークンへ交換する | 変更しない |
| §10.3 Access Token | `client_id` にエージェント、`sub` に委譲元 | RFC 9068 のクレームに加えて `agent_id` と `act` チェーンを載せる | 変更しない |
| §10.4.1、§10.4.2 トークン取得 | 非対称のクライアント認証を使い、静的なシークレットを使わない | `private_key_jwt` と `tls_client_auth` を提供するが、Agent への強制がない | 対象 |
| §10.5 Transaction Tokens | 内部呼び出し連鎖の縮小トークン | 未対応 | 対象外 |
| §10.6 Cross Domain Access | ID Chaining、ID-JAG | 未対応（既存の work item が保留中） | 対象外 |
| §10.7 Human in the Loop | CIBA による承認 | CIBA と `Supervised` の承認強制を実装済み | 変更しない |
| §10.10 Discovery | AS Metadata、PRM、CIMD、DCR | 実装済み | 変更しない |
| §11 Monitoring | SSF、CAEP、RISC、監査の最小項目、改ざん検知 | SSF と CAEP、委譲チェーンの監査軸は実装済み、改ざん検知は未対応 | 対象外 |

表のうち、IdMagic 単独で閉じ、既存の work item がなく、draft が明示的に禁じる経路を塞ぐものは §7 と §10.4 の組だけである。
この work item はその一点に絞る。

### 許可する認証方式

Agent に束縛できるクライアントの認証方式は `private_key_jwt` と `tls_client_auth` に限る。
どちらも、クライアントが保持する秘密鍵の所有を要求ごとに証明し、IdMagic 側には公開鍵か証明書の Subject DN だけが残る。
FAPI 2.0 プロファイルのクライアントに同じ二方式を課す `FAPI2-CLIENT-AUTH` と同じ集合であり、許可集合を一つの定義で共有できる。

`none` も拒否する。
公開クライアントは要求の送り手を証明しないため、Agent としてのトークンを受け取る主体を確定できない。
`client_credentials` はすでに confidential クライアントに限られているが、Authorization Code Grant と Token Exchange は公開クライアントを受け付けるため、明示的に閉じる必要がある。

### 強制点を二か所に置く理由

束縛の作成時だけで検査すると、束縛後に管理者がクライアントの認証方式を共有シークレットへ変えた場合を防げない。
クライアントは OAuth2 Context、束縛は IdManagement Context が所有しており、クライアント更新のたびに束縛の有無を問い合わせると、OAuth2 から IdManagement への依存が更新経路に増える。

そこで、発行時の検査を正とする。
Agent へのトークン発行では、`client_credentials` と CIBA の承認後の発行が `ResolveIssuableAgent` 系の関数で束縛先の Agent を解決し、Agent の `status` と所有者の検査もそこに集まっている。
トークン交換は交換に関わる Agent を別の関数で洗い出して `kind` を検査しているため、両方の判定から同じ認証方式の検査を呼ぶ。
こうすれば、束縛後の変更も含めて fail-closed になる。
束縛時の検査は、管理者が誤った構成を作った時点で理由を返すための補助として置く。

### 採用しない代替案

- **Agent に束縛したクライアントの認証方式の変更を、クライアント更新の時点で拒否する**：
  OAuth2 のクライアント更新が IdManagement の束縛を読む依存を新たに持つ。
  発行時の検査で安全性は満たせるため、依存を増やす利点がない。
- **テナント設定で共有シークレットを許可できるようにする**：
  未リリースの製品であり、共有シークレットで Agent を運用している既存利用者はいない。
  許可の設定を置くと、AIMS の反パターンを選べる構成を製品として提供することになる。
- **拒否せず警告と監査だけにする**：
  束縛の目的は Agent の資格情報を暗号的に証明可能にすることであり、警告だけではその保証が成り立たない。

### 拒否の表現

トークンエンドポイントでは、ほかの Agent 発行の拒否（無効化、所有者のオフボード）と揃えて `invalid_client` を返す。
認証方式の不一致をクライアントへ区別して知らせると、束縛の有無を外部から探れるためである。
監査イベントには拒否理由を区別して残す。

`BindAgentCredential` は管理者向けの操作であり、理由を区別して返す。
400 応答の union へ、共有シークレットまたは `none` のクライアントを束縛しようとしたことを表すエラーを追加する。

## 計画

1. 仕様を先に更新する。
   `standards.md` へ AIMS の節と要件行を追加し、IdManagement と OAuth2 のシナリオ、`BindAgentCredential` のエラーを追加して再生成する。
2. OAuth2 の発行時検査を実装する。
   `client_credentials`、トークン交換、CIBA、認可コード、リフレッシュトークンの各経路が Agent の解決を通るかを着手時に確認し、通らない経路があれば同じ検査へ寄せる。
3. IdManagement の束縛時検査を実装する。
   束縛の作成はクライアントの認証方式を読む必要があるため、既存の OAuth2 クライアント参照ポートで足りるかを確認する。
4. 管理コンソールで拒否理由を表示する。
5. テストと e2e の fixture のうち、共有シークレットのクライアントを Agent に束縛しているものを非対称の方式へ置き換える。
   シードデータには Agent の束縛がないことを起票時に確認した。

作るものを変え得る問いは次のとおり解決済みである。

- 許可集合は `private_key_jwt` と `tls_client_auth` とし、`none` を含めない。
- 束縛後の認証方式の変更は、クライアント更新では拒否せず発行時に拒否する。
- 既存データの移行は行わない（未リリースのため）。

## タスク

- [ ] T001 [Spec] `docs/domain/oauth2/standards.md` へ `draft-ietf-wimse-aims-00` の節と要件行を追加する。
- [ ] T002 [Spec] 束縛の拒否と発行の拒否の規範シナリオ、`BindAgentCredential` のエラーを追加し、再生成する。
- [ ] T003 [Acceptance] 共有シークレットのクライアントを束縛した Agent が `client_credentials` とトークン交換でトークンを得られることを、受け入れテストの RED として観測する。
- [ ] T004 [OAuth2] Agent の発行判定へ認証方式の検査を加え、単体 RED から GREEN にする。
- [ ] T005 [IdManagement] 束縛の作成時に認証方式を検査し、単体 RED から GREEN にする。
- [ ] T006 [UI] 管理コンソールのエージェント画面で束縛の拒否理由を表示する。
- [ ] T007 [Fixture] 共有シークレットのクライアントを Agent に束縛しているテストと e2e の fixture を置き換える。
- [ ] T008 [Verify] 検出能力を変異テストで確かめ、全体の検証を通す。

## 検証

- `mise run verify-spec`
- `mise run test-go`
  - 理由：束縛時の拒否、各発行経路での拒否、束縛後の認証方式変更による拒否、非対称方式での発行成功の境界を確かめる。
- `mise run test-go-mutation -- backend/oauth2/usecases`
- `mise run verify-ui`
- `mise run verify`

## リスク

リスクは medium とする。
トークン発行の前提条件を増やす変更であり、検査を誤ると正当な Agent がトークンを得られなくなる。
逆に、Agent の解決を通らない発行経路が残っていると、共有シークレットによる取得が塞がらない。
計画の 2 で全経路を洗い出し、経路ごとに受け入れテストを置いて両方向の誤りを検出する。

draft は `-00` の Informational であり、今後の改訂で文言が変わる可能性がある。
ただし、この work item が依拠する「静的な長期シークレットでエージェントを認証しない」要件は、RFC 7523 と RFC 8705 という確定した仕様を根拠にしており、draft の改訂で覆る可能性は低いと判断する。
判断は取り消し可能であり、許可集合の定義を変えれば元に戻せる。
