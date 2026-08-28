---
status: completed
authors: [tn]
risk: low
reversibility: reversible
evidence_policy: risk-based-v2
created_at: 2026-08-29
priority: p1
depends_on: []
change_kind: tooling
initial_context:
  specification: []
  typespec: []
  source:
    - mise.toml
    - .github/workflows/idmagic-ci.yaml
    - .golangci.yml
    - .gitignore
    - renovate.json
    - tools/package.json
    - tools/biome.json
    - tools/tsconfig.json
    - frontend/package.json
    - frontend/tsconfig.app.json
    - go.mod
  tests: []
  stop_before_reading:
    - backend
    - frontend/src
    - spec
    - docs/contexts
spec_impact: { kind: none, reason: "検査と依存宣言だけを足す変更であり、製品の振る舞い、HTTP 契約、データの意味を変えない。検査を足した結果として実装の欠陥が見つかった場合は、個別の work item として切り出す。" }
---

# リポジトリが所有する全ソース種別に、宣言された検査を持たせる

## Motivation

ツールチェーン監査 (2026-08-29) で、**宣言されたツールは筋が通っている一方、宣言されていないソース種別が残っている**ことが分かった。Go、TypeScript、TypeSpec、SQL には固定版のツールとタスクがあるが、シェルスクリプト、GitHub Actions ワークフロー、Dockerfile、Markdown には何も無い。

危ないのは「無い」ことそのものより、**手元では動いているように見える**ことである。この監査を実行したマシンには `shellcheck`、`shfmt`、`gitleaks` がすべて入っていた。グローバルな mise 設定が入れたものであって、リポジトリは何も要求していない。新しいチェックアウトでも CI でも、これらは一度も走らない。実際、リポジトリ文脈でシェルの検査を試みると shim がバージョン未設定で失敗する。**開発者の手元に入っていることは、プロジェクトの依存を満たさない。**

`.markdownlint.yaml` はこの構図の縮図である。設定ファイルだけが存在し、`markdownlint` 本体もタスクも無い。設定があるのだから検査されている、と読める状態が一番悪い。

未検査のまま残っているものを一度測った。いずれも実在する指摘であり、放置しても増え続ける。

| 対象 | ツール | 現状の指摘数 |
|---|---|---|
| シェルスクリプト 8 本 | shellcheck 0.11.0 | 12 件 (SC2155 が 8 件、SC2030/SC2031 が 4 件) |
| シェルスクリプト 8 本 | shfmt 3.13.1 (`-i 2 -ci`) | 2 ファイルが未整形 |
| ワークフロー 1 本 | actionlint 1.7.12 | 0 件 |
| ワークフロー 1 本 | zizmor 1.29.0 | 13 件 (`unpinned-uses` 8、`artipacked` 3) |
| Dockerfile 2 本 | hadolint 2.15.1 | 0 件 |
| Markdown (履歴を除く) | markdownlint-cli2 0.23.2 | 121 件 (MD034 が 85、MD001 が 23) |
| 作業ツリー全体 | betterleaks 1.8.1 | 既定の信頼度で 146 件、うち説明の付かないものは無し |

`unpinned-uses` 8 件と `artipacked` 3 件は、アイデンティティ基盤のリポジトリとして見過ごせない。CI のアクションはタグ固定であって内容固定ではなく、`actions/checkout` は既定で `.git/config` に資格情報を残す。

これに加えて、検査そのものではないが同じ監査で見つかった再現性の欠落を本 work item で閉じる。`tools` の依存は `latest` 指定が 6 件あり、導入も非 frozen な `bun install` である。同じリポジトリの `frontend` は exact 指定と `--frozen-lockfile` で運用されており、方針が 2 つ並んでいる。また `mise install` から始まる導入手順はあるが、**mise 自体の入手方法と版の制約がどこにも書かれていない**。mise は自分自身を bootstrap できない。

## Scope

