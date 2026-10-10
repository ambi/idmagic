---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-09
priority: p3
depends_on: []
change_kind: refactor
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: domain の関数が現在時刻と乱数を引数で受け取る形へ変え、作用の検査の対象を名前解決と接続へ狭めるだけで、HTTP の応答、生成する値の形式、発行するイベントは変わらず、リリースの読者へ知らせることがない。
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - tools/check/src/boundary-fitness.ts
    - tools/check/boundary-debt.json
    - docs/design/application/backend.md
    - docs/design/application/design-guidelines.md
    - backend/authentication/federation/domain/models.go
    - backend/authentication/trusteddevice/domain/trusted_device.go
    - backend/idmanagement/group/domain/group_csv.go
    - backend/oauth2/approval/domain/approval_request.go
    - backend/oauth2/authorization/domain/authorization_code.go
    - backend/oauth2/client/domain/cimd.go
    - backend/oauth2/client/domain/client.go
    - backend/oauth2/device/domain/device_authorization.go
    - backend/oauth2/domain/mcp_resource_server.go
    - backend/oauth2/token/domain/refresh_token.go
    - backend/provisioning/domain/connection.go
    - backend/saml/domain/authnrequest.go
    - backend/seeding/domain/plan.go
  tests:
    - tools/check/src/boundary-fitness.test.ts
  stop_before_reading: [frontend, spec, docs/modules]
spec_impact: { kind: none, reason: "domain のパッケージが自分で得ていた現在時刻、乱数、ネットワークの値を、usecases から引数で受け取る形へ変えるだけである。生成する識別子と期限の規則、HTTP の応答、発行するドメインイベント、永続状態は変えない。" }
---

# domain のパッケージが現在時刻、乱数、ネットワークを直接使わないようにする

## 動機

`docs/design/application/backend.md` は、`domain` で `time.Now`、`crypto/rand`、`math/rand`、OS、ネットワークを直接使わないと定めている。
`tools/check/boundary-debt.json` には、この規則の違反（`domain-effect`）が 20 件ある。

| Context | ファイル | 使っているもの |
| --- | --- | --- |
| Authentication | `federation/domain/models.go` | `net`、`time.Now` |
| Authentication | `trusteddevice/domain/trusted_device.go` | `crypto/rand` |
| IdManagement | `group/domain/group_csv.go` | `net` |
| OAuth2 | `approval/domain/approval_request.go`、`authorization/domain/authorization_code.go`、`device/domain/device_authorization.go`、`token/domain/refresh_token.go` | `crypto/rand`、`time.Now` |
| OAuth2 | `client/domain/cimd.go`、`client/domain/client.go`、`domain/mcp_resource_server.go` | `net`、`time.Now` |
| Provisioning | `domain/connection.go` | `net` |
| SAML | `domain/authnrequest.go` | `net`、`time.Now` |
| Seeding | `domain/plan.go` | `net` |

時刻と乱数を内部で得る関数は、テストで値を固定できず、期限や識別子の性質を決定的に検証できない。

## 対象範囲

- `time.Now` と `crypto/rand` を使う関数を、現在時刻と乱数（またはトークンの生成器）を引数で受け取る形へ変え、`usecases` から渡す。
- `net` の使用を、ファイルごとに分類する。
  URL や IP アドレスの構文の検証のように入力だけで決まる計算に `net` の型と関数を使っているものは、検査が誤って作用とみなしているかを確かめる。
  名前解決や接続を行っているものは `ports` へ移す。
- 解消した違反 ID を `tools/check/boundary-debt.json` から消す。

## 対象外

- 識別子や期限の生成規則の変更。

## 設計

`net` の扱いは、着手時に分類の結果で決める。
`net.ParseIP` や `net/url` の構文解析は決定論的な計算であり、`structure.md` が禁じる「ネットワークへの直接アクセス」ではない。
分類の結果、構文解析しか使っていないファイルが多ければ、検査の規則を「`net` の import」から「名前解決と接続の関数の呼び出し」へ狭めるほうが規則の意図に合う。
その場合は検査の変更をこの項目に含め、規則の文言と検査を一致させる。

### `net` の分類の結果

| ファイル | import | 使っている関数 | 分類 |
| --- | --- | --- | --- |
| `authentication/federation/domain/models.go` | `net/url` | `url.Parse` | 構文解析 |
| `idmanagement/group/domain/group_csv.go` | `net/mail` | `mail.ParseAddress` | 構文解析 |
| `oauth2/client/domain/cimd.go`、`oauth2/client/domain/client.go`、`oauth2/domain/mcp_resource_server.go` | `net/url` | `url.Parse` | 構文解析 |
| `provisioning/domain/connection.go`、`seeding/domain/plan.go` | `net/url` | `url.Parse` | 構文解析 |
| `saml/domain/authnrequest.go` | `net/url` | `url.ParseQuery`、`url.QueryEscape` | 構文解析 |

