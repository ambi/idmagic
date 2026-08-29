# ClaimMapping Decisions

- クレームの発行を `OAuth2` と XML フェデレーションから切り離し、独立した Bounded Context とする。発行は署名や搬送とは独立した純粋な変換であり、WS-* や SAML の RP 信頼を `OAuth2` へ取り込むと、その Context の責務が肥大するからである。
- 規則は宣言的な対応付けに限り、条件分岐や変換関数を持つ規則言語は導入しない。属性最小化とフェイルクローズを、規則を実行せずに静的に検証できる範囲へ保つためである。
- この Context を `Supporting` に分類する。プロトコルに依存しないクレーム開示ポリシーは必要な抽象だが、規則自体は属性からクレームへの宣言的な対応付けであり、複雑さも差別化も高くない。
- この Context は Aggregate を持たない。`ClaimMappingPolicy` は値オブジェクトとして、各プロトコル Context の信頼先 Aggregate（`OAuth2Client`、`SamlServiceProvider`、`WsFedRelyingParty`）に埋め込む。ポリシーは信頼先の登録と同時にしか正しくなりえず、独立した Aggregate にすると信頼先を削除してポリシーだけが残る状態を作れてしまうからである。この Context が持つのは、その値の意味と検証だけである。
