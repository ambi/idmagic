---
depends_on: []
status: completed
authors: [tn]
risk: low
created_at: 2026-08-23
priority: p1
change_kind: docs
initial_context:
  specification: [spec/README.md]
  source: [README.md, DEVELOPMENT.md, spec/api-rules.md, .github/workflows/idmagic-ci.yaml]
  tests: []
  stop_before_reading: [backend, frontend, work-items/done]
spec_impact: { kind: none, reason: "リポジトリの入口に置く文書を足す作業である。規範的な振る舞い、契約、Context の境界のいずれにも触れない。README への対象外の宣言は、spec/README.md の Context Map 索引を範囲宣言として参照する形にするので、範囲そのものを新たに定義することもしない。" }
---

# リポジトリ入口の文書を置く——脆弱性の報告経路、貢献の作法、変更の告知、対象外の宣言

## Motivation

`README.md` の 1 行目はこう宣言する。

> **プロダクションレディなエンタープライズ向け ID プロバイダー。**

その宣言に対して、リポジトリの入口には次が無い。ルートと `.github/` の双方を確認した（`.github/` には `workflows/idmagic-ci.yaml` のみ）。

| 欠けているもの | それが無いと何が起きるか |
|---|---|
| `SECURITY.md` | **脆弱性を見つけた人に、公開 issue 以外の選択肢が無い。** |
| `CONTRIBUTING.md` | 仕様先行のループを知らない人が、実装から書き始める |
| `CHANGELOG.md` | 非推奨の告知先が無い |
| 対象外の宣言 | 何を担わないかが、口頭でしか伝わらない |

**`SECURITY.md` が最も重い。** IdP は認証を預かる製品であり、脆弱性の報告は必ず来る想定で置くべきものである。報告経路が無ければ、報告者は公開 issue を立てるか、報告をやめるかのどちらかを選ぶ。前者は未修正の脆弱性を公開することであり、報告者の善意が製品の利用者を害する形になる。

**`CHANGELOG.md` は宙に浮いた約束を回収する。** [spec/api-rules.md](../spec/api-rules.md) §Deprecation は「`deprecated_since` を設定したインターフェースはレスポンスに `Deprecation` ヘッダーを付け」と定め、[DOCUMENTATION_GUIDE.md](../DOCUMENTATION_GUIDE.md) §9.3 は「CHANGELOG はその告知である」と書く。ヘッダーは付くが、**告知の本体を読む場所が無い。** `Deprecation: true` を受け取った利用者に、何がいつどう変わるのかを伝える手段が存在しない。

**対象外の宣言。** DOCUMENTATION_GUIDE §9.2 は、対象範囲を機能の箇条書きにしないこと、そして**対象外とその引き受け先**を書くことを求める。現状の `README.md` は「主な機能」8 件の箇条書きで範囲を語っており、対象外の記述は無い。IdP/IdM は特権アクセス管理、人事情報の正、アプリケーション内部の認可判定と境界を接する。**接する相手が具体的なほど、担わないことの宣言が要る。**

`README.md` の文書表にも漏れがある。`DOCUMENTATION_GUIDE.md`、`tools/README.md`、`work-items/` を指す行が無い。文書体系そのものを定義している `DOCUMENTATION_GUIDE.md` が、入口から辿れない。

## Scope

- `SECURITY.md` を置く。報告経路、対応する版の範囲、応答の目安、開示の方針を書く。
- `CONTRIBUTING.md` を置く。DOCUMENTATION_GUIDE §9.3 の形に従い、環境構築とコマンドは既存文書へ委ね、**Pull Request に求めるもの**（仕様先行、生成物の再生成、1 変更 1 work item と `affected_spec`）だけを書く。必須の検査の一覧は複製せず CI 定義を正本とする。
- 対象外の宣言を置く。IdP/IdM として担わないものと、その引き受け先を表にする。対象範囲は `spec/README.md` の Context Map 索引を参照する形にし、機能の箇条書きにしない。
- `README.md` の文書表に `DOCUMENTATION_GUIDE.md`、`tools/README.md`、`work-items/` の行を足す。
- `CHANGELOG.md` は Design の 3 の判断次第で置く。

## Out of Scope

