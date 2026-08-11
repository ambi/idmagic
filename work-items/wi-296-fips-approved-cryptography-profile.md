---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-25
depends_on: []
change_kind: feature
initial_context:
  scl:
    SigningKeys:
      - models.SignatureAlgorithm
      - models.KeyProvider
    OAuth2:
      - interfaces.Token
      - models.OAuth2Client
    Authentication:
      - standards.NISTSP80063B4
  decisions:
    - decisions/ADR-003-jwt-signing-algorithm.md
    - decisions/ADR-075-per-tenant-signing-keys-and-key-provider.md
    - decisions/ADR-026-password-policy.md
  source:
    - backend/shared/security/passwords_argon2id
    - backend/signingkeys/keys_jose
    - backend/signingkeys/keys_vault
    - backend/cmd/internal/bootstrap
    - infra/docker
  tests:
    - backend/shared/security/passwords_argon2id
    - backend/signingkeys
  stop_before_reading:
    - frontend
affected_spec:
  - { path: spec/contexts/signing-keys/models.tsp, symbol: IdMagic.Contract.SignatureAlgorithm }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.Contract.Token }
---

# FIPS 承認暗号のみで動作する運転モード (FIPS profile) を導入する

## Motivation

IdMagic の暗号選択は現状「強度重視」で、FIPS 140-3 承認アルゴリズムに限定する運転モードが無い:

- パスワードハッシュに **Argon2id** を使う (`backend/shared/security/passwords_argon2id`)。
  Argon2id は暗号学的には優れているが **FIPS 承認されていない**。FIPS 環境で承認される
  パスワードベースの鍵導出は SP 800-132 の PBKDF2 (承認済みハッシュ + HMAC) である。
- JWT 署名は PS256 / ES256 ([[ADR-003-jwt-signing-algorithm]]) で、これは承認範囲内。
- HIBP の k-anonymity 実装で SHA-1 を使う ([[wi-86-hibp-sha1-static-analysis-exception]])。
  SHA-1 は署名用途では承認外だが、この用途は署名でも認証でもないため、
  FIPS モードでの扱いを明示的に判断する必要がある。
- Go の FIPS 140-3 モード (`GOFIPS140`) や BoringCrypto を使う配備手順が無く、
  ビルド・実行時にどの暗号モジュールが使われているか宣言できない。

これが問題になる場面は限定的だが、その市場では**必須要件**になる:

- 米国連邦・州政府調達 (FedRAMP)、防衛関連サプライチェーン、金融の一部規制、
  医療 (HIPAA の実装ガイド)、日本でも政府情報システム (ISMAP) で暗号モジュールの
  適合が問われる。
- **Keycloak は FIPS 140-2 モードを公式に提供している** (BouncyCastle FIPS +
  適合ドキュメント)。Okta / Entra はサービス側で FedRAMP 認定を取得している。
  つまり「FIPS モードが無い」ことは、この 3 者と並べたときの明確な欠落である。

本 WI は FIPS 認証を取得することではなく、**「承認済み暗号モジュールと承認済み
アルゴリズムのみを使って動作し、それを起動時に検証・宣言できるモード」**を作る。

## Scope

- **decision**:
  - 新規 ADR (FIPS 運転モード): 適用範囲 (署名・トークン・パスワード保存・
    セッショントークン・OTP・XML 署名)、承認アルゴリズムの許可集合、
    FIPS モードでのパスワードハッシュ (PBKDF2-HMAC-SHA256 系) と既存 Argon2id ハッシュの
    **共存・段階移行**方式 (既存ハッシュを検証時に判別し、次回ログインで再ハッシュする)、
    HIBP の SHA-1 用途の扱い (FIPS モードで HIBP を無効化するか、非暗号用途として許容するかを
    明示的に決定する)、Go の FIPS モジュール利用方法とビルド手順、
    起動時の自己検査 (fail-fast) と `/version` 相当での宣言、
    FIPS モードで無効化される機能の一覧 (承認外アルゴリズムを要求するクライアント設定の拒否) を記録する。
