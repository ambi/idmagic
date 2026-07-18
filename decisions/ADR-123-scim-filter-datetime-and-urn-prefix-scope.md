---
status: accepted
authors: [tn]
created_at: 2026-07-18
---

# ADR-123: SCIM inbound filter grammar は dateTime 比較と schema URN プレフィックスまでを対応範囲とする

## コンテキスト

wi-238 は RFC 7644 §3.4.2.2 filter grammar のうち、属性・演算子の allowlist に閉じたサブセット
だけを実装した。この時点では `gt`/`ge`/`lt`/`le` が構文としては認識されるが、どの属性の allowlist
にも許可されていないため常に `invalidFilter` になり、SCIM の実運用で最も頻出する
`meta.lastModified gt "..."` による増分同期 (delta sync) が使えなかった。また schema URN プレフィックス
付き属性名 (`urn:ietf:params:scim:schemas:core:2.0:User:userName eq "x"`) も未対応で、URN 修飾した
属性名をデフォルトで送る一部 enterprise クライアントを弾いていた。

この work item (wi-244) はこの2点を埋めるが、RFC 7644 §3.4.2.2 filter grammar への完全対応では
ない。複数値属性の複合フィルタ (bracket 構文、例 `emails[type eq "work"]`) は User の email 表現が
単一値に簡略化されている現状 (wi-239) に依存し、custom / extension schema 属性へのフィルタは
属性の schema 拡張性自体が未実装であるため、どちらもこの work item の範囲外に置く。

ADR-121 の基準 (「SCL の `adoption` を `partial` にした work item は ADR を残す」) に従い、
この範囲選択の理由を構造化記録として残す。

## 決定

**1. `RFC7644-FILTERING` requirement を `adoption: partial` として追加する。**
対応範囲は「dateTime 属性 (`meta.created`/`meta.lastModified`) への `gt`/`ge`/`lt`/`le`/`eq`/`ne`」と
「schema URN プレフィックス付き属性名の解決」の2点に限定し、`reason` にこれと未対応範囲
(複数値複合フィルタ・custom schema 属性) を明記する (`spec/contexts/scim.yaml`)。

**2. dateTime 比較は文字列辞書順ではなく実時刻 (`time.Parse` した `time.Time` 同士) で行う。**
RFC3339 は同一時刻を複数の offset 表記で表現できる (`Z` と `+09:00` など) ため、文字列比較のままだと
offset 表記の異なるレコード間で誤った順序判定を招き、増分同期クライアントに欠落や重複を発生させる
(wi-244 Risk Notes)。`time.Time.Equal`/`Before`/`After` は instant 単位で比較するためこれを避けられる。

**3. schema URN プレフィックスは lexer ではなく属性解決段階で剥がし、既存の allowlist へ渡す。**
`urn:ietf:params:scim:schemas:core:2.0:User:` / `...:Group:` の既知プレフィックスだけを認識し、
剥がした後の属性名は prefix なしの場合と同一の allowlist 検証を通す (新しい検証経路を作らない)。
未知の URN プレフィックスは `invalidFilter` にする。allowlist を迂回する経路を作らないことで、
プレフィックス解決が injection・過剰計算量対策 (wi-238 Risk Notes) を弱めないことを保証する。

## 却下した代替案

- **複数値複合フィルタ (bracket 構文) も同時に対応する**: User の `emails` が単一値表現に簡略化
  されている現状 (wi-239) では bracket 構文が指す複数値要素が存在せず、意味のある実装にならない。
  wi-239 が multi-valued 対応した後に別 work item で扱う。
- **custom / extension schema 属性のフィルタにも対応する**: 属性の schema 拡張性自体が未実装であり、
  この work item 単体で対応するには範囲が大きすぎる。
- **URN プレフィックス解決専用の allowlist を新設する**: prefix 剥がし後に既存 allowlist と別の
  検証経路を持たせると、片方だけ更新して allowlist がずれるリスクがある。同一 allowlist を通す方が
  安全。

## 影響

- `spec/contexts/scim.yaml`: `standards.RFC7644.requirements` に `RFC7644-FILTERING`
  (`adoption: partial`) を追加。`interfaces.ListScimUsers`/`interfaces.ListScimGroups` の
  description を dateTime 比較・URN プレフィックス対応まで更新。
- `backend/scim/domain/filter.go`: `AttrDateTime` kind、`meta.created`/`meta.lastModified` の
  allowlist エントリ、`stripSchemaURNPrefix` を追加。
