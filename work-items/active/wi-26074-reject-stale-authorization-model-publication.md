---
status: pending
authors: ["tn"]
risk: medium
reversibility: reversible
created_at: 2026-10-31
priority: p1
depends_on: [wi-371-authorization-rebac-admin-ui]
change_kind: feature
affected_spec:
  - { path: docs/modules/authorization/features/model.md, requirement: REQ-AUTHORIZATION-001 }
  - { path: docs/modules/authorization/features/model.md, requirement: REQ-AUTHORIZATION-011 }
  - { path: spec/modules/authorization/main.tsp, symbol: IdMagic.Authorization.Operations.PutAuthorizationModel }
---

# 古いバージョンを基にした認可モデルの発行を拒否する

## 動機
認可モデルの発行はモデル全体を新しいバージョンとして登録する。
二人の管理者が同じバージョンから編集すると、後から発行した管理者が先の変更を気付かないまま消す。
消えた規則の分だけ判定の結果が変わるので、アクセスの意図しない拡大や停止につながる。

管理コンソールは差分の確認の直前に最新を読み直すが（REQ-AUTHORIZATION-011）、確認から確定までの間の発行は検出できない。

## 対象範囲
- 発行の要求に基にしたバージョンを載せ、最新と一致しなければ新しいバージョンを作らずに拒否する。
- 管理コンソールが基にしたバージョンを送り、拒否を受けたら入力を残して差分を確認し直せるようにする。
- 要件、TypeSpec、競合の拒否の監査の扱いを定める。

## 対象外
- ほかの管理 API の競合検出。[[wi-605-detect-lost-updates-with-etag-and-if-match]] が扱う。
- 関係タプルの書き込みの競合検出。タプルは差分で書き込むので、全体の上書きが起きない。

## 設計
着手時に次を決める。

| 問い | 候補 | 現時点の推奨 |
| --- | --- | --- |
| 基にしたバージョンの渡し方 | `expected_version` を本文に載せる。`If-Match` を使う | 本文。操作は `POST` で新しいバージョンを作り、ライフサイクルワークフローの `expected_revision` に前例がある。`If-Match` の規則は `PUT`、`PATCH`、`DELETE` を対象にする |
| 省略の扱い | 無条件に受理する。必須にする | 初回公開前なので必須を検討する。初回の発行の表現も合わせて決める |
| 拒否の状態コード | 409。412 | 本文の条件なので前例と同じ 409 |

比較と採番は同じトランザクションで行う。
読んでから書く 2 段階にすると、同時発行で検査をすり抜ける。

## 計画
1. 問いを解決し、要件と TypeSpec を更新する。
2. ドメイン、リポジトリ、ハンドラーの順に実装する。
3. 管理コンソールの送信と拒否の表示を実装する。

## タスク
- [ ] T001 [Spec] 要件の差分を設計へ書き、仕様を更新する。
- [ ] T002 [Adapter] メモリと PostgreSQL の共有契約で、古いバージョンの同時発行が 1 件だけ成功することを確かめる。
- [ ] T003 [HTTP] 拒否の状態コードと本文、拒否後にバージョンが増えないことを確かめる。
- [ ] T004 [UI] 拒否を受けても入力が残り、差分を確認し直せることを確かめる。

## 検証
- `mise run verify-changed -- main`
  - reason: API と UI の両方の層へ届くため。
- `mise run test-ui-e2e-file -- tests/e2e/authorization-rebac.spec.ts`
  - reason: 手書きの要求型と実 API の一致を確かめるため。

## リスク
必須にすると API を直接呼ぶ運用手順が変わる。
拒否の判定をトランザクションの外に置くと、同時発行で上書きが残る。