- **scl**:
  - `SigningKeys.models.SignatureAlgorithm` に FIPS モードでの許可集合を明記する。
  - `OAuth2.models.OAuth2Client` の署名アルゴリズム関連メタデータについて、
    FIPS モードで承認外の値を登録できないことを requires として書く。
  - `Authentication` にパスワードハッシュアルゴリズムの表現 (`password_hash_scheme`) を追加し、
    複数方式の共存と移行を状態として表現する。
  - `System` に FIPS モードの設定と起動時自己検査、モードの宣言を追加する
    ([[wi-103-startup-config-validation-and-reference]] の設定検証面と整合)。
  - `guarantees` に「FIPS モードで起動した場合、承認外アルゴリズムを使う経路は
    fail-closed で拒否される」を明文化する。
  - `scenarios`: FIPS モードで承認外署名アルゴリズムのクライアント登録が拒否される /
    FIPS モードで Argon2id ハッシュのユーザーがログインでき次回から PBKDF2 になる /
    FIPS モードで自己検査に失敗したら起動しない / 非 FIPS モードでは既存挙動が変わらない。
- **go**:
  - パスワードハッシュを**方式付き**にする。保存形式に方式識別子を持たせ (既存の
    Argon2id ハッシュを既存形式として判別できるようにし)、`PasswordHasher` port に
    PBKDF2 実装を追加する。検証は両方式に対応し、成功時に現行方式へ再ハッシュする。
  - FIPS モードのフラグ (環境変数) を追加し、bootstrap で
    (1) 承認外 adapter / アルゴリズム設定の拒否、(2) 暗号モジュールの自己検査、
    (3) 起動ログとバージョンエンドポイントでのモード宣言を行う。
  - 承認外アルゴリズムを要求する経路 (クライアント metadata、XML 署名アルゴリズム、
    OTP のハッシュ) を洗い出し、FIPS モードで fail-closed にする。
  - HIBP を ADR の決定に従って処理する (無効化する場合は
    `BREACHED_PASSWORD_CHECKER` が `hibp` のとき FIPS モードで起動を拒否する等)。
- **build / infra**:
  - FIPS 対応ビルドの手順を `justfile` に追加する (`just build-go-fips` 相当)。
    Go の FIPS 140-3 モジュールを使うビルドタグ / 環境変数を明示する。
  - Dockerfile に FIPS ビルドのターゲットを追加するか、ビルド引数で切り替える。
- **documentation**:
  - README の Configuration に FIPS モードの環境変数、FIPS モードで無効になる機能、
    ビルド手順、宣言の確認方法を追記する。
  - 「FIPS モードで動作する」ことと「FIPS 140-3 認証を取得している」ことの違いを明記する
    (誤解を招く表現を避ける)。

## Out of Scope

- FIPS 140-3 の第三者認証取得 (CMVP)。本 WI は承認済みモジュールを使って動作するモードを作る。
- TLS 終端の暗号スイート制限。ingress / プラットフォーム責務。
- FedRAMP / ISMAP の統制文書一式の作成。
- 承認外アルゴリズムの削除。非 FIPS モードでは Argon2id を既定として維持する
  (セキュリティ上は Argon2id が望ましいため、既定を変えない)。
- ハードウェア HSM の FIPS レベル要件。→ [[wi-32-kms-hsm-and-per-tenant-signing-keys]] の
  provider 選択の範囲。

## Plan

- **非 FIPS モードの既定を変えないのが原則**。Argon2id は Argon2id のままにする。
  FIPS は「規制のために選ぶ制約付きモード」であり、既定の強度を下げる理由にはならない。
  ADR にこの立場を明記する。
- **パスワードハッシュの方式付き化を最初に入れる**。これは FIPS 以外にも将来の
  アルゴリズム移行 (Argon2 パラメータ変更等) で必要になる基盤である。
  保存形式に方式識別子を持たせ、検証時に判別し、成功時に現行方式へ再ハッシュする。
  この「ログイン時に静かに移行する」パターンを最初のテストで固定する。
- **fail-closed で宣言する**。FIPS モードで承認外の設定が入っていたら**起動しない**。
  実行時に一部だけ承認外という状態は、規制上「FIPS モードで動いている」と言えないため、
  起動時 fail-fast にする。[[wi-103-startup-config-validation-and-reference]] の
  設定検証と同じ場所に置く。
- **承認外経路の洗い出しを網羅する作業として明示的にタスク化する**。署名・ハッシュ・
  乱数・KDF・XML 署名・OTP・HIBP の 7 面を列挙し、それぞれ FIPS モードでの扱いを決める。
  漏れがあると「FIPS モードなのに承認外を使っている」という最悪の状態になるので、
  一覧を ADR に残して検証可能にする。
- **HIBP は判断が必要な論点**。SHA-1 を使うが署名・認証用途ではない (漏洩パスワード
  データベースの検索プレフィックス)。厳密な運用では「承認外アルゴリズムの実装が
  バイナリに存在すること」自体を問われることもある。第一候補は「FIPS モードでは
  HIBP チェッカーを使えない (起動時に拒否)」とし、代替として
  ローカル辞書ベースのチェックを案内する。