- リリース・ロールバック手順の文書化。DOCUMENTATION_GUIDE §11.2 は `operations/release-and-rollback.md` を挙げるが、**このリポジトリはまだリリースを行っていない。** 手順の無い段階で手順書を作ると、§3 の「必要が生じていない文書を作らない」に反し、最初のリリースの時点で必ず書き直しになる。最初のリリースを定義する変更が持つ。
- テスト水準の文書（§10.3 の `testing.md`）。検証のはしごと拒否のテストの規範は [DEVELOPMENT.md](../DEVELOPMENT.md) §4 が既に持っており、いま独立した文書が無いことで困っている読み手はいない。
- サービス目標の正本化。[[wi-400-service-objectives-need-stable-ids]] が持つ。
- 製品概要の詳細（対象ユーザー、解決する問題）。対象外の宣言と同じ場所に置くかは Design の 1 で決めるが、内容を書き起こすのはこの変更の後でよい。

## Design

3 点とも着手時に確定した。1 と 2 は人が決めた（DOCUMENTATION_GUIDE §12.3 が「プロダクトの目的、対象ユーザー、対象外の線引き」「許容するセキュリティ・運用リスク」を人間の責任とする）。

1. **対象外の宣言は `README.md` の節に置く。** DOCUMENTATION_GUIDE §9.2 は `/docs/product-overview.md` を挙げるが、このリポジトリに `docs/` は無く、手順は `DEVELOPMENT.md`（ルート）、`infra/README.md`、`infra/runbooks/`、`frontend/README.md` と、**それが動かすものの隣**に置かれている。対象外 3 行のためにディレクトリを 1 つ増やすと、`DEVELOPMENT.md` などをそこへ移すかどうかの判断まで巻き込む。`spec/` と `docs/` の区別そのものへの疑問は別の work item が持つ。

2. **`SECURITY.md` は GitHub Security Advisories の非公開報告を経路とし、応答の日数を約束しない。** リモートが `github.com/ambi/idmagic` なので追加の運用なしに使え、報告者に公開 issue 以外の選択肢を渡せる。日数を約束しないのは、守れない期限を書くことが期限を書かないことより悪いためである。後から厳しくするのは容易で、緩めるのは難しい。

3. **`CHANGELOG.md` は作らない。** `deprecated_since` と `@deprecated` は `spec/**/*.tsp` に 1 件も無い（着手時に確認）。**告知すべき非推奨がまだ存在しない。** リリースもしていないので、いま作れば空のまま古くなる。§3 の「必要が生じていない文書を作らない」に従う。api-rules.md が約束する `Deprecation` ヘッダーを実際に出す最初の変更が、告知先を用意する責任を持つ。

### 併せて分かったこと

`LICENSE` がリポジトリに無い。ライセンスの無いリポジトリは既定で全権利留保であり、公開しているなら利用条件が不明という意味になる。**入口の文書としては本 work item の 4 件と同じ性質だが、Scope に挙げていないので追加しない。** ライセンスの選定は人の判断であり、文書を置く作業ではない。

## Plan

- 1 と 2 の判断を先に取る。これが決まらないと何も書けない。
- `SECURITY.md` を最初に入れる。4 件のうち唯一、無いこと自体が利用者への危険になっている。
- `CONTRIBUTING.md` は既存文書への委譲が主なので、判断待ちにならない。並行して入れられる。
- 3 は `deprecated_since` の実在を確かめてから決める。

## Tasks

- [x] T001 [Design] 対象外の宣言の置き場所、`SECURITY.md` で約束する内容、`CHANGELOG.md` の要否を確定し `## Design` に記録する。
- [x] T002 [Docs] `SECURITY.md` を置く。GitHub Security Advisories を経路とし、扱う境界を `spec/authorization.md` に紐づけた。
- [x] T003 [Docs] `CONTRIBUTING.md` を置く。必須の検査の一覧を複製せず、CI 定義を正本として指す。
- [x] T004 [Docs] 対象外の宣言を `README.md` の節へ置いた。対象範囲は `spec/README.md` の Context Map 索引を参照する形にした。
- [x] T005 [Docs] `README.md` の文書表に `DOCUMENTATION_GUIDE.md`、`tools/README.md`、`work-items/`、および新設した 2 件の行を足す。
- [x] T006 [Docs] `CHANGELOG.md` は作らない。`deprecated_since` が 1 件も無く、告知すべき非推奨が存在しないため（Design 3）。
- [x] T007 [Verify] `mise run verify` を通し、README からの相対リンクがすべて解決することを確認した。

## Verification

