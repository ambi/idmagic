package usecases

// self-service のセッション一覧と失効 (wi-20 スライス 2)。actor.sub == target.sub に固定し、
// 本人は自分の LoginSession のみ参照・失効できる。失効は LoginSession を物理削除して SSO
// セッションを終了する。OAuth クライアントへ発行済みの refresh token はセッションに 1:1 で
// 紐づかないため本スライスでは失効しない (per-session の refresh 失効は後続スライス)。

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/ambi/idmagic/backend/authentication/domain"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

// ErrSessionNotFound は対象セッションが存在しないか、本人のものでない場合。
var ErrSessionNotFound = errors.New("session not found")

// SessionDeps はセッション use case の依存。
type SessionDeps struct {
	Store authnports.SessionStore
	Emit  func(spec.DomainEvent)
}

// SessionView は一覧表示用のセッション射影。secret は持たず、識別子と認証情報のみ。
type SessionView struct {
	ID        string
	Current   bool
	AMR       []string
	ACR       string
	StartedAt time.Time
	ExpiresAt time.Time
}

// ListSessions は sub の有効なセッションを開始時刻の降順で返す。currentSessionID に
// 一致するものを Current=true でマークする。
func ListSessions(
	ctx context.Context,
	store authnports.SessionStore,
	sub, currentSessionID string,
) ([]SessionView, error) {
	if store == nil {
		return []SessionView{}, nil
	}
	sessions, err := store.ListBySub(ctx, sub)
	if err != nil {
		return nil, err
	}
	views := make([]SessionView, 0, len(sessions))
	for _, sess := range sessions {
		views = append(views, SessionView{
			ID:        sess.ID,
			Current:   sess.ID == currentSessionID,
			AMR:       sess.AMR,
			ACR:       sess.ACR,
			StartedAt: time.Unix(sess.AuthTime, 0).UTC(),
			ExpiresAt: sess.ExpiresAt,
		})
	}
	sort.Slice(views, func(i, j int) bool { return views[i].StartedAt.After(views[j].StartedAt) })
	return views, nil
}

// RevokeOwnSession は本人のセッション 1 件を失効する。対象が存在しないか本人のもので
// なければ ErrSessionNotFound。失効は tombstone であり、既に失効済みの対象への再送は
// idempotent に成功する (event は再送しない、ADR-126)。
func RevokeOwnSession(
	ctx context.Context,
	deps SessionDeps,
	sub, sessionID string,
	now time.Time,
) error {
	sess, err := deps.Store.FindOwned(ctx, sessionID, sub)
	if err != nil {
		return err
	}
	if sess == nil {
		return ErrSessionNotFound
	}
	alreadyRevoked := sess.RevokedAt != nil
	if err := deps.Store.Revoke(ctx, sessionID, spec.SessionEndSelfRevoke, now); err != nil {
		return err
	}
	if !alreadyRevoked {
		emitSessionEnded(deps.Emit, sess, sub, spec.SessionEndSelfRevoke, now)
	}
	return nil
}

// RevokeOtherSessions は keepSessionID を除く本人の全セッションを失効する
// ("他のセッションを全て終了")。対象は ListBySub が返す有効なセッションに限るため、
// 既に失効済みの行は自然に対象外になる。失効した sessionID の一覧を返す。呼び出し側
// (account_sessions_handler) はこれを sid として oauth2 の RevokeTokensBySid へ渡し、
// refresh token family を横断して失効させる (ADR-127)。
func RevokeOtherSessions(
	ctx context.Context,
	deps SessionDeps,
	sub, keepSessionID string,
	now time.Time,
) ([]string, error) {
	sessions, err := deps.Store.ListBySub(ctx, sub)
	if err != nil {
		return nil, err
	}
	revokedIDs := make([]string, 0, len(sessions))
	for _, sess := range sessions {
		if sess.ID == keepSessionID {
			continue
		}
		if err := deps.Store.Revoke(ctx, sess.ID, spec.SessionEndSelfRevoke, now); err != nil {
			return nil, err
		}
		emitSessionEnded(deps.Emit, sess, sub, spec.SessionEndSelfRevoke, now)
		revokedIDs = append(revokedIDs, sess.ID)
	}
	return revokedIDs, nil
}

// AdminSessionView は admin 向け一覧表示用のセッション射影 (wi-28 T007)。self-service
// の SessionView と異なり current マーカーの代わりに対象 UserID と LastSeenAt を持つ
// (操作者と対象ユーザーが別人のため、ADR-127 決定9)。
type AdminSessionView struct {
	ID         string
	UserID     string
	AMR        []string
	ACR        string
	StartedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
}

