---
depends_on: []
status: completed
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-08-29
priority: p2
change_kind: bugfix
evidence_policy: risk-based-v2
initial_context:
  specification: [docs/contexts/provisioning/standards.md]
  typespec: [IdMagic.Contract.ProvisioningAuthMethod]
  source:
    - backend/provisioning/module.go
    - backend/provisioning/domain/connection.go
    - backend/provisioning/client_scim/client.go
    - backend/provisioning/client_scim/transport.go
    - backend/provisioning/db_postgres/repositories.go
    - infra/schema/postgres.sql
  tests:
    - backend/provisioning/client_scim
    - backend/provisioning/db_memory
  stop_before_reading: [frontend, backend/oauth2, backend/authentication]
affected_spec:
  - { path: docs/contexts/provisioning/standards.md, requirement: RFC7644-OUT-AUTHENTICATION }
  - { path: spec/contexts/provisioning/models.tsp, symbol: IdMagic.Contract.ProvisioningAuthMethod }
---

# 接続が受け付ける `oauth2_client_credentials` を、実際にトークンを取って使う

## Motivation

`ProvisioningAuthMethod` は `bearer_token` と `oauth2_client_credentials` の 2 値を受け付け、接続の登録画面も API もこの 2 つを提示する。ところが送出クライアントを組み立てる経路は認証方式を一切見ておらず、保存した資格情報をそのまま `Authorization: Bearer` の値として送る。

`oauth2_client_credentials` を選んだ接続では、これはクライアントシークレットをベアラートークンとして下流へ提示することを意味する。**下流が正しく実装されていれば必ず 401 になり、正しく実装されていなければシークレットがアクセストークンとして通ってしまう。** どちらも望ましくない。

さらに、401 は `RFC7644-OUT-ERROR-RESPONSE` の「再試行しない失敗」に落ちるので、管理者が見るのは配信の `dead_letter` だけで、「この認証方式は動かない」という事実はどこにも現れない。設定できるのに動かない選択肢が、失敗の原因として自分を名乗らないまま残っている。

[docs/contexts/provisioning/standards.md](../../docs/contexts/provisioning/standards.md) の `RFC7644-OUT-AUTHENTICATION` は、この食い違いを避けるために「認証は `Authorization: Bearer` の 1 方式に限り、資格情報を得るための別の要求は送らない」という真の宣言を置いている。この work item が直せば、その行は方式ごとの分岐を書けるようになる。

## Scope

- クライアント資格情報フロー (RFC 6749 §4.4) でアクセストークンを取得し、下流への要求に使う。
- トークンエンドポイント、スコープ、クライアント認証方式を接続の設定として持つかどうかを決める。いまの資格情報モデルには `client_id` と `client_secret` に相当する欄しか無い。
- 取得したトークンの有効期限内での再利用と、期限切れおよび 401 での再取得を決める。
- 実装できないと決めるなら、逆に `ProvisioningAuthMethod` から値を取り除く。**受け付ける値と動く値を一致させることが目的であり、方向はどちらでもよい。**
- `RFC7644-OUT-AUTHENTICATION` の `Statement` と証拠テストを更新する。

## Out of Scope

- 相互 TLS、署名付き JWT クライアント認証などの他のクライアント認証方式。
- 下流ごとの独自の認証方式。
- 資格情報の保存方式そのもの。既存のエンベロープ暗号化を使う。

## Design

**実装する側を取る。** 受け付ける値と動く値を一致させる方向は 2 つあり、費用は取り除く側のほうが小さい。それでも実装を選んだのは、`oauth2_client_credentials` が下流 SCIM 連携で最も普通の認証方式であり、取り除けば「bearer token を発行できない下流とは繋げない」という制約が製品に残るためである。

### 何が足りていないか

トークン取得が無いことに加えて、**トークンエンドポイントの URL・`client_id`・`scope` は API が受理して検証したあと、どこにも保存されずに捨てられている**。`ProvisioningCredentialInput` は 4 つの欄を持ち `admin.go` が必須検査までするのに、`Secret()` が返すのは `client_secret` だけで、残り 3 つを載せる先が集約にも表にも無い。したがってこの work item は永続化の追加を含む。

### 型と効果の境界

