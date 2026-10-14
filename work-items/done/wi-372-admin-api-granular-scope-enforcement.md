---
status: completed
archived_from: 854f521daf0113319b826044107f04189d1f24c1
---

# 管理 API を API アクセストークンの粒度スコープで認可する

管理 API 210 本の operation に、API アクセストークンが到達するために必要な `ApiTokenScope` を TypeSpec の `x-api-token-scopes` として宣言し、ランタイム契約 (`Operation.ApiTokenScopes`) 経由でフェイルクローズに強制するようにした。
意味の差分は 2 つある。第 1 に、**粒度スコープが初めて管理 API へ到達する**。従来、API アクセストークンはポータル境界の `idmagic.admin` を持たないため管理 API のすべてで `insufficient_scope` に落ちており、33 個の粒度スコープは発行画面で選べるだけで何も許可していなかった。アカウント API と同じ形に揃え、管理 API では operation が宣言する粒度スコープがポータル境界のスコープの代わりになる。第 2 に、**その到達が operation 単位に閉じる**。`saml:read` を持つトークンは SAML SP を参照できるが、SAML SP の登録もユーザーの削除も署名鍵のローテーションも通らない。宣言のない operation と、契約に対応のないルートは拒否する。ロールとの論理積は変わらず、発行者がロールを失えばスコープを満たすトークンでも `access_denied` になる。
対話セッション限定として、API アクセストークンからは到達できないと宣言した operation が 19 本ある (API アクセストークン自身の管理 3、外部 IdP 接続 9、ReBAC 6、DEK 健全性 1)。いずれも既存の語彙に対応がなく、既存スコープへ畳み込むと昇格経路になる。
`ApiTokenScope` の列挙値は 1 つも削除していない。棚卸しの結果、37 個の管理粒度スコープはすべて 1 つ以上の operation に対応づいた (`sessions:*`、`consents:*`、`tenants:*` を含む)。`just check-api-compat` に破壊的変更は現れない。発行済みトークンが失う能力もない — 管理 API へ到達できていたトークンが存在しないためで、影響は「到達できるようになる」方向にしかない。影響の算出は、保存済みのスコープ集合 (`api_tokens.scopes`) と契約の `x-api-token-scopes` の突き合わせで静的に行える。
正本は TypeSpec 1 か所である。起票時の設計はルート登録側に対応表を持つ案 (b) だったが、`@TypeSpec.OpenAPI.extension` で新しいデコレーターなしに宣言でき、既存の `generate-contract` / `check-generated-contract` に乗ることが分かったため案 (c) を採った。ルート登録側 (20 ファイル) の書き換えは不要になり、宣言はレンダリングした API 文書にも載る。
**正本を 1 か所に保つための後続の是正。** 最初の実装では、`Scope` の記述どおり operation とスコープの対応表を各 Context の `Authorization Boundary` にも置いた。これは TypeSpec の宣言と同じ対応を散文側に複製したもので、照合する検査がないまま片方が古びる。表を外し、各 Context の境界文には「どのスコープ語彙を使うか」と「`read` / `write` の割り当てが操作の名前から読み取れない場合の判断とその理由」だけを残した。CSV エクスポートを参照系に置く理由、`KillAgent` を `agents:write` に含める理由、デフォルトサインインポリシーを `settings:*` に対応させる理由、外部 IdP 接続を対話セッション限定にする理由がそれにあたり、いずれも注釈からは読み取れない durable な判断である。網羅的な対応は TypeSpec だけが持つ。

全文は `git show 854f521daf0113319b826044107f04189d1f24c1:<このファイルのパス>` で読む。
