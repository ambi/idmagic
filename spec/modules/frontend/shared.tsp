namespace IdMagicUi;

/** 入力と操作の定義を共有しても、入力の識別子は利用するページで展開する。 */
model PageData<Forms, Actions, Parameters = {}, States = FormStates> {
  parameters: Parameters;
  forms: Forms;
  states: States;
  actions: Actions;
}

const entered = #{initial: "空", condition: "対象のフォームを操作するとき"};
const accountContext = #{area: Area.account, actor: "認証済みの本人", context: "対象テナントの本人のセッションで参照する。CSRF と再認証は共通規則に従う。", references: #["[共通の UI 規則](user-interface.md)"]};
const adminContext = #{area: Area.admin, actor: "テナント管理者", context: "対象テナントの管理権限で参照し、操作の認可は API が判断する。", references: #["[共通の UI 規則](user-interface.md)"]};
const systemContext = #{area: Area.system, actor: "システム管理者", context: "制御プレーンの管理権限と必要な認証保証で参照する。", references: #["[共通の UI 規則](user-interface.md)"]};
const browserContext = #{area: Area.authentication, actor: "認可フローの利用者", context: "対象テナントのブラウザートランザクションを使う。CSRF は表示 API から得る。", references: #["[共通の UI 規則](user-interface.md)"]};
const navigation = #{condition: "リンクを選択したとき", data: "対象の識別子とテナントの文脈"};
const update = #{mode: "data", data: "対象の識別子と操作の入力値"};

/** 共通の状態。個別のページは必要な状態をこのモデルへ加える。 */
model ReadStates {
  ready: "取得した内容を表示する。";
  loading: "ページ移動中は直前の表示を保つ。初回の取得はルーターが扱う。";
  error: "表示できなければ共通エラーを表示し、操作失敗は操作の近くに残す。";
  forbidden: "権限不足を表示し、未反映の状態を保つ。";
  expired: "認証が必要なページでは再認可し、復帰できなければログインへ移る。";
  success: "成功した操作を一度通知し、表示内容を再取得する。";
}

/** 入力を伴うページに適用する状態。 */
model FormStates {
  ...ReadStates;
  invalid: "入力不備の表示は共通フォーム規則に従う。項目ごとの規則は別に定義する。";
  submitting: "実行中の操作を示し、二重送信を防ぐ。";
}

/** 一覧に適用する状態。 */
model ListStates {
  ...FormStates;
  empty: "対象がないことと、作成または検索条件を変える入口を示す。";
}

/** CSV の処理の結果。 */
model ImportStates {
  ...FormStates;
  preview: "取り込み前の内容と行ごとの問題を表示する。";
  confirm: "取り込みの適用対象と影響を確認するダイアログを表示する。";
  partial: "成功と失敗を区別し、失敗した行と項目を示す。";
}

/** URL を持たない表示と、システム外の到達先。 */
model Destinations {
  @destination(#{id: "external-idp", name: "外部 IdP", kind: "external", description: "フェデレーションの認証先。認証の結果を IdMagic のコールバックへ返す。"})
  idp: unknown;
  @destination(#{id: "client-application", name: "呼び出し元アプリケーション", kind: "external", description: "認可要求のリダイレクト先。認可結果と state をプロトコルに従って返す。"})
  client: unknown;
  @destination(#{id: "email", name: "通知メール", kind: "external", description: "パスワード再設定またはメール確認のリンクを届ける。リンクから該当するページへ戻る。"})
  email: unknown;
  @destination(#{id: "download", name: "ダウンロード", kind: "external", description: "エクスポートしたファイルまたは本人のデータをブラウザーへ渡す。"})
  download: unknown;
  @destination(#{id: "shared-error", name: "共通エラー表示", kind: "shared", description: "ルーターがページを表示できないときに出し、再ログインやポータルへの戻り方を示す。ページマーカーは error。"})
  error: unknown;
  @destination(#{id: "step-up", name: "再認証", kind: "shared", description: "元の操作の確認後、ポップアップで再認証する。成立後は一度だけ再試行し、開けなければページ全体で再認可する。"})
  stepUp: unknown;
}