8 ファイルすべてが入力だけで結果が決まる構文解析であり、名前解決や接続を行うものはない。
そこで、検査が作用とみなす `net` の import から、値の構文だけを扱う `net/url`、`net/mail`、`net/netip` を除く。
`net` そのもの、`net/http` などの接続を行うパッケージは、引き続き作用として拒否する。
`backend.md` の規則にも、構文解析はネットワークへのアクセスに当たらないことを書く。

### 乱数の供給元（D5）

| 項目 | 内容 |
| --- | --- |
| 入力 | `backend.md` の `domain` の規則、[作用の境界](../../docs/design/application/design-guidelines.md#作用の境界)、本項目のリスクの節（`math/rand` を受け付けない型にする） |
| 適用した判断基準 | D5：複数のモジュール（Authentication、OAuth2）が「暗号論的に安全な乱数を、テストで固定できる形で受け取る」という同じ技術的な規則を必要とする。どのモジュールの型も業務上の意味も必要としない |
| 採る案 | `backend/shared/security/entropy` に `Source` 型を置く。フィールドは非公開で、本番の構築関数 `Crypto()` は `crypto/rand` を包む。テストは固定のバイト列から作る `Fixed` を使う。`Source` は `Read` と、偏りのない整数を返す `Int` を持つ。`io.Reader` を受け取る構築関数は置かないので、`math/rand` は渡せない |
| 配線 | 乱数を必要とする usecases が呼び出しの場所で `entropy.Crypto()` を作って渡す。usecases は作用を編成する層なので、依存の構造体へ新しいフィールドを足さない |
| 採らない案 | `io.Reader` で受け取る案は、`math/rand` の `*Rand` も満たすのでリスクの節を満たさない。乱数のバイト列を渡す案は、`user_code` の偏りのない選択に可変長の入力が要り、その関数だけ形が揃わない |

### 時刻を補う処理

`domain` の関数のうち、渡された時刻がゼロのときに `time.Now` で補っているものは、補う処理を消す。
本番の呼び出し元がゼロ時刻を渡し得る場合だけ、その usecases で現在時刻を得る。
ゼロ時刻のまま期限を判定すると、期限切れのものを有効と判定するので、呼び出し元を一つずつ確かめる。

### 証拠

`domain` の関数の引数を変え、そのテストも書き換えるので、[振る舞いを保つ変更](../../docs/development/specification-first-workflow.md#振る舞いを保つ変更)には当たらない。

| 証拠 | 内容 |
| --- | --- |
| 受け入れ RED | N/A: 製品の振る舞いを変えない作用の配置の変更であり、対応する REQ がない。代替として、台帳から `domain-effect` を先に消し、`mise run check-boundaries` が未記録の違反として失敗することを確かめる |
| 単体 RED | 検査の変更は `boundary-fitness.test.ts` に、`net/url`、`net/mail`、`net/netip` だけを import する `domain` を違反にしない例を足し、変更前の検査で失敗させる。`entropy` は `Fixed` の値から `Int` と `Read` の結果が決まることを、実装前に失敗させる |
| 変更耐性 | `mise run test-go-mutation` を `entropy` と変更した `domain` のパッケージへ走らせ、生き残りを読む |

## タスク

- [x] T001 [Inventory] `net` を使う 7 ファイルを、構文解析だけか、名前解決や接続を行うかで分類する。
  - 実際には 8 ファイルあり、すべて構文解析だった。結果は設計の「`net` の分類の結果」に記録した
- [x] T002 [App] 現在時刻と乱数を引数で受け取る形へ変える。
  - `backend/shared/security/entropy` を足した。RED: `mise run test-go-package -- ./backend/shared/security/entropy` が実装前にビルドの失敗で落ち、実装後に通った
  - 乱数: `NewTrustedDevice`、`TrustedDevice.Rotate`、`GenerateAuthReqID`、`GenerateAuthorizationCode`、`GenerateDeviceCode`、`GenerateUserCode`、`GenerateInitialRefreshToken`、`RotateRefreshToken` が `entropy.Source` を受け取る。usecases は呼び出しの場所で `entropy.Crypto()` を渡す
  - 時刻: ゼロ時刻を補う処理を、approval、authorization code、device、refresh token、federation、SAML の domain から消した。CIMD の解析は時刻を引数で受け取り、取得するアダプターが `time.Now()` を渡す。SAML は時刻を内部で取る `ValidateSignIn` と時刻を受け取る `ValidateSignInAt` を、時刻を受け取る `ValidateSignIn` 一つにした
  - 本番の呼び出し元は、どれも usecases か handler で現在時刻を得てから渡しており、ゼロ時刻は届かない。`GenerateAuthorizationCode` と `IsCodeExpired` は別名の宣言以外に本番の呼び出し元がない
  - ゼロ時刻の補完を固定していたテストの表明（`TestIsCodeExpired`、`TestIsDeviceExpired`、`TestIsRefreshTokenAbsoluteExpired` の末尾と `TestGenerateInitialRefreshToken_ZeroNow`）は、本番から届かない振る舞いなので（b）書かない、に分類して消した。どれも `//spec:covers` を持たない
  - `TestGenerateUserCode_DrawsEachCharacterFromTheSource` を足し、固定した乱数から `user_code` が決まることと、上限以上の値を捨てることを確かめる
  - 検査: `mise run lint-go`、`mise run test-go-changed`
- [x] T003 [App] 分類に従い、`net` の使用を `ports` へ移すか、検査を狭める。
  - 検査が作用とみなす `net` から `net/url`、`net/mail`、`net/netip` を除いた。RED: `mise run test-tools-file -- check/src/boundary-fitness.test.ts` の `allows domain packages to parse URLs, mail addresses, and IP addresses` が変更前に失敗し、変更後に通った
  - `backend.md` の規則に、構文解析はネットワークへのアクセスに当たらないことと、乱数を `entropy.Source` で受け取ることを書いた
- [x] T004 [Tooling] 解消した違反 ID を `tools/check/boundary-debt.json` から消す。
  - `domain-effect` の 20 件をすべて消した。台帳の違反 ID は 157 件から 137 件になった
  - 検査: `mise run check-boundaries`、`mise run check-boundary-debt-ratchet -- main`
- [x] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run verify`

## リスク

- 乱数の供給元を引数にすると、本番の配線で弱い乱数を渡す誤りがありうる。
  本番の配線では `crypto/rand` を使う生成器だけを渡し、`math/rand` を受け付けない型にする。

## 完了

- **完了日**: 2026-10-10
- **要約**:
  `mise run spec-diff` は規範の差分なしを報告する。
  `domain` の関数が `time.Now` と `crypto/rand` を直接使わず、現在時刻と乱数を引数で受け取る形にした。
  乱数は共有ライブラリ `backend/shared/security/entropy` の `Source` で受け取る。本番は `crypto/rand` を包む `entropy.Crypto()` だけを使い、`io.Reader` から作る構築関数を置かないので `math/rand` は渡せない。
  `domain-effect` の検査は、構文解析だけを行う `net/url`、`net/mail`、`net/netip` を作用とみなさないようにし、`backend.md` の規則を合わせた。
  境界の負債の台帳から `domain-effect` の 20 件を消し、台帳は 157 件から 137 件になった。
  生成する値の形式、HTTP の応答、発行するイベント、永続状態は変えていない。
- **受け入れ RED の証拠**:
  - **テスト**: `mise run check-boundaries`
  - **要件**: N/A: 規範上の振る舞いを変えない、作用の配置の変更であり、対応する REQ がない。
  - **観測した失敗**: `domain-effect` を消した台帳を、domain を書き換える前のチェックポイント（`entropy` を足しただけの状態）に置くと、`mise run check-boundaries` が Authentication の 2 件、OAuth2 の 9 件、SAML の 1 件を `violation(s) are absent from boundary-debt.json` として失敗した。変更後は同じ台帳で通った。
  - **検出できる理由**: 作用の配置を変える作業なので、製品の振る舞いの RED は存在しない。代わりに、リファクタリングを必要にした構造ゲートが変更の前後の違いを観測した。
- **単体 RED の証拠**:
  - **テスト**: `allows domain packages to parse URLs, mail addresses, and IP addresses`（`tools/check/src/boundary-fitness.test.ts`）、`TestFixedReadReturnsTheGivenBytesInOrder`、`TestFixedReadFailsWhenTheBytesRunOut`、`TestFixedIntRejectsValuesAtOrAboveTheBound`、`TestCryptoReadFillsTheBuffer`（`backend/shared/security/entropy`）
  - **要件**: N/A: 検査器と共有ライブラリの変更であり、対応する REQ がない。
  - **観測した失敗**: 検査のテストは、変更前の検査が `net/url` などの import を `domain-effect:...:net` として報告して失敗した。`entropy` のテストは、実装前にパッケージのビルドの失敗で落ちた。
  - **検出できる理由**: どちらも変更する判断（作用とみなす import の集合、供給元から値を読む規則）を直接表明する。
- **変更耐性の結果**:
  `mise run test-go-mutation -- backend/shared/security/entropy` は 1 件中 1 件を検出した。
  `mise run test-go-mutation -- backend/oauth2/domain` の生き残り 3 件は今回変えていない `delegation_mode.go` のものであり、分類しない。
  ツールが表現できない「注入した供給元を無視して `crypto/rand` を直接読む」変異を手で入れた。`GenerateUserCode` に入れると `TestGenerateUserCode_DrawsEachCharacterFromTheSource` が、`Source.Int` に入れると `TestFixedIntRejectsValuesAtOrAboveTheBound` が失敗した。
- **検証結果**:
  - `mise run check-boundaries` - 成功
  - `mise run check-boundary-debt-ratchet -- main` - 成功（domain-effect 12 → 0。基準の数は検査を狭めた後の数え方による）
  - `mise run lint-go` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 成功（39 件）
