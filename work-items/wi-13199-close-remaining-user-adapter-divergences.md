---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-30
priority: p2
depends_on: []
change_kind: tooling
spec_impact: { kind: none, reason: "idmanagement/user の永続化アダプターの既存の振る舞いを、共有の契約テストでさらに固定する作業である。シナリオも公開契約も変えない。規範判断を要する食い違いが見つかった場合は bugfix work item に切り出す。" }
---

# idmanagement/user の契約テストへ、未検証の食い違い候補 6 件を加える

## 動機

wi-16658 は、`idmanagement/user` のメモリ実装と PostgreSQL 実装を `backend/idmanagement/user/testing_contract` の同じ契約へ通した。
その契約は 7 つのサブテストからなり、検出した食い違いは「メモリ User repository が同一テナントで有効な `preferred_username` の重複を受理する」1 件だけだった。

wi-16658 の着手時にコードを読んだところ、この契約が確かめていない食い違いの候補が 6 件見つかった。
いずれもコードを読んで立てた仮説であり、テストでは観測していない。

| 候補 | 対象 | 予想される差 |
|---|---|---|
| C1 | メモリ `UserRepository.Save` | 削除済みの user A と同じ `preferred_username` を、有効な user B が再利用しているとき、A を保存し直すとメモリ実装は `errPreferredUsernameExists` で拒否する。PostgreSQL の部分一意索引は削除済みの行を除外するので受理する。 |
| C2 | メモリ `UserRepository` の読み取り | 保存したポインターをそのまま返し、`Save` の引数も複製せずに保持する。呼び出し側が返り値や引数を後から書き換えると、`Save` を経ずに保存内容が変わる。PostgreSQL 実装では起きない。 |
| C3 | メモリ `UserRepository.Save` の更新 | 既存の user を保存し直すと `CreatedAt` と `TenantID` を上書きする。PostgreSQL の `SaveUser` は `ON CONFLICT (id)` でこの 2 列を更新しない。 |
| C4 | PostgreSQL の状態フィルター | `UserLifecycle.Status` の JSON タグに `omitempty` が無いので、未設定の状態は `"status":""` として保存される。`coalesce(lifecycle->>'status', 'active')` は空文字を置き換えないため、`active` で絞り込むとその user が漏れる。メモリ実装とドメインの `EffectiveStatus` は未設定を `active` として扱う。 |
| C5 | メモリ `EmailChangeTokenStore.Find` | 返すエンベロープは構造体の複製だが、`Payload` の map は保存内容と共有している。返り値の map を書き換えると、次の `Find` の結果が変わる。 |
| C6 | 時刻の分解能 | PostgreSQL は `TIMESTAMPTZ` をマイクロ秒に切り捨てて返す。メモリ実装はナノ秒のまま返す。wi-16658 の設計は「保存した時刻はマイクロ秒で読み戻る」ことを契約として固定するとしたが、契約はマイクロ秒に切り捨てた入力しか与えていない。 |

上位のテストはメモリ実装をフェイクとして使い、`PERSISTENCE=memory` は本番の構成でも選べる。
C2 と C5 の共有があると、`Save` を呼び忘れたユースケースでもメモリ実装のテストは通ってしまう。
C4 が事実なら、管理画面で状態を `active` に絞り込んだときに、PostgreSQL 構成でだけ一部の user が一覧から消える。

## 対象範囲

- C1〜C6 の各候補について、`backend/idmanagement/user/testing_contract` にサブテストを加え、両実装で実行して食い違いが実在するかを観測する。
- 実在した食い違いのうち、既存の PostgreSQL の制約またはドメインの規則に合わせるだけで直せるものは、この項目で直す。wi-16658 と同じ扱いである。
- 実在しなかった候補も、契約のサブテストとして残す。同じ誤りの再発を検出するためである。

## 対象外

- `idmanagement/user` 以外の Context の契約。同じ種類の共有（C2、C5）や時刻の分解能（C6）の差が他の Context にもあり得るが、この項目では扱わない。`idmanagement/user` で観測した結果から、別の項目で扱うかを判断する。
- 規範判断を要する食い違いの修正。bugfix work item に切り出す。
- `userdomain.UserLifecycle` の JSON 表現を変えるデータ移行。製品は未リリースなので、表現を変える場合もスキーマ宣言の更新だけで足りる。

## 設計

契約は wi-16658 の `testing_contract.Run` と `Fixture` をそのまま使い、サブテストを足す。
新しい仕組みは加えない。

各候補が契約として固定する性質は次のとおりである。

