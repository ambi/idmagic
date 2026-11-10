---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-11
priority: p1
depends_on: [wi-25266-authenticate-agent-clients-with-jwt-svid]
change_kind: bugfix
affected_spec:
  - { path: docs/modules/workloadidentity/features/attestation-exchange.md, requirement: REQ-WORKLOADIDENTITY-001 }
  - { path: docs/modules/oauth2/standards.md, requirement: RFC8693-SUBJECT-TOKEN }
---

# ワークロードのアテステーションの交換で、要求したクライアントと束縛先のクライアントの関係を決める

## 動機

REQ-WORKLOADIDENTITY-001 は、JWT のアテステーションを「その Agent の束縛先の `client_id` の資格情報として」短命のアクセストークンへ交換すると定める。
実装（`backend/oauth2/internal/token/usecases/exchange_token.go`）は、トークンの `sub` を束縛先のクライアントにする。
一方、トークンの `client_id` と認可の評価には、トークンエンドポイントで認証した別のクライアント（要求したクライアント）を使い、両者が一致するかを確かめない。

その結果、Token Exchange を許可されたどのクライアントでも、手に入れた JWT-SVID を提示すれば、束縛先の Agent を `sub` とするトークンを得られる。
トークンの `client_id` も束縛先と異なり、リソースサーバーと監査から見える主体の意味がぶれる。
JWT-SVID は Bearer なので、漏えいした JWT-SVID の悪用を、要求したクライアントの認証で狭められていない。

## 対象範囲

- 要求したクライアントと束縛先のクライアントの関係を仕様で定め、実装を合わせる。
- 拒否の理由を `WorkloadAttestationRejected` に足す。
- 既存の利用者への移行の案内を、リリース文書に書く。

## 対象外

- JWT-SVID によるクライアント認証そのもの。[[wi-25266-authenticate-agent-clients-with-jwt-svid]] が扱う。
- 人の利用者を `sub` とする委譲の Token Exchange。

## 設計

| 案 | 内容 | 評価 |
| --- | --- | --- |
| A（推す） | 要求したクライアントが束縛先のクライアントと一致する場合だけ交換する。一致しなければ `invalid_grant` で拒否する | 発行するトークンの `sub` と `client_id` が同じ主体になる。wi-25266 の後は、Agent が JWT-SVID でクライアント認証し、同じ JWT-SVID か別の JWT-SVID を `subject_token` に載せればよく、長期の鍵は要らない |
| B | クライアント認証なしで交換を受け付ける（Google Cloud の STS の形） | 構成は単純になるが、トークンエンドポイントで公開クライアントの交換を許すことになり、AIMS00-AGENT-CLIENT-AUTH と食い違う |
| C | 要求したクライアントを仲介者として認め、`act` に記録し、関連付けごとに許す仲介者を登録する | 仲介の構成を表せるが、需要が確かめられておらず、登録の項目と管理画面が増える |

案 A を推す。
wi-25266 の後は、`spiffe_jwt` でクライアント認証して `client_credentials` を使えば同じトークンを得られる。
Token Exchange の経路は互換のために残し、案 A の制約を足す。
仲介の構成が必要と示されたら、案 C を別の作業項目で検討する。

着手時に、既存の配備で要求したクライアントと束縛先が異なる使い方があるかを確かめる。
あれば、拒否に切り替える前に、監査イベントで不一致を記録する期間を置くかどうかを決める。

## 計画

1. 案を確定し、既存の使い方の有無を確かめる。
2. 仕様（REQ-WORKLOADIDENTITY-001 と、RFC8693-SUBJECT-TOKEN の注記）を更新する。
3. 不一致の拒否を RED にしてから実装する。

## タスク

- [ ] T001 [Design] 案と移行の扱いを確定する。
- [ ] T002 [Spec] 仕様と拒否の理由を更新する。
- [ ] T003 [Acceptance] 束縛先と異なるクライアントによる交換が拒否されることを RED にする。
- [ ] T004 [App] 照合と拒否のイベントを実装する。
- [ ] T005 [Docs] リリース文書に移行の案内を書く。
- [ ] T006 [Verify] 検証を実行する。

## 検証

- `mise run test-go-package -- ./backend/oauth2/...` と `./backend/workloadidentity/...`。
- Token Exchange の HTTP 統合テスト（localhost の待受けを使うため、サンドボックスの外で実行する）。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

- 仲介のクライアントで交換している既存の配備が壊れる。T001 で使い方を確かめ、リリース文書で案内する。
- 判定を要求の `client_id` の文字列で比べると、大文字小文字や空白の違いですり抜けうる。認証済みのクライアントの ID と、束縛の ID を正規の値で比べる。
