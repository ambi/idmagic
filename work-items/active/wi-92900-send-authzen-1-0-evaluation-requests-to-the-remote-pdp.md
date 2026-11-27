---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-28
priority: p2
group: spec-conformance
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/authorization/standards.md, requirement: AUTHZEN-FGA-EVALUATION }
  - { path: docs/modules/authorization/standards.md, requirement: AUTHZEN-FGA-FAIL-CLOSED }
---

# リモートの AuthZEN の判定点へ、AuthZEN 1.0 の形の評価要求を送る

## 動機

`AUTHZEN=remote` のとき、`backend/shared/policy/authorization_http` は判定を外部の判定点へ委ねる（[構成](../../docs/operations/configuration.md)）。
2026-11-28 に [AuthZEN Authorization API 1.0](https://openid.net/specs/authorization-api-1_0.html)（Final、2026-01-11 公開）の本文を読み、実装と比べると、次の点で食い違っていた。

| 項目 | AuthZEN 1.0 | 実装 |
| --- | --- | --- |
| 要求の `action` | 5.3 節：必須の `name` を持つオブジェクト | `spec.AuthZRequest.Action` を文字列で送る |
| 要求の `resource` | 5.2 節：`type` と `id` が必須 | `id` が空なら省く（`omitempty`） |
| 応答 | 5.5 節：必須の `decision` と任意の `context` | `decision` に加えて仕様にない `permit` と `reasons` を読み、`permit` を優先する |
| 拒否の理由 | 5.5.2.1 節：`context` の `reason_admin`、`reason_user` などの例 | 読まない |

仕様に従う判定点は文字列の `action` を不正な要求として扱い得る。
その場合、保護対象の操作はすべて fail-closed で拒否され、遠隔の判定点を使う構成が成り立たない。
既存の `TestRemoteAuthorizer` は、実装を書いた人が手で書いた `{"decision": true}` だけを返し、要求の `action` の形を見ていない。

この食い違いは、[wi-94480](../done/wi-94480-decide-recordings-for-the-remaining-external-counterparts.md) で外向きの HTTP の相手を洗い出し直したときに見つけた。

## 対象範囲

- 送る評価要求を AuthZEN 1.0 の 6.1 節の形にする。`action` は `{"name": ...}`、`resource` は `type` と `id` を常に持つ。
- 応答は 5.5 節の `decision` で判定し、`context` の理由を拒否の理由へ写すかを決める。`permit` と `reasons` を読み続けるかは、それに依る判定点が実在するかを確かめて決める。
- 10.1.4 節の Figure 28 と Figure 29、5.5.2.1 節の Figure 11 を転記した記録で再生の検査を置く（[外部契約と設定の利用](../../docs/development/testing.md#外部契約と設定の利用)）。
- `docs/modules/authorization/standards.md` に、リモートの判定点へ送る側の標準要件を足す。

## 対象外

- IdMagic が AuthZEN の判定点として評価入口を提供するか。[wi-80811](wi-80811-decide-authzen-compatible-evaluation.md) が扱う。
- バッチの評価（Access Evaluations API）と探索 API。

## 設計

ポートの型 `spec.AuthZRequest` は、ローカルの評価器と規則表も使う。
AuthZEN 1.0 の線の形への変換はリモートのアダプターの中に閉じ、ポートの型は変えない案を第一候補にする。
`resource.id` を持たない判定（作成の前の評価など）に何を送るかは、仕様の例と既存の判定の呼び出し元を読んで決める。

## 計画

1. 呼び出し元が渡す `Resource.ID` の空の場合と、応答の `permit` に依る判定点の有無を確かめる。
2. 標準要件を足し、記録の再生で RED を作る。
3. アダプターの変換を直す。

## タスク

- [ ] T001 [Spec] 送る側の標準要件を `authorization/standards.md` に足す。
- [ ] T002 [Acceptance] AuthZEN 1.0 の例を転記した記録の再生で RED を確認する。
- [ ] T003 [Adapter] 要求と応答の変換を直す。

## 検証

- 記録の要求の `action` を文字列へ戻すと再生の検査が落ちる。
- `{"decision": false}` と `context` を持つ応答で拒否し、許可へ退避しない。
- `mise run verify-changed -- main`。

## リスク

`permit` と `reasons` を返す独自の判定点へ既に接続している運用があれば、応答の解釈を変えると判定が変わる。
解釈を変える前に、その形を返す判定点の有無を確かめる。