| 候補 | 契約として固定する性質 | 直す側の候補 |
|---|---|---|
| C1 | 削除済みの user は、その `preferred_username` を別の有効な user が使っていても保存し直せる。その後も `FindByUsername` は有効な user を返す。 | メモリ実装。重複の判定と名前の索引を、削除済みの user を除外して行う。 |
| C2 | `Find*`、`FindAll`、`ListPage*` の返り値を書き換えても、保存内容は変わらない。`Save` の後で引数を書き換えても、保存内容は変わらない。 | メモリ実装。境界で深く複製する。`TenantUserAttributeSchemaRepository` の `cloneUserAttributeSchema` が前例である。 |
| C3 | 既存の user を保存し直しても、`CreatedAt` と `TenantID` は最初に保存した値のまま読み戻る。 | メモリ実装。`TenantUserAttributeSchemaRepository.Save` が `CreatedAt` を保持するのと同じ形にする。 |
| C4 | `Lifecycle.Status` が未設定の user は、`active` での絞り込み（`ListPage*Filtered`、`CountFiltered`）に含まれる。 | PostgreSQL 実装。SQL で空文字も `active` として扱う。 |
| C5 | `Find` が返すエンベロープの `Payload` を書き換えても、次の `Find` の結果は変わらない。 | メモリ実装。`Payload` を複製して返す。 |
| C6 | ナノ秒を含む時刻を保存すると、マイクロ秒に切り捨てた値として読み戻る。対象は `User.CreatedAt`、`User.UpdatedAt`、エンベロープの `IssuedAt`、`ExpiresAt` である。 | 実在すれば、合わせる側を着手時に決める（計画を参照）。 |

採用しない案は次のとおりである。

| 案 | 採用しない理由 |
|---|---|
| 候補ごとに項目を分ける | 6 件とも同じ契約パッケージに同じ形のサブテストを足す作業であり、別々に受け入れられる成果がない。記録を分けると、readiness pass と証拠の固定費が 6 倍になる。 |
| メモリ実装の読み取りを複製せず、「返り値を書き換えない」ことを呼び出し側の規約にする | 規約は検査できず、`Save` を呼び忘れたユースケースのテストが通り続ける。設計ガイドラインの「所有者を曖昧にしない可変値」にも反する。 |

## 計画

1. C1〜C6 のサブテストを書き、両実装で実行して RED を観測する。DB を通すテストはサンドボックスの外で実行し、PostgreSQL 側がスキップされていないことを確かめる。
2. 実在した食い違いを、局所的な修正と規範判断を要するものに分ける。
3. 局所的な修正を 1 候補ずつ行い、GREEN にする。
4. 規範判断を要する食い違いは bugfix work item に切り出し、契約では wi-16658 と同じく、その項目を名指す理由付きで一時的に外す。
5. `docs/development/testing.md` の契約の節に書き足すことがあれば更新する。

未解決の問いは 2 つあり、どちらも観測した結果で決まるので、手順 1 の後に解決する。

- **C4 は製品の振る舞いの修正か。** 管理画面の user 一覧で状態による絞り込みを定めるシナリオがあれば、C4 の修正はそのシナリオへの準拠を回復する bugfix になる。その場合は `change_kind` を `bugfix` に変え、`affected_spec` に該当する `REQ-*` を記す。
- **C6 はどちらへ合わせるか。** 時刻の分解能を規範が定めていなければ、メモリ実装を PostgreSQL に合わせてマイクロ秒に切り捨てる。上位のテストがナノ秒の一致に依存していて壊れる場合は、規範判断が要るものとして切り出す。

## タスク

- [ ] T001 [Test] C1〜C6 のサブテストを書き、両実装で実行して RED を記録する。
- [ ] T002 [Triage] 実在した食い違いを分類し、未解決の 2 つの問いに答えて設計へ書く。
- [ ] T003 [Fix] 局所的な修正を 1 候補ずつ GREEN にする。
- [ ] T004 [Triage] 規範判断を要する食い違いを bugfix work item に切り出す。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- メモリ実装と PostgreSQL 実装のテストが、追加したサブテストを同じ契約本体から実行する。
- 各修正を戻すと、対応するサブテストのうち、直した実装の側だけが落ちる。
- `mise run test-go-package -- ./backend/idmanagement/user/db_postgres/` をサンドボックスの外で実行し、スキップされずに通る。
- `mise run test-go-changed`
- `mise run verify`

## リスク

- **C2 の複製で上位のテストが落ちる。** メモリ実装の共有に依存して、`Save` を呼ばずに状態を変えている上位のテストやユースケースがあれば、複製によって落ちる。これは製品の誤りの検出であり、テストを緩めて通さない。落ちた箇所が多い場合は、その件数を記録して進め方を決め直す。
- **PostgreSQL 側のテストが黙ってスキップされる。** サンドボックス内では `testing_postgres.Require` がスキップする。検証はサンドボックスの外で行い、PostgreSQL 側でだけ落ちる C4 の RED を観測して、実行されたことを確かめる。
- **候補が実在しない。** 読解だけから立てた仮説なので、観測で否定される候補があり得る。その場合も、サブテストは再発の検出として残す。
