---
status: completed
authors: [tn]
risk: low
reversibility: reversible
evidence_policy: risk-based-v2
created_at: 2026-08-29
priority: p2
depends_on: [wi-433-repository-toolchain-covers-every-owned-source]
change_kind: tooling
initial_context:
  specification: []
  typespec: []
  source:
    - mise.toml
    - .github/workflows/idmagic-ci.yaml
    - dev.sh
  tests: []
  stop_before_reading:
    - backend
    - frontend
    - spec
    - docs/contexts
spec_impact: { kind: none, reason: "検査を減らす変更であり、製品の振る舞いも公開契約も変えない。" }
---

# 検査の数を、置いている理由を言えるものだけに戻す

## Motivation

[[wi-433-repository-toolchain-covers-every-owned-source]] は、所有していながら検査を持たなかったソース種別すべてに検査を入れた。**入れたこと自体は正しかったが、入れた数が多すぎた。** 7 つの道具のうち 4 つは、今後このリポジトリで鳴る見込みが無いか、鳴っても書式の話しかしない。

実行時間は問題ではない。7 つ合わせて約 800ms で、Go の 1 パッケージのテストより安い。効いているコストは 2 つである。**固定した版が 6 個増えて更新の Pull Request が回り続けること**と、**欠陥ではない指摘に手を動かさせられること**である。後者は実際に起きた。`dev.sh` には、情報レベルの誤検出を黙らせるためだけの抑制コメントが 2 か所入っている。

| ツール | 実行 | 導入後の指摘 | 判断 |
|---|---|---|---|
| betterleaks | 148ms | 0 | 残す。漏洩は取り消せない唯一の非対称な枠である |
| shellcheck | 281ms | 0 | 残す。ただし SC2030 / SC2031 だけ外す |
| zizmor | 63ms | 0 | 残す。実在する 11 件を出した唯一の道具である |
| actionlint | 51ms | 0 | 外す |
| hadolint | 115ms | 0 | 外す |
| shfmt | 20ms | 0 | 外す |
| Biome (k6) | 118ms | 0 | 外す |

## Scope

- `mise.toml` から `shfmt`、`actionlint`、`hadolint` の宣言と、`lint-repo` からその 3 つと k6 の Biome を外す。
- `shellcheck` から誤検出の SC2030 / SC2031 だけを外し、`dev.sh` の抑制コメント 2 か所を消す。
- `load/biome.json` を削除する。
- `format-repo` を削除する。中身は `shfmt` と k6 の Biome だけだったので、空になる。
- CI のステップ名を、実際に走るものに合わせる。

## Out of Scope

- `betterleaks`、`shellcheck`、`zizmor` の削減。残す理由は Design に書いた。
- すでに当てた `shfmt` の整形と k6 の Biome の整形の差し戻し。整形済みの状態そのものに害は無く、戻すほうが差分を増やす。今後は誰も強制しない。
- Go、TypeScript、TypeSpec、Markdown リンク、work item の各検査。wi-433 が触っていない既存の枠であり、本 work item の対象ではない。

## Design

外す 4 つは、それぞれ違う理由で「置いている理由を言えない」。

- **Biome (k6)。** 対象が `load/k6/oauth-smoke.js` 1 ファイルで、そのために 3 つ目の Biome 設定 `load/biome.json` を新設していた。構文の破綻は `mise run check-k6` が k6 自身のパースで検出する。1 ファイルのスタイル検査のために設定ファイルを維持する取引が成立しない。
- **shfmt。** 空白の揃え方だけを見る。**欠陥を 1 件も捕まえておらず、設計上これからも捕まえない。** 得るものは差分の見た目の統一で、払うものは「インデントで CI が赤くなる」である。
- **hadolint。** 主力の規則 (`apt-get` の版固定、`latest` タグの回避) が、`apt` を使わない distroless の 2 本には当たらない。故障注入では鳴ったが、それは注入した違反がこのリポジトリの書き方から外れていたからで、当たらない規則を持つ道具を常設していることの証明にしかならない。
- **actionlint。** **このリポジトリのワークフローが実際に壊れる壊れ方は、すでに自前の検査が持っている。** `check-command-map` が `.github/workflows/*` を読み、`mise run <task>` の呼び出しが `mise.toml` に存在することを検証している。ワークフローは 1 ファイル 90 行で、中身のほぼ全部が `mise run` の羅列であるため、現実的な誤りはタスク名の食い違いであり、そこは埋まっている。残る YAML 構文と式の誤りは push して CI を待てば分かる範囲であり、そのために道具を 1 つ常設する差にはならない。