- `mise run verify`
- 手動: `README.md` と新しく置いた文書のすべての相対リンクを辿り、実在するファイルを指していることを確認する。入口の文書で壊れたリンクは、最初の読み手が最初に踏む。
- 手動: `SECURITY.md` に書いた報告経路へ実際に 1 通送り、届くことを確認する。**届かない報告先を書くのは、報告先を書かないより悪い。**
- 手動: 対象外の表の各行について、引き受け先が具体的に名指しされていることを確認する。「本製品の範囲外」とだけ書いた行は、読み手に何も渡していない。

## Risk Notes

リスクは low。文書の追加であり、コードにも仕様にも触れない。

**危険はむしろ、置いた文書が守られない約束を含むことにある。** `SECURITY.md` の応答目安、`CONTRIBUTING.md` の Pull Request 要件、対象外の表——どれも、書いた瞬間に外部から参照される契約になる。守れないものは書かない。とくに応答の目安は、**書かずに「報告先だけを示す」ほうが、守れない日数を書くより誠実である。**

`CONTRIBUTING.md` については、必須の検査の一覧を書き写す誘惑が強い。書き写すと CI 定義との複製になり、片方が必ず古くなる。DOCUMENTATION_GUIDE §9.3 が明示的に「この文書へ一覧を複製しない」と書いているとおり、方針と理由だけを書いて一覧は CI 定義に委ねる。

対象外の宣言は、**書いた時点では正しく、製品が育つと嘘になる種類の文書**である。「特権アクセス管理は担わない」と書いた後で PAM 機能を入れたら、その行を消す責任が生じる。範囲を Context Map の索引に委ねているのは、索引なら Context が増減したときにしか変わらないからである。対象外の表は索引ほど安定しないので、行数を欲張らず、境界を接する相手が具体的な 3〜5 件に絞る。

## Completion

- **Completed At**: 2026-08-23
- **Summary**:
  リポジトリの入口に、これまで存在しなかった 3 つの経路を置いた。`SECURITY.md` は脆弱性の報告を GitHub Security Advisories の非公開経路へ導き、扱う境界を `spec/authorization.md` のテナント境界・フェイルクローズ・鍵素材に紐づけ、開発用の既定値を本番で使った場合を対象外として切り分けた。応答の日数は約束していない。`CONTRIBUTING.md` は Pull Request に求める 5 点（仕様先行、1 変更 1 work item、生成物の再生成、拒否の二重の assert、英語のコミット）を述べ、必須の検査の一覧は複製せず CI 定義を正本として指す。`README.md` には対象外の表 3 行を足し、対象範囲の宣言は `spec/README.md` の Context Map 索引に委ねた。あわせて文書表に `CONTRIBUTING.md`、`SECURITY.md`、`DOCUMENTATION_GUIDE.md`、`work-items/`、`tools/README.md` の 5 行を足した。`CHANGELOG.md` は作っていない。`deprecated_since` と `@deprecated` が `spec/**/*.tsp` に 1 件も無く、告知すべき非推奨が存在しないためである。規範的な仕様は変えていない（`mise run spec-diff` が `no normative specification change`）。
- **Verification Results**:
  - `mise run verify` - passed（exit 0）
  - `mise run spec-diff` - no normative specification change against main
  - 手動: `README.md` / `SECURITY.md` / `CONTRIBUTING.md` の相対リンクを全件解決確認 - passed
  - 手動: 報告経路への実送信による到達確認 - **未実施**（下記）

## Left Undone

- **`SECURITY.md` が案内する報告経路は、まだ開いていない可能性がある。** GitHub の非公開の脆弱性報告は、リポジトリ設定の Private vulnerability reporting を有効化するまで受け付けない。**有効化するまで、この文書は届かない報告先を案内している。** Risk Notes が「届かない報告先を書くのは、報告先を書かないより悪い」とした状態そのものなので、有効化を最優先で行う。
- **`LICENSE` が無い。** Design の「併せて分かったこと」に記録した。入口の文書としては本 work item の 4 件と同じ性質だが、ライセンスの選定は文書を置く作業ではなく人の判断なので、Scope に加えなかった。
- **`spec/` と `docs/` の区別への疑問。** Design 1 で `docs/` を作らない判断をしたが、その根拠は「このリポジトリの実態に合わせる」であって、DOCUMENTATION_GUIDE §4 が `docs/` と `operations/` を勧めていること自体は変えていない。ガイドと実態の食い違いは別の work item が持つ。