- 未決定: Go の FIPS モジュールと BoringCrypto のどちらを使うか。Go 標準の FIPS 140-3
  モジュール (`GOFIPS140`) を第一候補とし、対象プラットフォームで動かない場合に
  BoringCrypto を検討する。着手時に Go のバージョンで確認する。

## Tasks

- [ ] T001 [Survey] 暗号使用箇所を 7 面 (署名 / パスワードハッシュ / 乱数 / KDF /
      XML 署名 / OTP / HIBP) で棚卸しし、各々の FIPS モードでの扱い案を作る。
      結果を ADR の下書きに反映する。
- [ ] T002 [ADR] FIPS 運転モードの ADR を起票する (適用範囲・許可集合・パスワードハッシュ
      移行・HIBP の扱い・ビルド方法・起動時自己検査・無効化される機能一覧)。
- [ ] T003 [Spec] SignatureAlgorithm の FIPS 許可集合、client metadata の requires、
      `password_hash_scheme`、System の FIPS 設定と自己検査、guarantee、scenario 4 件を
      追加し `just check-scl` を通す。
- [ ] T004 [Hash] パスワードハッシュを方式付きにする。既存 Argon2id ハッシュの判別、
      PBKDF2 実装の追加、検証成功時の現行方式への再ハッシュを実装する。
      RED: 既存 Argon2id ハッシュのユーザーが検証でき、成功後に現行方式へ移行する
      テストを先に書く (scenario `Authentication.password_hash_scheme_migration`) → GREEN。
- [ ] T005 [Persistence] ハッシュ列の形式変更を `infra/schema/postgres.sql` に反映する
      (方式識別子を含む形式を採るなら列は変えずに値の形式で表す)。RED: 既存行の
      読み込み互換テスト → GREEN。
- [ ] T006 [Mode] FIPS モードのフラグと bootstrap での fail-fast 検証、モード宣言
      (起動ログ + version エンドポイント) を実装する。RED: 承認外設定で起動が失敗する
      テスト → GREEN。
- [ ] T007 [Fail-closed] 承認外アルゴリズムを要求する経路 (client metadata / XML 署名 /
      OTP / HIBP) を FIPS モードで拒否する。RED: 各経路の拒否テスト → GREEN。
- [ ] T008 [Build] `justfile` に FIPS ビルドレシピを追加し、Dockerfile に FIPS ターゲット
      またはビルド引数を追加する。ビルド成果物が FIPS モジュールを使っていることを
      確認する手順を残す。
- [ ] T009 [Docs] README に FIPS モードの設定・制約・ビルド手順・「認証取得とは別」である
      ことを追記する。
- [ ] T010 [Verify] 下記 Verification を緑にする。FIPS ビルドで主要フロー
      (ログイン / トークン発行 / SAML SSO) が通ることを確認する。

## Verification

- `just check` / `just check-scl` / `just check-work-items` / `just check-ids`
- `just test-go` / `just test-go-race` / `just verify-go`
- `just build-go` および FIPS ビルド (新設レシピ) の両方が成功する
- 手動: (1) 非 FIPS モードで既存の Argon2id ユーザーがログインでき、挙動が変わらないこと、
  (2) FIPS モードで起動し、既存 Argon2id ユーザーがログインでき、次回から現行方式に
  なっていること、(3) FIPS モードで承認外署名アルゴリズムのクライアントを登録できないこと、
  (4) FIPS モードで HIBP を有効化した設定が起動時に拒否されること (ADR の決定に従う)、
  (5) version エンドポイントがモードを宣言すること、を確認する。

## Risk Notes

**パスワードハッシュの形式変更は認証の根幹に触る**。移行を誤ると全ユーザーが
ログインできなくなる。方式判別と再ハッシュを最初のテストで固定し、
既存ハッシュの読み込み互換をマイグレーションテストで守る。
FIPS モードで PBKDF2 を使うことは、Argon2id に比べて**オフライン総当たりへの耐性が下がる**。
これは規制準拠のための意図的なトレードオフであり、ADR に明記する。
非 FIPS モードの既定を変えないことで、一般利用者の強度は維持する。
「FIPS モードで動く」を「FIPS 認証済み」と誤読されると虚偽表示になる。README と
ADR で明確に区別し、マーケティング的な表現を避ける。
承認外経路の洗い出しに漏れがあると、モードの意味が失われる。7 面の棚卸しを
ADR に残し、将来の暗号追加時に見直す対象として明示する。
