---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-01
depends_on: []
---

# wi-314 の最終検証で発見した既存の仕様整合・DBテスト分離・UI複雑度違反を解消する

## Motivation

wi-314 の対象 UI は個別検証を通過したが、リポジトリ全体の `just verify` では、その変更と
無関係な既存の3つの失敗が再現した。仕様上の `AgentStatus.Disabled` が Go の wire 値
`disabled` に解決されない、TOTP 秘密再暗号化の全表検査が同一パッケージ内の先行テスト行を
拾う、`AccountProfilePage.tsx` が既存 complexity ceiling を超える、という問題である。
検証基準を例外扱いで緩めず、各原因を最小範囲で修正して全体検証を再び信頼できる状態に戻す。

## Scope

- `spec/contexts/identity-management.yaml` と、合成 glossary 上で `Disabled` の wire alias を
  所有する該当 context
- `backend/shared/spec/coherence_test.go` の既存 `AgentStatus` 整合テスト
- `backend/authentication/totp/db_postgres/reencrypt_test.go` の DB テスト分離
- `frontend/src/features/account/AccountProfilePage.tsx` と抽出する profile 表示コンポーネント
- `architecture.yaml` の既存 `ui-page-lines` ceiling（値を緩和せず、実装側を ceiling 以下へ戻す）

## Out of Scope

- Agent lifecycle や TOTP 再暗号化の API・永続化契約の変更。
- Account profile の表示項目・保存挙動・ナビゲーション変更。
- complexity ceiling の引き上げや debt 例外の追加。

## Design

- SCL の canonical enum 名は維持し、`Disabled` が既存 Go/API wire 値 `disabled` に解決される
  glossary alias を正本側で補う。Go 定数を SCL の表示名へ変える契約変更は行わない。
- TOTP の「全表に平文が残らない」保証は維持する。その検査を実行する前に共有 embedded
  PostgreSQL 内の先行テストデータを明示的に分離し、このテストが投入・backfill した行だけで
  全表条件を成立させる。
- Account profile は既存 Container/Presentation 分離を維持し、属性グループ表示・編集 field 群を
  page ファイル外へ抽出する。complexity budget は緩和しない。
- 外部入力文法や認可判定の新設はないため、fuzz/property test は追加しない。

## Plan

1. SCL glossary を先に修正し、既存 coherence test と `just check-scl` を green にする。
2. TOTP の既存 failing test を test isolation の修正で green にし、パッケージ検証を通す。
3. Account profile の presentational helper を抽出し、既存 UI test と architecture check を通す。
4. `just verify` を再実行して3つの失敗が全て解消したことを確認する。

## Tasks

- [x] T001 [SCL] `AgentStatus.Disabled` が wire 値 `disabled` に解決されるよう glossary alias を
      修正し、既存 `TestAgentStatusMatchesSCL` を GREEN にする。合成時に優先される DataKeys の
      `Disabled` glossary に wire alias を追加し、SCL 派生物を再生成した。
- [x] T002 [Test] TOTP backfill 全表検査を同一パッケージの先行テスト行から分離し、
      `TestMfaFactorReencryptor_NoPlaintextSurvivesBackfillAcrossTenants` を GREEN にする。
      検査条件は全表のまま維持し、テスト fixture 作成前に共有 DB の factor 行だけを消去した。
- [x] T003 [UI] Account profile の属性表示・編集 presentational helper を page 外へ抽出し、
      `ui-page-lines` を ceiling 499 以下へ戻す（ceiling は変更しない）。page は 509 行から
      304 行になり、属性の表示・編集操作テストを追加して既存動作を確認した。不要になった
      architecture debt 例外も削除した。
- [x] T004 [Verify] `just check-scl` / 対象 Go package test / Account profile UI test /
      `just check` / `just verify` を実行し、全て GREEN にする。

## Verification

- `just check-scl`
- `just test-go-package ./backend/shared/spec`
- `just test-go-package ./backend/authentication/totp/db_postgres`
- `just test-ui-unit-file src/features/account/AccountProfilePage.test.tsx`
- `just check`
- `just verify`

## Risk Notes

公開契約や本番データ処理を変えず、仕様の wire alias、テスト DB の分離、UI ファイル分割だけを
扱うためリスクは低い。TOTP の全表検査を弱めないことと、Account profile の表示・保存回帰を
既存テストで確認することを完了条件とする。

## Completion

- **Completed At**: 2026-08-01
- **Summary**:
  `AgentStatus.Disabled` の wire alias 欠落を SCL 正本で補い、派生 HTML を再生成した。
  TOTP の全表 plaintext 検査は条件を弱めず、共有 embedded PostgreSQL のテスト fixture を
  分離した。Account profile は属性表示・編集責務を専用モジュールへ抽出して page を 509 行から
  304 行へ縮小し、不要になった complexity debt 例外を削除した。本番 API・永続化契約・画面挙動は
  変更していない。
  test-first からの逸脱 (self-attest): T001/T002 は `just verify` の既存 RED を確認してから修正。
  T003 は既存 architecture RED に対する構造リファクタリングで、抽出後に属性表示・編集の回帰
  テストを追加したため、新規テスト単体の RED は先行確認していない。
- **Verification Results**:
  - `just check-scl` / `just scl-render` - passed。
  - `just test-go-package ./backend/shared/spec` - passed。
  - `just test-go-package ./backend/authentication/totp/db_postgres` - passed。
  - `just test-ui-unit` - passed（516 tests）。
  - `just check` / `just verify` - passed。
