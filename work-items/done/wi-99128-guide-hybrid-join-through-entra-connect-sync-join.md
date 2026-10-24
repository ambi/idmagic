---
status: completed
---

# Hybrid Join を、Entra Connect の sync join で行う前提で案内する

規範仕様の差分はない。WS-Trust の能動的な対応を usernamemixed の Issue だけに絞る判断に、windowstransport を社外へ公開してはならないという Microsoft の要求と、それを実装して出口 IP で絞る案を退けた理由を足し、Hybrid Join はフェデレーションによる即時の登録に失敗した後の sync join（Windows 10 1803 以降、Entra Connect がコンピューターとデフォルトの端末の属性を同期する）で終わると書いた。
RP と Entra フェデレーションの管理の信頼性の節と、構成の結果の `hybrid_join_not_supported` の辞書の文言を、managed や AD FS への誘導から sync join の案内に改めた。制限のコードと返す条件は変えていない。
wi-79 の検証範囲を、sync join で登録が終わることの実テナントでの確認に改めた。wi-43000 が取り込むようにした AD のコンピューターアカウントは利用側がなくなったので、扱いを別に決める。
