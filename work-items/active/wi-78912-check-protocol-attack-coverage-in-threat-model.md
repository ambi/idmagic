---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-20
priority: p2
group: security-hardening
depends_on: []
change_kind: docs
spec_impact: { kind: none, reason: "既知の攻撃の類型と既存の脅威モデルの行、標準仕様の規範 ID の対応を調べ、欠落を脅威モデルに planned または accepted の行として記録する。既存の THREAT-NNN の ID と状態、規範シナリオ、標準仕様の規範 ID、TypeSpec の契約は変えない。" }
---

# 認証プロトコルの既知の攻撃を脅威モデルがどこまで覆うかを確かめ、欠落を記録する

## 動機

IdMagic は OAuth 2.0、OIDC、SAML 2.0、WS-Federation を実装する。
どれも、実装を誤ると認可の横取りやなりすましにつながる攻撃の類型が、標準化団体と研究で整理されているプロトコルである。

[セキュリティ設計](../../docs/design/security/README.md)は、プロトコル固有の制御について「各モジュールの標準仕様とシナリオで定める」と一文で書くだけである。
[脅威モデル](../../docs/design/security/threat-model.md)の「プロトコルとトークン」の節には、リダイレクト先の照合、認可コードの横取り、リフレッシュトークンの再生、XML 署名の組み替えなどの行がある。
しかし、出典が挙げる攻撃の類型を一つずつ照らし合わせて、行の漏れがないかを確かめた記録はない。
SAML と WS-Federation にはモジュールの `security.md` があるが、OAuth2 にはない。

漏れがあっても、今の脅威モデルからは気付けない。
漏れた類型は `planned` の行としても現れず、是正の作業項目も起票されないためである。

この項目は [wi-83807](wi-83807-make-system-design-docs-easier-to-read.md) の起票時に、設計文書の見直しから分けた。

## 対象範囲

- 下の出典を本文で読み、攻撃の類型ごとに、応える脅威モデルの行、標準仕様の規範 ID、モジュールの `security.md` の節を対応付ける。
- 応える行がない類型を、脅威モデルに `planned` または `accepted` の行として足す。
  `accepted` にする場合は、受容の理由と再検討の条件を「受容した残留リスク」に書く。
- 既存の行の `Threat` の記述が類型より狭く、類型の一部しか覆っていない場合は、残りの部分を新しい行として足す。
- セキュリティ設計の入口に、プロトコル固有のセキュリティ設計の所在（脅威モデルの節、モジュールの `standards.md` と `security.md`）を書く。
- OAuth2 にモジュールの `security.md` を新設するかを決め、新設する場合は書く。

## 対象外

- 見つけた欠落の是正（規範の追加と実装）。
  欠落ごとに別の作業項目として起票し、その作業項目から `THREAT-NNN` を名指す。
- 脅威モデルの `Controls` の参照をリンクへ書き換える作業。
  [wi-83807](wi-83807-make-system-design-docs-easier-to-read.md) が扱う。
  この項目で足す行は、どちらが先に完了しても合うよう、wi-83807 の完了後ならリンクの形で、完了前なら既存の書式で書く。
- 侵入試験と外部のセキュリティレビュー。
- 上の四つ以外のプロトコル（SCIM、SSF、MCP など）。
  同じ方法で調べる価値はあるが、出典の性格が異なるので、必要になったら別の項目にする。

## 設計

### 出典

次の出典を本文で読む。
起票時点では読んでいない。
着手時に、読んだ範囲と調査日をこの節に書き足す。

| プロトコル | 出典 | 読む範囲 |
| --- | --- | --- |
| OAuth 2.0 | RFC 9700（OAuth 2.0 Security Best Current Practice） | 第 4 章「Attacks and Mitigations」 |
| OAuth 2.0 | RFC 6819（OAuth 2.0 Threat Model and Security Considerations） | RFC 9700 が更新していない脅威の確認に限る |
| OIDC | OpenID Connect Core 1.0 | 第 16 章「Security Considerations」 |
| SAML 2.0 | OASIS, Security and Privacy Considerations for the OASIS SAML V2.0 | 脅威と対策の章 |
| SAML、WS-Federation | Somorovsky ほか, "On Breaking SAML: Be Whoever You Want to Be", USENIX Security 2012 | XML 署名ラッピング攻撃の類型 |
| WS-Federation | OASIS, WS-Federation 1.2 | Security Considerations の節 |

### 対応表の扱い

類型と制御の対応表は、調べるための作業の材料であり、設計文書には残さない。
脅威モデルが「どの脅威に何が応えるか」の一次情報であり、対応表を別に残すと、二つの表が別々に古くなるためである。
対応表はこの作業項目の設計に置き、完了時の要約には、調べた範囲、足した行の ID、OAuth2 の `security.md` の判断だけを残す。

### 行の粒度

新しい行は、既存の行と同じく、IdMagic に当てはまる脅威を一文で書く。
出典の類型の名前をそのまま行にはしない。
IdMagic が提供しない機能（例：暗黙フロー）に向けた類型は、行を足さず、対応表に「提供しない機能」と書くだけにする。

### OAuth2 の `security.md` を新設する条件

脅威モデルの行と標準仕様の規範 ID だけでは説明できない機構があれば新設する。
たとえば、複数の制御を組み合わせて一つの攻撃を防ぐ仕組みや、規範 ID には現れない実装上の前提（照合の順序、正規化の手順）である。
どの類型もいずれかの規範 ID で説明できれば、新設しない。
形式上揃えるためだけに文書を作らない（[文書ガイド](../../docs/formats/documentation-guide.md)）。

## 計画

1. 出典を読み、プロトコルごとに対応表を作る。
2. 欠落を脅威モデルに足す。
3. OAuth2 の `security.md` の要否を決め、必要なら書く。
4. セキュリティ設計の入口に所在を書く。
5. 欠落ごとの是正の作業項目を起票する。

## タスク

- [ ] T001 [Research] 出典を読み、読んだ範囲と調査日を設計に書く。
- [ ] T002 [Research] プロトコルごとに、攻撃の類型と既存の行、規範 ID、`security.md` の対応表を作る。
- [ ] T003 [Docs] 欠落を脅威モデルに `planned` または `accepted` の行として足す。
- [ ] T004 [Docs] OAuth2 の `security.md` の要否を決め、必要なら書く。
- [ ] T005 [Docs] セキュリティ設計の入口に、プロトコル固有のセキュリティ設計の所在を書く。
- [ ] T006 [Docs] 欠落ごとの是正の作業項目を起票する。
- [ ] T007 [Verify] 変更を検証する。

## 検証

- `mise run check-spec`：足した `THREAT-NNN` の一意性と、`Controls` が指す規範 ID の解決を確かめる。
- `mise run check-security-controls`：脅威モデルの行と制御の対応の検査が通ることを確かめる。
- `mise run check-links`：入口に書いた所在のリンクと、受容した残留リスクのアンカーを確かめる。
- `mise run check-work-items`：起票した是正の作業項目の形式を確かめる。

## リスク

- 対応付けを甘く判定すると、既存の行が類型を覆っていると誤って結論し、欠落を見逃す。
  既存の行の `Controls` が指す規範 ID の本文まで読み、類型の攻撃手順のどの段を止めるかを確かめてから、覆っていると判定する。
- 出典の類型をすべて行にすると、脅威モデルが読めない量になる。
  IdMagic に当てはまらない類型は行にせず、対応表にだけ記録する。
