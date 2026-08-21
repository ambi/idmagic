# Audit Internals

## Search attribute registry

検索文法は `AuditSearchRegistry` が宣言する属性、演算子、変換方法の閉じた集合に限り、任意の SQL や JSONPath は受け付けない。個人識別情報にあたる属性には、レジストリで定めた変換 (ハッシュ化や丸め) を適用してから照合するため、検索用インデックスに平文は残らない。平文を保持できるのは `audit_events.payload` だけであり、対象を失敗イベントに限ったうえで保持期間を短くする。
