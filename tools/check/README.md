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

## 用語

`mise run check-terminology` は、設計文書が採らないと決めた表記を拒否する。
対象は `docs/` の Markdown と、`terminology.ts` の `TERMINOLOGY_ROOT_DOCUMENTS` が挙げる root 直下の文書である。
生成物の `CONFIGURATION.md` と `ROUTE_PRIORITY.md`、および書かれた時点の記録である `work-items/` は読まない。

規則は `terminology.ts` の `TERMINOLOGY_RULES` が持つ。
1 件は、採らない表記、代わりに使う表記、そして残す共起からなる。

```ts
{
  term: '秘密',
  adopt: '「シークレット」',
  allow: [{ literal: '秘密鍵', reason: 'private key であってシークレットではない' }],
}
```

`allow` の literal は対象語を含み、その literal が覆う位置に現れた occurrence だけを通す。
「トポロジ」と「トポロジー」のように、採用した表記が採らない表記を含む場合もこれで扱える。

免除はこの規則表にしかない。
ファイル単位の免除は持たない。文書ごとに外せるようにすると、その文書だけ用語が戻ったことに誰も気付かないためである。
正当な用法が落ちたときは、理由を書いた `allow` を足す。