- **シェル。** `shellcheck` と `shfmt` を `mise.toml` の `[tools]` に宣言する。既存の指摘 12 件と未整形 2 ファイルを直す。
- **GitHub Actions。** `actionlint` と `zizmor` を宣言する。`unpinned-uses` に対しては `.github/zizmor.yml` で所有者別の方針を置き、GitHub 自身が持つ `actions/*` と `github/*` はタグ運用を許し、それ以外の発行者にはコミットハッシュ固定を要求する。`artipacked` に対して `persist-credentials: false` を置く。actionlint は `run:` を shellcheck に渡すので、同じ mise の `PATH` 上で shellcheck が解決できることを確かめる。
- **Dockerfile。** `hadolint` を宣言し、2 本とも対象にする。現時点の指摘は 0 件であり、これは「今の状態を固定する」検査になる。
- **Markdown。** 検査を持たせない。宙に浮いていた `.markdownlint.yaml` を削除して、設定だけが存在する状態を解消する。判断の経緯は Out of Scope に書いた。
- **秘密情報。** `betterleaks` を宣言し、作業ツリーの走査を `lint-repo` に、履歴の走査を `audit-secrets-history` に置く。既知の値は `.betterleaks.toml` の allowlist で、1 件ずつ理由を書いて許可する。
- **束ね方。** 上の読み取り専用の検査を `lint-repo`、書き込む側を `format-repo` にまとめる。言語ごとにタスクを分けない。
- **`tools` の依存の再現性。** `latest` 指定 6 件をロックファイルが解決している版に固定し、`typescript` を `peerDependencies` から `devDependencies` へ移し、`setup-tools` を `bun install --frozen-lockfile` にする。両 `package.json` に `packageManager` を書く。
- **フロントエンドの型検査範囲。** `frontend/tests/**` と `frontend/scripts/**` を型検査に入れる。`tsconfig.app.json` は `include: ["src"]` のままビルドが使い、検査用の設定を別に持たせて `typecheck-ui` をそこへ向け、`verify` と CI に入れる。
- **lint の範囲漏れ。** `load/k6/oauth-smoke.js` を Biome の対象に入れる。`tools/biome.json` の除外パス `infra/observability/…` と `infra/load-tests/k6` は存在しないパスを指しているので消す。
- **bootstrap。** `mise.toml` に `min_version` を書き、`docs/development/local-development.md` に mise 自体の入手手順を書き、CI の `jdx/mise-action` に mise の版を渡す。
- **細部。** `GOLANGCI_LINT_CACHE` の `/tmp` 固定を外して既定に戻す。`.gitignore` に残る `just dev` を現在のタスク名に直す。`renovate.json` に最小の方針を書く。`verify` の Go テストを CI と同じ race 付きに揃える。
- 追加した検査はすべて `verify` と `verify-serial` と CI に配線する。個別にも呼べる状態を保つ。

## Out of Scope

- **infra アセットの CI 検証。実行コストに見合わないと判断し、意図的に見送る。** `check-compose` / `check-k8s` / `check-monitoring` / `check-k6` は、いずれも Docker で kustomize、kubeconform、promtool、k6 のイメージを引いてから実行する。1 回あたりの所要は他の全検査の合計を超え、CI では毎 PR でイメージ取得が発生する。得られるのは、変更頻度の低い宣言的アセットの構文検査である。**頻度の低い入力に、最も重い検査を毎回払う形になるので採らない。** これらのタスクは `mise tasks` から個別に呼べる状態を維持し、infra を触ったときに手で走らせる。この判断は work item が完了しても残るので、[継続的インテグレーション](../../docs/development/continuous-integration.md) に「CI で検証しないもの」として理由ごと明記する。
- **依存の脆弱性検査 (govulncheck、osv-scanner、Trivy)。** [[wi-291-dependency-vulnerability-management-and-disclosure-policy]] が持つ。同じ範囲を 2 つの work item で扱わない。CI にコメントアウトで残っている container + Trivy ジョブの扱いも、その work item の判断に委ねる。
- **SBOM と成果物の署名。** [[wi-100-supply-chain-sbom-cosign-slsa-provenance]] が持つ。
- **Markdown の書式検査。** 導入して `--fix` の結果を読んだうえで採らないと決めた。理由は Design に書いた。`.markdownlint.yaml` は消す。リンクと参照の整合性は `check-links` が持ち続ける。
- **検査を足した結果として見つかる実装の欠陥の修正。** 見つかれば個別に切り出す。検査の導入と欠陥の修正を同じ変更に混ぜると、後からどちらが何を意味したのか読めない。
- **`.stitch/DESIGN.md` の整形。** 外部ツールが生成する文書なので lint の対象にしない。