残す 3 つの理由も、外した理由と同じ物差しで書いておく。

- **betterleaks。** 他の検査は見逃しても後から直せるが、資格情報の漏洩は取り消せない。0 件だから外す、という理屈がこの枠にだけ通らない。
- **shellcheck。** `infra/backup/*.sh` は災害復旧のスクリプトである。wi-433 で直した SC2155 は `readonly X="$(pg_dump ...)"` の形で、**中のコマンドの終了ステータスが握り潰される**という指摘だった。バックアップが失敗しているのに成功して見えるのは、まさにこの検査の守備範囲である。**誤検出の SC2030 / SC2031 だけを外す。** この 2 つは部分シェルに閉じた意図した `export` を誤検出し、`dev.sh` に抑制コメントを書かせた。

着手時は `-S warning` で情報レベルごと落とす予定だったが、**故障注入で取り消した。** 引用漏れの `echo $1` を仕込んでも shellcheck が鳴らず、SC2086 (変数の引用漏れ) が情報レベルにあることが分かった。**これは書式ではなく本物のバグ種別**で、深刻度で切ると誤検出と一緒に落ちる。誤検出の 2 つを名指しで外す形なら、SC2086 は残り、抑制コメントも消せる。
- **zizmor。** wi-433 で 11 件の実在する問題 (サードパーティアクションの可動タグ参照、`.git/config` に残る資格情報) を出した唯一の道具である。今後鳴るのはアクションを足したときだけで、**頻度が低く、security に効き、目視で間違えやすい**という、機械の検査が人のレビューに勝つ条件が揃っている。

### 却下した選択肢

- **actionlint を残す。** ワークフローの誤りを push 前に知れるのは確かに利点である。採らないのは、この リポジトリのワークフローに限れば `check-command-map` が同じ役割の大半を先に果たしているからで、一般論として actionlint が不要だという主張ではない。ワークフローが増えて式やマトリクスを持ち始めたら、その時点で戻す判断になる。
- **shfmt を書き込み専用 (`format-repo`) として残し、ゲートから外す。** ゲートでないなら誰も走らせず、走らせないなら整形は揃わない。中途半端に残すより、道具ごと外して「誰も強制しない」ことを明示するほうが正直である。

## Plan

1. 外す 3 つの宣言とタスク内の呼び出しを消し、`load/biome.json` を削除する。
2. `shellcheck` から SC2030 / SC2031 を外し、`dev.sh` の抑制コメントを消して、それでも緑であることを確かめる。
3. `lint-repo` が残る 3 つで緑になること、そして**故障注入で赤になること**を確かめる。減らした結果として何も見ていない状態になっていないかを、緑だけで判断しない。
4. CI のステップ名を合わせ、`mise run verify` を通す。

## Tasks

- [x] T001 [Tooling] `mise.toml` から `shfmt` / `actionlint` / `hadolint` を外し、`lint-repo` からその 3 つと k6 の Biome の呼び出しを消し、`format-repo` と `load/biome.json` を削除した。
- [x] T002 [Tooling] `shellcheck` から SC2030 / SC2031 を外し、`dev.sh` の抑制コメント 2 か所を消した。故障注入の結果、当初予定の `-S warning` から名指しの除外へ変えた。
- [x] T003 [Verify] `lint-repo` が緑であること、および残した 3 つが同時の故障注入で `failed: shellcheck zizmor betterleaks` になることを確かめた。CI のステップ名を合わせ、`mise run verify` を通した。