保存する 3 つは秘密ではないので、秘密の投影である `ProvisioningConnectionCredentialMetadata` に置く。表には列を 3 つ足す (`credential_oauth2_token_url` / `credential_oauth2_client_id` / `credential_oauth2_scope`)。`client_secret` は既存のエンベロープ暗号化をそのまま使い、保存先も `credential_secret` のままとする。

トークン取得は次の 2 つに分ける。

- **計算 (`tokenGrant`)**: トークン応答の本文と現在時刻から、載せるべきアクセストークンと失効時刻を決める純関数。`expires_in` の欠落、0 以下、極端に大きい値をここで正規化する。
- **作用 (`oauth2TokenSource`)**: トークンエンドポイントへの POST と、取得したトークンの保持。時刻は `now func() time.Time` として入力に置き、HTTP クライアントは注入する。

`ports.ProvisioningTargetClient` の形は変えない。`client_scim.Client` が `BearerToken string` を持っていたところを、要求ごとに値を返す `tokenSource` に替える。`bearer_token` の接続は固定値を返す実装、`oauth2_client_credentials` の接続は取得・再利用・再取得を行う実装を渡す。こうすると送出側の 6 経路はどれも書き換えずに済み、認証方式の分岐は組み立て (`NewTargetClient`) の 1 か所に閉じる。

### 期限内の再利用と再取得

- 取得したトークンは失効時刻まで再利用する。安全余裕として 60 秒手前で切る (時計ずれと往復時間)。
- 下流が 401 を返したら、保持しているトークンを捨てて 1 度だけ取り直し、同じ要求を再送する。2 度目も 401 なら失敗として扱う。無条件に繰り返すと、資格情報そのものが誤っている接続が下流のトークンエンドポイントを叩き続ける。
- `expires_in` が無い応答は 1 度きりのトークンとして扱う (次の要求で取り直す)。RFC 6749 §5.1 は `expires_in` を推奨に留めているので、欠落を失敗にはしない。

### 秘密の取り扱い

取得したアクセストークンも `client_secret` も、ログ・監査記録・エラー本文へ出さない。トークン取得が失敗したときのエラーには下流の状態コードだけを載せ、応答本文は載せない (本文にトークンや秘密が含まれうる)。

検討した代替案:

- **列挙値 `oauth2_client_credentials` を取り除く**: 費用は小さく、`check-api-compat` の破壊的変更 1 件で済む。しかし bearer token を発行できない下流と繋げない制約が残る。上記の理由で採用しない。
- **トークンを表へ保存する**: 再起動をまたいで再利用できるが、アクセストークンという短命な秘密を増やすことになる。プロセス内の保持に留め、再起動後は取り直す。採用しない。
- **`client_secret` とは別に取得したトークンを暗号化して持つ**: 同上。プロセス内に留める。

## Plan

1. 実装するか取り除くかを決める。
2. 選んだ側の RED を置く。
3. 実装し、規範行を更新する。

## Tasks

- [x] T001 [Design] 実装する側を確定した (利用者の判断)。
- [x] T002 [Acceptance] トークンを取得して提示することの受け入れ検査を RED で置いた。
  `TestClient_OAuth2ClientCredentials_*` 6 件。標準行: `RFC7644-OUT-AUTHENTICATION`。
- [x] T003 [Domain] `tokenGrant` (純関数) と `oauth2TokenSource` (作用) を実装した。
  Unit RED: `TestTokenGrant_NormalizesExpiry`。
- [x] T004 [Adapters] `Client` の `BearerToken` を `tokenSource` に替え、401 での 1 度きりの
  取り直しを入れた。認証方式の分岐は `newTokenSource` の 1 か所に閉じた。
- [x] T005 [Infrastructure] トークン取得の設定 3 つを永続化した。列 3 つと CHECK 制約 1 つ。
  未リリースのため移行は不要。
- [x] T006 [Spec] `RFC7644-OUT-AUTHENTICATION` を `partial` から `required` にし、TypeSpec の
  `ProvisioningAuthMethod` と `ProvisioningConnectionCredentialMetadata` を更新した。
- [x] T007 [Verify] `mise run verify`、`mise run check-api-compat`、`mise run check-schema`。

## Verification

- `mise run check-spec`
- `mise run check-api-compat`
- `mise run test-go`
- `mise run verify`

## Risk Notes