## Design

### ツールをどの層が所有するか

監査と同じ層の分け方をそのまま使う。**版を自然に所有する最も低い層に、1 つだけ宣言を置く。**

| ツール | 版 | 置き場所 | 理由 |
|---|---|---|---|
| shellcheck | 0.11.0 | `mise.toml` `[tools]` | リポジトリの検査が使う外部バイナリで、どのパッケージグラフにも属さない |
| shfmt | 3.13.1 | `mise.toml` `[tools]` | 同上 |
| actionlint | 1.7.12 | `mise.toml` `[tools]` | 同上 |
| zizmor | 1.29.0 | `mise.toml` `[tools]` | 同上 |
| hadolint | 2.15.1 | `mise.toml` `[tools]` | 同上 |
| betterleaks | 1.8.1 | `mise.toml` `[tools]` | 同上 |

k6 スクリプトの Biome は、すでに `tools` の `devDependencies` にある同じ Biome を使う。エコシステム層が版を持っているものを、mise に二重に宣言しない。

`aqua` backend を使う。すでに golangci-lint、sqlc、psqldef が同じ形で入っており、checksum と GitHub の artifact attestation の検証が付く (betterleaks は cosign で検証される)。

### 秘密情報スキャナの選定

**betterleaks を採る。** 確立したスキャナがまだ無いので移行の制約が無く、redact 付きの作業ツリー検査が既定で使える。gitleaks の設定形式と互換 (`.gitleaks.toml` も読む) なので、後で乗り換える余地も残る。

**信頼度の閾値では絞らない。既定のまま走らせ、既知のものだけを allowlist で消す。**

着手時は `--confidence medium` を採る予定だった。既定では 146 件、`medium` で 8 件、`high` で 1 件になり、既定の 146 件はほぼ `generic-password` — テストの固定値、`*.i18n.ts` の UI 文言、ビルド生成物 — だったからである。**この判断は実装中に取り消した。** 検査が本物を見つけるかを確かめるため、AWS のアクセスキーの形をした文字列をリポジトリに置いて走らせたところ、既定では検出され、`--confidence medium` では**報告されなかった**。ここでの信頼度は重大度ではなく検出器の確信度であり、一般的な形の資格情報ほど下に落ちる。閾値で件数を減らすことは、まさに見つけたいものを見えなくすることだった。

そこで既定の信頼度のまま、`.betterleaks.toml` の allowlist で消す。生成物のディレクトリ、ローカル開発の固定資格情報、デモスクリプトの Basic 認証、そして「パスワード」という語を含むだけの文字列 (UI 辞書、テスト、TypeSpec のフィールド名、ロックファイル、文書) と SAML/ACR の URN を、それぞれ理由付きで許可する。**146 件を baseline ファイルに固める道は採らない。** 検査を導入したという記録を作るだけで、次に混ざる 1 件を見つけられない。

許可の範囲が広すぎないことは、AWS のアクセスキーと、通常の Go ファイルに置いた高エントロピーのパスワードの 2 つを仕込み、どちらも報告されることで確かめる。

**資格情報の生存確認 (live validation) は有効にしない。** 検出した文字列を外部サービスへ送る動作であり、依頼されていない。履歴走査は `audit-secrets-history` として明示的に呼ぶタスクに分け、毎 PR のゲートには入れない。

### Markdown の検査は入れず、宙に浮いた設定を消す

着手時は `markdownlint-cli2` を入れる予定だった。**実際に入れて `--fix` の結果を読み、入れない判断に変えた。**

導入直後の指摘は 121 件で、`--fix` を当てると 4 つの規則が正準文書を壊した。