// AdminListSessions は admin が対象ユーザーの有効なセッションを開始時刻の降順で
// 一覧する (wi-28, ADR-127 決定9)。ListUserSignInActivity と同じアクセス制御
// パターン (TenantAdministrator, resource=User/input.user_id) を前提とし、
// アクセス制御自体は HTTP 層が担う。
func AdminListSessions(ctx context.Context, store authnports.SessionStore, targetUserID string) ([]AdminSessionView, error) {
	if store == nil {
		return []AdminSessionView{}, nil
	}
	sessions, err := store.ListBySub(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	views := make([]AdminSessionView, 0, len(sessions))
	for _, sess := range sessions {
		views = append(views, AdminSessionView{
			ID: sess.ID, UserID: sess.UserID, AMR: sess.AMR, ACR: sess.ACR,
			StartedAt: time.Unix(sess.AuthTime, 0).UTC(), LastSeenAt: sess.LastSeenAt, ExpiresAt: sess.ExpiresAt,
		})
	}
	sort.Slice(views, func(i, j int) bool { return views[i].StartedAt.After(views[j].StartedAt) })
	return views, nil
}

// AdminRevokeSession は admin が対象ユーザーのセッション 1 件を失効する (wi-28)。
// sessionID が targetUserID の所有でなければ ErrSessionNotFound (URL 上の user_id と
// session の実所有者の不一致を fail-closed で拒否する)。RevokeMySession と同じ
// tombstone 契約で、既に失効済みの対象への再失効は idempotent に成功する。
func AdminRevokeSession(
	ctx context.Context,
	deps SessionDeps,
	actorUserID, targetUserID, sessionID string,
	now time.Time,
) error {
	sess, err := deps.Store.FindOwned(ctx, sessionID, targetUserID)
	if err != nil {
		return err
	}
	if sess == nil {
		return ErrSessionNotFound
	}
	alreadyRevoked := sess.RevokedAt != nil
	if err := deps.Store.Revoke(ctx, sessionID, spec.SessionEndAdminRevoke, now); err != nil {
		return err
	}
	if !alreadyRevoked {
		emitSessionEnded(deps.Emit, sess, actorUserID, spec.SessionEndAdminRevoke, now)
	}
	return nil
}

// AdminRevokeUserSessions は admin が対象ユーザーの全セッションを失効する (wi-28)。
// RevokeOtherSessions と異なり、操作者自身のセッションではないため除外対象
// (keepSessionID) が無い。失効した sessionID の一覧を返し、呼び出し側が oauth2 の
// RevokeTokensBySid へ渡す (ADR-127)。
func AdminRevokeUserSessions(
	ctx context.Context,
	deps SessionDeps,
	actorUserID, targetUserID string,
	now time.Time,
) ([]string, error) {
	sessions, err := deps.Store.ListBySub(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	revokedIDs := make([]string, 0, len(sessions))
	for _, sess := range sessions {
		if err := deps.Store.Revoke(ctx, sess.ID, spec.SessionEndAdminRevoke, now); err != nil {
			return nil, err
		}
		emitSessionEnded(deps.Emit, sess, actorUserID, spec.SessionEndAdminRevoke, now)
		revokedIDs = append(revokedIDs, sess.ID)
	}
	return revokedIDs, nil
}

// EndSession は RP-Initiated Logout (/end_session) など、所有者 (sub) を未検証の
// sid (id_token_hint または browser cookie から解決) を失効する。self-service の
// Revoke* と異なり FindOwned による所有者確認はしない — sid 自体が検証済みの
// id_token_hint か、本人だけが送れる browser cookie から来ているため十分な認可根拠
// となる (ADR-127)。Find は有効な (未失効・未期限切れ) セッションのみ返すため、
// 既に失効済み・期限切れ・未知の sid は自然に no-op になる (idempotent)。
func EndSession(
	ctx context.Context,
	deps SessionDeps,
	sid string,
	now time.Time,
) error {
	if sid == "" {
		return nil
	}
	sess, err := deps.Store.Find(ctx, sid)
	if err != nil {
		return err
	}
	if sess == nil || sess.TenantID != tenancy.TenantID(ctx) {
		return nil
	}
	if err := deps.Store.Revoke(ctx, sid, spec.SessionEndLogout, now); err != nil {
		return err
	}
	emitSessionEnded(deps.Emit, sess, sess.UserID, spec.SessionEndLogout, now)
	return nil
}

func emitSessionEnded(
	emit func(spec.DomainEvent),
	sess *domain.LoginSession,
	actorUserID string,
	reason spec.SessionEndReason,
	now time.Time,
) {
	if emit == nil {
		return
	}
	emit(&domain.SessionEnded{
		At: normalizedNow(now), TenantID: sess.TenantID, UserID: sess.UserID,
		SessionID: sess.ID, ActorUserID: actorUserID, Reason: reason,
	})
}
