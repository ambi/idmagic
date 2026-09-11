# check

`check/` は、リポジトリ検査を一つの深いモジュールとして提供する。
検査名と実装は `registry.ts` だけで対応づけ、`runner.ts` が共有 snapshot に対して選択した規則を並行実行する。
各規則は `process.argv`、`process.exit`、リポジトリ root の推測を持たず、所見を値として返す。

利用者向けの入口は `mise` タスクである。

```bash
mise run check
mise run check-spec
mise run check-work-items
mise run check-api-compat
```

`mise run check` は登録済みの TypeScript 検査を一回の Bun 起動で実行し、一つの規則が例外を投げても残りの結果を集める。
個別タスクは同じ registry から一つの規則を選ぶため、検査ごとの CLI shell は持たない。

work item の識別子はファイル名の stem 全体である。
同じ stem を `work-items/` と `work-items/done/` に置くと `mise run check-work-items` が拒否する。
先頭の番号は割り当ての目安であり、同じ番号だけでは衝突としない。

## Coverage debt

次の台帳は、テストがまだ名前を記載していない normative id を保持する。
`mise run check-coverage-debt-ratchet -- <base-revision>` は、指定した Git revision の台帳にない id の追加を拒否する。

| ファイル | 対象 | 追加を拒否する基準 | 検査 |
| --- | --- | --- | --- |
| `example-coverage-debt.json` | 実行可能な scenario example | Git 基準 revision の同じ台帳 | `check-coverage-debt-ratchet` |

`standards.md` の行は台帳を持たない。
宣言した行は、その id を名指すテストを持つか `mise run check-spec` に落ちるかのどちらかである。
台帳は wi-495 が空にして削除した。
同じ名前のファイルを置き直しても、どの検査も読まない。

ローカルの既定基準は `main` である。
Pull Request では base SHA を、`main` への push では push 直前の SHA を CI が渡す。
既存 id の `reason` 更新と id の削除は許可する。

`mise run report-coverage-debt` は、契約上の拒否を使って debt の優先順位を報告する。
この照会は検査の合否を変えない。