リスクを `high` へ上げた。着手前は `medium` だったが、資格情報の取得・保持・提示に触れる変更であり、誤ると秘密をそのまま外部へ提示する側へ倒れうるためである。評価の表は認証と資格情報を強い行へ置いており、実際の帰結がそちらを指している。

`reversibility` は `reversible` とする。未リリースなので、この変更が決めた列とスキーマ制約を、リポジトリの外の誰かが既に保存・送信・信頼しているという状態が無い。撤回するなら列を落とすだけで済む。

**security**: 取得したアクセストークンはプロセス内にのみ保持し、表にもログにも監査記録にも出さない。トークン取得の失敗を報告するエラーは下流の状態コードだけを載せ、応答本文は載せない (本文にトークンや秘密が含まれうる)。トークンエンドポイントは運用者が入れた URL なので、SCIM の接点と同じ SSRF 検査 (`ValidateOutboundBaseURL` と再解決する dialer) を通す。知らない認証方式は fail-closed で拒否し、秘密をそのまま送る経路を残さない。

**compatibility**: 公開 API に対しては追加のみ (`ProvisioningConnectionCredentialMetadata` に 3 つの省略可能な欄)。`mise run check-api-compat` は破壊的変更を報告しない。列挙値は取り除いていないので、既存のクライアントが送る値はすべて引き続き受理される。

**migration**: 未リリースなので移行は要らない。列 3 つはいずれも `NOT NULL DEFAULT ''` で、新しい CHECK 制約 (`active` な `oauth2_client_credentials` は token URL と client_id を持つ) を満たさない行は、配備先のどこにも存在しない。`infra/schema/postgres.sql` をそのまま当てればよい。

一度リリースした後であれば話は別で、そのときは移行が要る。`auth_method='oauth2_client_credentials'` の既存行は token URL と client_id を持たず —— それらは以前 API が受理して捨てていたので埋め直す元データが無く —— CHECK 制約に掛かる。該当する接続を `disabled` にしてから当てることになる (修正前も 1 件も配信できていないので、失われる機能は無い)。この work item ではその状況に無いので、スクリプトは置かない。

**rollback**: コードを戻す場合、列 3 つと CHECK 制約は残しても古いコードは読み書きしないので害は無い (列は既定値を持つ)。落とし直す場合も、未リリースなので保存済みのデータを気にする必要は無い。

## Completion

- **Completed At**: 2026-08-30
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。`REQ-` シナリオは動いていない。変わったのは `RFC7644-OUT-AUTHENTICATION` の `Adoption` (`partial` → `required`) と `Statement`、TypeSpec の 2 モデルの記述、そして実装である。`oauth2_client_credentials` の接続が、クライアント資格情報フロー (RFC 6749 §4.4) で取得したアクセストークンを提示するようになった。それまでは保存した `client_secret` をそのままベアラートークンとして送っており、この方式の接続は 1 件も配信できていなかった。トークンは失効の 60 秒手前まで再利用し、期限切れと下流の 401 で取り直す。401 による取り直しは 1 度だけである。**永続化も併せて足した** —— トークン URL・`client_id`・`scope` は以前 API が受理して検証したあと捨てられており、保存先が集約にも表にも無かった。列 3 つと、`active` な oauth2 接続に token URL と client_id を要求する CHECK 制約 1 つを加えている。未リリースなので、制約に掛かる既存行は無く、移行は要らない。
- **Acceptance RED Evidence**:
  - **Test**: `TestClient_OAuth2ClientCredentials_FetchesAndPresentsAnAccessToken` ほか `TestClient_OAuth2ClientCredentials_*` 6 件 (`backend/provisioning/client_scim/oauth2_test.go`)
  - **Requirement**: N/A: 該当する `REQ-` シナリオは無い。規範は `docs/contexts/provisioning/standards.md` の標準行 `RFC7644-OUT-AUTHENTICATION` (MUST) と RFC 6749 §4.4 である。
  - **Observed Failure**: `undefined: newOAuth2TokenSource` / `undefined: oauth2ClientCredentials` (build failed)。振る舞いが存在しないことがコンパイル段階で現れた。
  - **Detection Reason**: 下流を、**発行したアクセストークンだけを受理する**サーバーとして組み立てる。`client_secret` をそのまま提示する実装はここで 401 になり、通らない。加えて `Authorization` の値を直接見て、秘密の文字列が含まれていないことを主張する —— 下流が緩い実装だった場合 (秘密がトークンとして通ってしまう場合) を、下流の応答とは独立に分けるためである。トークン取得の回数を数えるので、期限内の再利用と再取得を取り違えた実装も分かれる。