## Verification

### 着手前に宣言する RED

減らす変更なので、検査が落ちることではなく**落とすべきものを落とせるままであること**が要点になる。緑は減らしすぎたときにも出る。

- **Acceptance RED の代替**: `mise run verify` は変更の前後で緑のままである。ここに RED は無い。
- **Unit RED の代替**: 残した 3 つへの故障注入。引用漏れのシェル関数で shellcheck、可動タグのサードパーティアクションで zizmor、通常の Go ファイルに置いた高エントロピーのパスワードで betterleaks が、それぞれ赤になること。3 つとも赤にできなければ、減らしたのではなく壊したことになる。

### 完了時に通すもの

- `mise run verify`
- `mise run lint-repo`

## Risk Notes

- **減らしすぎて何も見ていない状態になる。** 緑では判別できないので、残した 3 つすべてに故障注入をして確かめる。
- **外した領域の劣化が誰にも見えなくなる。** シェルの整形、Dockerfile、ワークフローの構文は、今後この リポジトリの誰も検査しない。**これは意図した取引である。** 劣化が実際に痛みを生んだ時点で、その痛みを根拠に戻す。今戻す根拠は無い。
- **shellcheck の除外が本物の指摘を隠す。** 深刻度で切る形はこの危険が現実になり、故障注入で見つけて取り消した (Design を参照)。名指しで外した SC2030 / SC2031 は、部分シェルの中で完結する `export` に対してだけ出るもので、この リポジトリでの用法は 2 か所とも意図した局所化である。今後この 2 つが本物を指す場面が出たら、除外ではなく個別の抑制コメントへ戻す。

## Completion

- **Completed At**: 2026-08-29
- **Summary**:
  `mise run spec-diff` は空である。意味上の差は、[[wi-433-repository-toolchain-covers-every-owned-source]] が入れた 7 つの検査のうち、置いている理由を言えなかった 4 つ (shfmt、actionlint、hadolint、k6 の Biome) を外し、mise の固定バージョンが 6 個から 3 個に減ったことである。`load/biome.json` と `format-repo` は不要になったので削除した。shellcheck は誤検出 2 件を名指しで外し、`dev.sh` の抑制コメントを消した。
- **Acceptance RED Evidence**:
  - **Test**: `mise run verify`
  - **Requirement**: N/A: 検査を減らす変更であり、対応する規範シナリオが無い。
  - **Observed Failure**: N/A: 減らす変更なので、前後とも緑が期待値である。**緑は減らしすぎたときにも出るため、これ単独では証拠にならない。** 実際の証拠は下の故障注入に置いた。
  - **Detection Reason**: 上記のとおり、この境界では区別が付かない。
- **Unit RED Evidence**:
  - **Test**: 残した 3 つへの同時の故障注入
  - **Requirement**: N/A: 上と同じ理由。
  - **Observed Failure**: `demo.sh` に引用漏れの `echo $1`、ワークフローの `jdx/mise-action` を可動タグへ、通常の Go ファイルに高エントロピーのパスワードを同時に仕込むと `failed: shellcheck zizmor betterleaks` で落ちた。3 つとも戻して緑に復帰した。
  - **Detection Reason**: 減らした結果として何も見ていない状態になっていないことは、緑では判別できない。残した検査の 1 つずつに、それが担当するはずの欠陥を与えて赤を確認するしかない。**この確認が設計判断を 1 つ取り消した。** 当初予定していた `-S warning` では引用漏れが報告されず、深刻度による除外が SC2086 という本物のバグ種別を巻き添えにしていた。
- **Independent Verification**:
  `risk: low` / `reversibility: reversible` のため要求されない。
- **Change-Resistance Results**:
  `risk: low` のため要求されない。上の故障注入がその役割を兼ねる。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run lint-repo` - passed (3 checks, 約 500ms)