- **MD001 (見出しの飛び)。** 23 件のうち 22 件は `docs/contexts/*/scenarios.md` の `# 見出し` → `### REQ-...` である。[SPECIFICATION_FORMAT.md](../../SPECIFICATION_FORMAT.md) 6 節が「シナリオの見出しは、上に何も無くても H3 のままにする。`### REQ-...` は参照、検査、生成されるアンカーがすべて前提にしている形である」と明文で定めている。規範どおりに書かれた正準文書 22 本を、違反として報告し続ける。
- **MD034 (裸の URL)。** `<…>` で囲む修正が、シナリオの中で**値として**書かれた URL まで囲んだ。`redirect_uri "https://app.example.com/callback"` が `redirect_uri "<https://app.example.com/callback>"` になる。規範文書に書かれたリテラルの値が変わる。
- **MD037 (強調記号の中の空白)。** 日本語の文中の `WS-* が` を `WS-*が` に詰めた。`*` を強調の開始と誤認しており、**日本語の助詞の前の空白を勝手に削る**。
- **MD029 (番号付きリストの接頭辞)。** コードブロックを挟んで続く手順の `4. 5. 6. 7.` を `1. 2. 1. 1.` に振り直し、手順書の番号を壊した。

**4 つを無効化して残す道も試したが、採らなかった。** 無効化したあとに残る本物の指摘は、言語指定の無いコードブロック 3 件、見出しの飛び 1 件、余分な空行 1 件だけである。**この 5 件のために、この リポジトリの日本語の文書に対して壊れた修正を出す道具を常設し、規則ごとの例外の一覧を維持し続ける取引が見合わない。** 検査を入れた副作用で正準文書の中身が変わるのは、検査が無い状態より悪い。当てた `--fix` の結果は全部差し戻した。

代わりに、監査が指摘した「設定だけが宙に浮いている」状態そのものを解消する。`.markdownlint.yaml` は本体もタスクも無いまま置かれており、**検査されているという誤解を生む**のがこの指摘の中身だった。道具を入れないと決めた以上、設定を消すのが正しい解消である。

Markdown のリンクと参照の整合性は、この リポジトリが自分で持つ `mise run check-links` が 613 文書に対して見ており、そちらは残る。

### 検査の配線

**新しい検査は `lint-repo` 1 つに束ねる。** 着手時は `verify` の平らな依存の形に合わせて言語ごとにタスクを分ける予定だったが、そうすると 12 個増えて `mise tasks` が読めなくなる。**この判断は実装中に取り消した。**

