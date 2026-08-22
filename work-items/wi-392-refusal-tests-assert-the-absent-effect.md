---
depends_on: [wi-390-security-control-test-standard-and-gate]
status: pending
authors: [tn]
risk: low
created_at: 2026-08-22
priority: p2
change_kind: tooling
spec_impact: { kind: none, reason: "既存テストへのアサーション追加であり、製品の振る舞いも配線契約も変えない。アサーションを足した結果として実装の欠陥が見つかった場合は、個別の work item として切り出す。" }
---

# 拒否のテストに、起きなかった副作用の確認を足す

## Motivation

[[wi-390-security-control-test-standard-and-gate]] は `DEVELOPMENT.md` に規範を置いた。拒否は「返ったステータス」ではなく「起きなかった副作用」で確かめる。**規範を書いただけで、既存のテストは直していない。**

`mise run report-security-test-gaps` の実測 (2026-08-22) では、状態を変える操作の拒否を検証しているテスト 120 件のうち **84 件が、拒否のあとに状態を読み直していない**。

| 領域 | 件数 |
|---|---|
| backend/application/handlers_http | 10 |
| backend/sourcing/scim | 10 |
| backend/oauth2/handlers_http | 9 |
| backend/shared/http | 9 |
| backend/idmanagement/group | 8 |
| backend/application/usecases | 5 |
| backend/saml/handlers_http | 5 |
| その他 | 28 |

これが放置できないのは、**まさにこの形のテストが CSRF の素通りを通してしまった**からである。3 つの拒否ケースはいずれも 403 を assert していた。403 は返っていた。要求も通っていた。

wi-390 で 1 件だけ直した `TestEnsureDefaultAndRejectDefaultDisable` が典型である。`ErrDefaultTenant` を assert するだけで default テナントが有効なままかを見ておらず、**先に無効化してから拒否を返す実装でも通っていた**。

## Scope

- 84 件の拒否テストに、拒否された操作が状態を変えていないことの確認を足す。作成されるはずだった行が無い、更新されるはずだった値が元のまま、発行されるはずだったイベントが出ていない、のいずれかを、テスト自身が読み直して確かめる。
- 足した結果として落ちるテストがあれば、それは実装の欠陥である。**本 work item では直さず、欠陥として個別に切り出す。** テストの修正と実装の修正を同じ変更に混ぜると、どちらが何を意味するのか後から読めない。
- `mise run report-security-test-gaps` の件数が減ることを完了の指標にする。

## Out of Scope

- 副作用の不在を機械的に強制する検査。wi-390 が見送った判断をそのまま引き継ぐ。「本体に読み直しが 1 つ以上ある」という構文の要求は、意味のない 1 行で満たせてしまう。
- 拒否の宣言の床と 137 件の分類。[[wi-391-refusal-declaration-floor-and-reinventory]] が持つ。
- 読み取りの拒否。素通りしても副作用が無いので対象にしない。
- 見つかった実装の欠陥の修正。切り出した先で扱う。

## Design

未定。着手時に次の 2 点を確定して本節に記録する。

1. **分割の単位。** 84 件を 1 度に扱うと着手できない。領域ごと (`application/handlers_http` の 10 件、`sourcing/scim` の 10 件…) に切るのが素直だが、テストの書き方は層 (handlers_http / usecases / db_*) で揃っているので層ごとのほうが手数が少ないかもしれない。最初の 1 領域を通してから決める。
2. **「読み直し」の書き方の型。** handlers_http では HTTP で読み直すのか、リポジトリを直接読むのか。前者は経路ごと確かめられるが、読み取り側の認可にも依存する。後者は素直だが、テストがリポジトリを握っていない場合に書けない。型を決めて `DEVELOPMENT.md` の例に足すかを判断する。

## Plan

- 報告タスクの件数がもっとも多い領域から着手し、1 領域を通して手順と 1 件あたりの重さを測る。
- 落ちたテストは切り出して次へ進む。止まらない。
- 件数の推移を work item に記録する。減っていることが見えなければ、この作業は続かない。

## Tasks

- [ ] T001 [Design] 分割の単位と「読み直し」の書き方の型を確定し `## Design` に記録する。
- [ ] T002 [Test] 最初の 1 領域に副作用の不在の確認を足し、手順と重さを測る。
- [ ] T003 [Triage] 足した結果落ちたテストを実装の欠陥として切り出す。
- [ ] T004 [Test] 残りの領域へ広げる。
- [ ] T005 [Verify] `mise run report-security-test-gaps` の件数が減っていることを確認する。`mise run verify` を通す。

## Verification

- `mise run report-security-test-gaps`
  - reason: 着手前後の件数を比べ、減っていることを完了の指標にする。
- `mise run test-go`
- `mise run verify`
- 手動: 直したテストのうち 1 件について、拒否の前に副作用を起こす実装へ一時的に変え、テストが落ちることを確認する。落ちなければ、その確認は書けていない。

## Risk Notes

リスクは low。テストへのアサーション追加であり、製品には触れない。

ただし**形だけ満たす危険がある**。「何かを読み直す 1 行」を足せば報告の件数は減るが、それでは CSRF のときと同じである。読み直した値が拒否前と同じであることまで assert しなければ意味がない。件数の減少を指標にすると、この形骸化を招きやすい。Verification の手動確認 (拒否の前に副作用を起こす実装でテストが落ちるか) を、件数より上の判断基準として扱う。

落ちるテストが出た場合、それは製品の欠陥であって作業の失敗ではない。切り出して先へ進む。