- **Unit RED Evidence**:
  - **Test**: `TestTokenGrant_NormalizesExpiry` (同ファイル)
  - **Requirement**: N/A: 上と同じ理由。
  - **Observed Failure**: `undefined: tokenGrant` / `undefined: tokenResponse` (build failed)。
  - **Detection Reason**: `tokenGrant` は時刻を入力に取る純関数なので、失効の判定を実時間に頼らず固定できる。`expires_in` の欠落・0・負・上限超過・`access_token` の欠落という 5 つの入力それぞれに対して、再利用可否と失効時刻を主張する。上限の主張は「上限ちょうど」を等値で見るので、切り方を変えた実装が落ちる。安全余裕の主張は境界の両側 (残り 70 秒と 50 秒) を見るので、余裕を外した実装と広げすぎた実装の双方が分かれる。
- **Change-Resistance Results**:
  変更した論理を系統的に変異させた。9 件のうち 8 件が検出され、1 件は等価変異である。
  M1 `newTokenSource` の oauth2 分岐を落として `bearer_token` と同じ扱いにする (**元の欠陥そのもの**) → 検出。
  M2 失効の安全余裕 (`tokenRefreshSkew`) を外す → 検出。
  M3a 負の `expires_in` を 0 に丸める処理を外す → **生存 (等価変異)**。負の寿命は `ExpiresAt` を過去にするので `usableAt` は偽になり、丸めた場合と観測できる差が無い。丸めを残しているのは `ExpiresAt` を無意味な値にしないためで、現在の接口からは識別できない。
  M4 下流が申告する寿命の上限を外す → 検出。
  M5 `access_token` が空の応答を受理する (fail-open) → 検出。
  M6 トークンエンドポイントの SSRF 検査を外す → 検出。
  M7 401 の取り直しを 1 度きりから無制限の繰り返しに変える → 検出 (**ただし失敗ではなく非停止として**)。常に 401 を返す下流に対して検査が終わらなくなり、`timeout 60` で打ち切られた。時間を判定に使っているのではなく、上限の欠如が停止性として現れている。
  M8 401 の取り直しを一切しない → 検出。
  M9 トークン取得の失敗のエラーに下流の応答本文を載せる (秘密の漏洩) → 検出。
  **方法が見つけたもの**: 最初の版では M1 と M4 が生存した。M1 は、6 件の受け入れ検査がすべて `newOAuth2TokenSource` を直接組み立てており、**認証方式から供給元を選ぶ分岐を 1 つも通っていなかった**ためである。元の欠陥そのものが素通りしていたことになる。`TestNewTokenSource_SelectsBySAuthMethod` を足して閉じた。M4 は、検査が `expires_in: 1<<40` を使っており、`time.Duration` のナノ秒で int64 が溢れて負に回り込むため「上限より前」が無条件に成立する**空虚な主張**になっていたためである。これは実装の欠陥でもあった (秒のまま上限と比べるよう直した)。検査は溢れない値・溢れる値・`int64` 最大値の 3 つを等値で見るように書き換えた。M7 も、最初は「誤った資格情報」の検査しか無く、その経路はトークン取得の段階で失敗するので繰り返しに到達していなかった。トークンは取れるが SCIM が常に 401 を返す下流を足して閉じた。
- **Verification Results**:
  - `mise run verify` - passed (exit 0)
  - `mise run check-spec` - ok (148 document(s), 333 operation(s), 845 TypeSpec symbol(s))
  - `mise run check-api-compat` - `no breaking changes` (公開 API へは省略可能な欄の追加のみ)
  - `mise run check-schema` - `schema convergence check passed`
  - `mise run lint-go` - 0 issues
  - `go test ./backend/provisioning/... -count=1` - 全 8 パッケージ ok (postgres の往復検査は実 DB に対して PASS)
  - `mise run spec-diff` - `no normative specification change against main`

## Follow-up

無し。未リリースのため、配備前に実行すべき移行は無い。