分けない根拠は所要時間である。shellcheck、shfmt、actionlint、zizmor、hadolint、Biome、betterleaks を合わせても数秒で終わり、Go の 1 パッケージのテストより安い。[検証のはしご](../../docs/development/specification-first-workflow.md#5-verification-ladder) が「最も狭いタスクから」と言うのは、狭くすることに実際の時間差があるからであって、ここには差が無い。差の無いところで選択肢を増やすと、はしごが読みにくくなるだけである。

`lint-repo` は**最初の失敗で止めず、落ちた検査を全部並べてから終わる**。1 つ直すたびに走らせ直して次の 1 つを知る往復は、束ねたことで生まれる形の待ち時間なので、そこは埋める。

書き込む側は `format-repo` 1 つ。履歴の秘密情報監査 (`audit-secrets-history`) だけは重く、毎回のゲートに入れるものではないので分けて残す。

CI では既存の `verify` ジョブに 1 ステップとして足す。`check-command-map` がワークフローの呼び出しと `mise` タスクの整合を機械検査しているので、タスク名の食い違いはそこで落ちる。

### フロントエンドの型検査

`tsconfig.app.json` の `include` を広げず、検査専用の設定を追加して `src` と `tests` と `scripts` を含める。ビルド (`bun run build`) は今までどおり `tsconfig.app.json` だけを見る。**ビルドの型検査にテストを混ぜると、テストの型エラーで製品のビルドが落ちる**ことになり、原因と結果が離れる。`typecheck-ui` を新しい設定に向け、`verify` と CI に入れる。

### 却下した選択肢

- **markdownlint を、壊れる 4 規則を無効化したうえで残す。** 無効化のあとに残る本物の指摘は 5 件だけである。その 5 件のために、日本語の文書に対して壊れた修正を出す道具と、規則ごとの例外の一覧を常設する取引が見合わない。
- **shellcheck / shfmt をグローバルな mise 設定に任せる。** まさに今の状態であり、動いているように見えて新しいチェックアウトでは走らない。この work item の出発点そのものなので採らない。
- **`tools` の `latest` 指定をロックファイルに任せたまま残す。** ロックファイルがある限り日々は動く。問題はロックを更新した瞬間に、意図しない major を含む任意の版へ飛ぶことと、`bun install` が非 frozen なので CI がロックを黙って書き換えうることである。`frontend` と方針が違う理由が無い。
- **`verify` の Go テストを非 race のまま残す。** 手元は速いままだが、CI は race で走る。手元で緑、CI で赤という差が残るのは、[CONTRIBUTING.md](../../CONTRIBUTING.md) が避けようとしている形そのものである。

## Plan

1. `mise.toml` にツールと `min_version` を宣言し、`mise install` が通ることを確かめる。**この時点でタスクはまだ足さない。** 宣言だけを先に入れて、各ツールが単体で動く状態を作る。
2. 検査タスクを 1 つずつ追加し、追加した直後にそのタスクが**現時点の指摘で赤くなる**ことを見る。赤を見ずに修正から入ると、その検査が本当に対象を見ているのか分からない。
3. 指摘を種別ごとに直す。シェル → ワークフロー → Dockerfile → Markdown の順にする。前 3 つは件数が小さく、Markdown が最も重いので最後に置く。
4. 秘密情報スキャンを入れ、`.betterleaks.toml` の allowlist を 1 件ずつ理由付きで書く。
5. 再現性の修正 (`tools` の依存、`packageManager`、フロントエンドの型検査設定、Biome の範囲) を入れる。
6. 細部 (`GOLANGCI_LINT_CACHE`、`.gitignore`、`renovate.json`、`verify` の race) を直す。
7. `verify` / `verify-serial` / CI へ配線し、[継続的インテグレーション](../../docs/development/continuous-integration.md) に「CI で検証しないもの」を書く。
8. `mise run verify` を通し、作業ツリーに書き戻しが無いことを確かめる。

## Tasks

- [x] T001 [Spec] 仕様影響なし。`spec_impact` の理由が実装後も成り立つことを確認した。
- [x] T002 [Tooling] shellcheck / shfmt / actionlint / zizmor / hadolint / betterleaks を `mise.toml` に宣言し、`min_version` を書いた。`mise install` と `mise run tool-versions` が通る。
- [x] T003 [Tooling] シェルの検査を追加し、**赤を確認してから** shellcheck の 12 件と shfmt の 2 ファイルを直した。`dev.sh` の SC2030/SC2031 は部分シェルに閉じた意図した export なので、理由を書いて個別に無効化した。
- [x] T004 [Tooling] ワークフローの検査を追加し、赤 (zizmor 13 件) を確認してから `.github/zizmor.yml` の所有者別方針、サードパーティのハッシュ固定、`persist-credentials: false` を入れた。actionlint が同じ `PATH` の shellcheck を解決することを、引用漏れの `run:` を一時的に入れて確かめた。
- [x] T005 [Tooling] Dockerfile の検査を追加した。既存の指摘は 0 件なので、DL3008/DL3009/DL3015 を一時的に入れて落ちることを確かめてから戻した。
- [x] T006 [Tooling] `markdownlint-cli2` を入れて赤 (121 件) を確認し、`--fix` の結果を読んだ。MD001 / MD034 / MD037 / MD029 の 4 つが正準文書を壊すため、**道具ごと採らない判断に変えた。** 当てた修正は全部差し戻し、宙に浮いていた `.markdownlint.yaml` を削除した。
- [x] T007 [Tooling] `betterleaks` と `.betterleaks.toml` を入れた。信頼度の閾値では絞らず、allowlist に 1 件ずつ理由を書いた。仕込んだ 2 種類の秘密が報告されることを確かめた。
- [x] T008 [Tooling] `tools` の `latest` 6 件を固定し、`typescript` を `devDependencies` へ移し、`setup-tools` を `--frozen-lockfile` にし、両 `package.json` に `packageManager` を書いた。
- [x] T009 [Tooling] `frontend/tsconfig.check.json` を追加して `typecheck-ui` を向け直し、露出した e2e の型エラー 3 件を直した。
- [x] T010 [Tooling] `load/biome.json` で k6 スクリプトを Biome の対象に入れ、`tools/biome.json` の存在しない除外パスを消した。
- [x] T011 [Tooling] `GOLANGCI_LINT_CACHE` の `/tmp` 固定を外し、`clean-lint-cache` を `golangci-lint cache clean` に直した。`.gitignore` の `just dev` と `renovate.json` を直し、`verify` を race 付きに揃えた。
- [x] T012 [Docs] mise 自体の入手手順を [ローカル開発](../../docs/development/local-development.md) に書き、CI の `jdx/mise-action` に mise の版を渡した。
- [x] T013 [Docs] [継続的インテグレーション](../../docs/development/continuous-integration.md) に「CI で検証しないもの」として infra アセットの判断と理由を書いた。
- [x] T014 [Verify] 検査を `lint-repo` / `format-repo` に束ねて `verify` / `verify-serial` / CI に配線し、`mise run verify` を通した。`git status` に生成物の書き戻しが無いことを確かめた。

## Verification

### 着手前に宣言する RED

製品の振る舞いを変えないので、観測境界の受け入れ検査も単体検査も存在しない。代わりに**追加した検査そのものを RED として使う**。検査を足した直後に、既存のリポジトリに対してその検査が落ちることを見る。落ちなければ、その検査は対象を見ていない。

- **Acceptance RED の代替**: `mise run verify` が、新しい検査を配線した直後に落ちること。落ちる内訳が Motivation の表と一致すること。
- **Unit RED の代替**: 各検査を導入した直後の失敗。検査を束ねる前の段階で 1 つずつ観測する。
  - shellcheck — 12 件
  - shfmt — 未整形 2 ファイル
  - zizmor — 13 件
  - markdownlint — 121 件 (評価のためだけに観測した。道具は最終的に採らなかったので、`lint-repo` には残っていない)
  - `mise run typecheck-ui` — `tests` と `scripts` を範囲に入れた結果の型エラー (件数は実測)
  - hadolint と actionlint — 現状 0 件で、**RED にならない 2 つ**である。既知の違反を一時的に入れて落ちることを確かめ、戻す。緑を確かめずに緑を信じない。
  - betterleaks — allowlist を書く前の指摘
- **allowlist が広すぎないことの確認**: betterleaks の allowlist を書いたあと、AWS のアクセスキーと、通常の Go ファイルに置いた高エントロピーのパスワードを仕込み、どちらも報告されること。
- **束ねたあとの確認**: `mise run lint-repo` が、2 つの検査を同時に落としたときに両方を報告すること。

### 完了時に通すもの

- `mise run verify`
- `mise run check-command-map` (ワークフローの呼び出しと `mise` タスクの整合)
- 追加した各タスクを個別に実行し、**修正前に赤、修正後に緑**であったことを記録する。
- `git status --porcelain` が空であること。検査タスクが作業ツリーへ書き戻さないことの確認になる。
- 新しいチェックアウトに相当する経路として `mise install` と `mise run setup` を実行し、`bun install --frozen-lockfile` がロックファイルを書き換えないことを確かめる。

## Risk Notes

- **検査の追加が既存の CI を止める。** 指摘を直してから配線するので、順序を守れば起きない。逆順にすると main が赤くなる。
- **秘密情報スキャンの誤検出で作業が止まる。** `--confidence medium` と allowlist で今日の 8 件は説明が付くが、新しいテスト固定値が引っかかることは今後もある。allowlist に足すときは理由を必ず書き、閾値を下げて黙らせない。
- **アクションのハッシュ固定で更新が止まる。** タグ追従を捨てるので、更新は `renovate.json` が拾う必要がある。T011 の renovate 設定で GitHub Actions の manager が有効であることを確かめる。
- **Markdown の書式修正が、日本語の文書の意味を変える。** これは想定した危険ではなく、実際に起きたことである。詳細は Design に書いた。道具を採らないことで解消したが、将来 Markdown の検査を再検討するときは、規則を有効にする前に `--fix` の結果を 1 件ずつ読むところから始めること。
- **`--frozen-lockfile` 化が手元の作業を止める。** `tools/package.json` を編集したあと `mise run setup` が失敗するようになる。これは意図した動作で、依存を足したときは `bun install` を明示的に呼ぶ。`frontend` はすでに同じ運用である。

## Completion

- **Completed At**: 2026-08-29
- **Summary**:
  `mise run spec-diff` は空である。規範も TypeSpec も変えていない。意味上の差は、リポジトリが所有していながら検査を持っていなかったソース種別 — シェルスクリプト 8 本、GitHub Actions ワークフロー、Dockerfile 2 本、k6 モジュール — が、固定版のツールによる検査を持ったことである。Markdown だけは道具を入れて評価した結果、採らない判断に至り、宙に浮いていた設定を消した。加えて、作業ツリーの秘密情報スキャンが無い状態から有る状態になり、`tools` の依存が `latest` から固定版と `--frozen-lockfile` になり、`frontend/tests` と `frontend/scripts` が型検査の範囲に入った。CI で検証しない infra アセットの判断は、`docs/development/continuous-integration.md` に理由ごと残した。
- **Acceptance RED Evidence**:
  - **Test**: `mise run verify`
  - **Requirement**: N/A: 検査と依存宣言だけを足す変更であり、対応する規範シナリオが無い。
  - **Observed Failure**: 新しい検査を配線した直後、`lint-repo` が `failed: shellcheck shfmt zizmor betterleaks` で落ち、`typecheck-ui` が e2e の型エラー 3 件で落ちた。
  - **Detection Reason**: 検査そのものを RED として使った。検査が対象を見ていなければ、既存のリポジトリに対して緑になる。実際に落ちた内訳が、着手前に測った件数 (shellcheck 12、shfmt 2 ファイル、zizmor 13、betterleaks 8) と一致したので、各検査がこのリポジトリのファイルを読んでいることが確かめられた。
- **Unit RED Evidence**:
  - **Test**: hadolint と actionlint への故障注入
  - **Requirement**: N/A: 上と同じ理由。
  - **Observed Failure**: hadolint と actionlint は導入時点で指摘 0 件であり、RED にならない 2 つだった。`infra/docker/Dockerfile` に版を固定しない `apt-get install` を一時的に足すと DL3008 / DL3009 / DL3015 で落ち、ワークフローに引用漏れの `run: echo $HOME/probe` を足すと `shellcheck reported issue in this script: SC2086` で落ちた。どちらも戻して緑に復帰した。
  - **Detection Reason**: 緑を確かめずに緑を信じないため。後者は同時に、actionlint が mise の `PATH` 上の shellcheck を解決していることの証拠でもある。宣言前は同じ呼び出しが `No version is set for shim: shellcheck` で失敗していた。
- **Independent Verification**:
  `risk: low` / `reversibility: reversible` のため要求されない。
- **Change-Resistance Results**:
  `risk: low` のため要求されないが、秘密情報スキャンについては allowlist が検査を無効化していないことを確かめた。AWS のアクセスキーの形をした文字列を追跡外のファイルに、高エントロピーのパスワードを通常の Go ファイルに仕込むと、それぞれ `generic-api-key` と `generic-password` として報告された。**この確認が設計判断を 1 つ取り消した。** 当初予定していた `--confidence medium` では前者が報告されず、閾値が「見つけたいもの」を消していた。既定の信頼度に戻し、allowlist で絞る形に変えた。
  `lint-repo` が最初の失敗で止まらないことも、引用漏れのシェル関数と版を固定しない `apt-get install` を同時に仕込み、`failed: shellcheck hadolint` を得ることで確かめた。
  markdownlint については逆向きの確認をした。`--fix` が出した修正を 1 つずつ読み、MD001 / MD034 / MD037 / MD029 の 4 つが正準文書を壊すことを確かめた。**検査を入れた副作用で文書の中身が変わるのは、検査が無い状態より悪い。** 当てた修正は全部差し戻し、道具ごと採らない判断に至った。
- **Verification Results**:
  - `mise run verify` - passed
  - `git status --porcelain` - 意図した変更のみ。検査タスクによる書き戻しなし。
  - `mise install` / `mise run setup` - `bun install --frozen-lockfile` がロックファイルを書き換えないことを確認
