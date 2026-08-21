# SharedSignals Decisions

- `SsfStream` の登録、更新、有効化、無効化、削除、および配送状況の参照は、`admin` ロールを持つ、有効かつ認証済みのユーザーだけが所属テナントに対して行える。
- 受信エンドポイント (`/ssf/streams/{stream_id}/events`) はブラウザーセッションを持たない外部の送信側が呼ぶため、管理 API の認証経路には載せない。代わりに、SET の署名が受信ストリームに登録済みの `trusted_issuer` の鍵で検証できること、`jti` が未使用であること、主体がそのストリームのテナント内で解決できることをすべて満たしたときにだけ受理する。1 つでも満たさなければ失効を反映せず拒否する。受理した SET が変更できるのは対象 Agent の失効エポックだけであり、これを進める以外の副作用は持たない。
- 失効エポックの前進 (`AdvanceRevocationEpoch`) と参照 (`CheckRevocationEpoch`) は HTTP に公開せず、Domain Event と OAuth2 の `Introspect` からの内部呼び出しに限る。エポックを巻き戻す操作は、どの権限にも存在しない。
