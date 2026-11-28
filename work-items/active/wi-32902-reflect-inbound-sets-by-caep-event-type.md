---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-28
priority: p1
group: federation-protocols
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/sharedsignals/features/receiver.md, requirement: REQ-SHAREDSIGNALS-010 }
  - { path: docs/modules/sharedsignals/standards.md, requirement: SSF10-RECEIVE-SUB-ID }
---

# 受信した SET の反映を CAEP のイベントの種類と購読で決める

## 動機

2026-11-28 に [[wi-21740-decide-recordings-for-requests-idmagic-receives]] で、OpenID CAEP 1.0 final の Examples の SET を本物の verifier と受信のユースケースへ流して、二つの欠陥を見つけた。
記録は `backend/sharedsignals/internal/verify_jose/testdata/caep-1_0-examples.json`、テストは同じパッケージの `TestReceiveSecurityEvent_CAEPExamples` である。

一つ目に、受信は署名と主体を検証できた SET を、イベントの種類を問わず主体の失効として反映し、User のセッションを終わらせる。
受信側のストリームが購読する種類（`SsfStream.EventTypes`）とも照合しない。
session-revoked だけを購読するストリームで、CAEP 1.0 §3.3.2 Figure 9 の credential-change（FIDO2 の認証器の作成）と、§3.4.2 Figure 10、11 の assurance-level-change（保証レベルの上昇）を受けると、利用者が失効した。
CAEP 1.0 の credential-change は資格情報の作成、変更、取り消し、削除の通知であり、assurance-level-change は保証レベルの変化の通知であって、どちらもセッションの失効を意味しない。
REQ-SHAREDSIGNALS-010 は「受信側のストリームが SET を受理したとき、主体の失効エポックを進める」とだけ述べ、種類に触れていない。

二つ目に、主体を解決できる SET の種類が IdMagic の `CaepEventType` にないとき（risk-level-change、device-compliance-change、session-established、session-presented）、受信は拒否ではなく `caep event type is not in enum` の内部エラーを返す。
`writeReceivedSecurityEventError` はこのエラーをそのまま返すので、送信側は 500 を受け取り、RFC 8935 の push では再送を繰り返す。
記録の例では主体の形式が先に拒否されるのでこの経路に届かず、`iss_sub` の発行者の照合を外した変異で観測した。

利用者が誤った結果（不要な失効とセッションの終了）を受け取るので、グループの基準の p3 ではなく p1 とする。
2026-11-28 に利用者が (c) 直すと分類した。

## 対象範囲

- 受信がイベントの種類ごとの反映を決める。少なくとも、session-revoked は失効させ、ストリームが購読しない種類は失効させない。
- 購読しているが失効を意味しない種類と、IdMagic が知らない種類の扱い（受理だけを記録するか、RFC 8935 §2.3 の誤りで拒否するか）を、着手時に仕様で決める。どちらでも 500 は返さない。
- REQ-SHAREDSIGNALS-010 の文面と例を、決めた扱いに合わせる。
- [[wi-21740-decide-recordings-for-requests-idmagic-receives]] の `caepNotReplayed` の三つの例を、決めた期待値で再生する。

## 対象外

- `complex` などの未対応の主体の形式の採用。SSF10-RECEIVE-SUB-ID の宣言を変える別の判断であり、[[wi-36958-conform-to-shared-signals-framework-1-0]] の範囲と照らして決める。

## 設計

着手時に、CAEP 1.0 の各イベントの意味と、SSF 1.0 の受信側の要件（購読していない種類の扱い）を読み、種類ごとの反映を表にする。
反映の判断はドメインの層の、種類と購読を入力にした計算にし、ユースケースは結果に従って失効と記録を編成する。

## 計画

1. `spec-change` で REQ-SHAREDSIGNALS-010 と例を更新する。
2. CAEP の例の再生で RED を観測し、実装を直す。

## タスク

- [ ] T001 [Spec] 種類ごとの反映を仕様に書く。
- [ ] T002 [Acceptance] CAEP の例の再生と、未知の種類の SET で RED を確認する。
- [ ] T003 [App] 反映の判断と、未知の種類の扱いを直す。
- [ ] T004 [Verify] 差分から選んだ検査を実行する。

## 検証

- `mise run test-go-package -- ./backend/sharedsignals/internal/verify_jose`、`mise run test-go-package -- ./backend/sharedsignals/internal/usecases`、`mise run test-go-package -- ./backend/sharedsignals/internal/handlers_http`。

## リスク

失効させる種類を狭めすぎると、送信側が失効を意図した SET を見落とす。
session-revoked のほかに失効を意味する種類（credential-change の取り消しなど）を、CAEP の文面から一つずつ判断する。
