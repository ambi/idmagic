package handlers_http

// SCL interface: ListAuthenticationEventBuckets (bounded_context: Authentication)。
// SCL permission: AdminAuditEventsRead を再利用する (集約も監査可視化の一部)。
// 攻撃時にログイン失敗を個別行へ落とさず集約した bucket を、所属テナント境界内で
// 新しい窓順に返す (wi-20 スライス 3)。書き込み経路は定義しない。

import (
	"net/http"
	"time"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type authEventBucketResponse struct {
	Kind        string    `json:"kind"`
	KeyHash     string    `json:"key_hash"`
	WindowStart time.Time `json:"window_start"`
	Count       int       `json:"count"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
}

const listAuthenticationEventBucketsQuery = "ListAuthenticationEventBuckets"

func handleListAuthEventBuckets(d Deps, c *echo.Context) error {
	actor, err := d.RequireAuditReader(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	page, err := support.ParsePageRequest(c, d.PaginationCodec, actor.TenantID, listAuthenticationEventBucketsQuery, authusecases.AuthEventBucketDefaultLimit, authusecases.AuthEventBucketMaxLimit)
	if err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	afterWindowStart := time.Time{}
	if page.AfterPrimary != "" {
		afterWindowStart, err = time.Parse(time.RFC3339Nano, page.AfterPrimary)
		if err != nil {
			return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "cursor is invalid or expired.")
		}
	}
	buckets, err := authusecases.ListAuthEventBuckets(
		c.Request().Context(), d.AuthEventBucketStore, actor.TenantID, afterWindowStart, page.AfterID, page.Limit+1,
	)
	if err != nil {
		return err
	}
	hasMore := len(buckets) > page.Limit
	if hasMore {
		buckets = buckets[:page.Limit]
	}
	response := make([]authEventBucketResponse, len(buckets))
	for i, bucket := range buckets {
		response[i] = authEventBucketResponse{
			Kind:        bucket.Kind,
			KeyHash:     bucket.KeyHash,
			WindowStart: bucket.WindowStart,
			Count:       bucket.Count,
			FirstSeen:   bucket.FirstSeen,
			LastSeen:    bucket.LastSeen,
		}
	}
	if hasMore {
		last := buckets[len(buckets)-1]
		afterKey := last.Kind + "|" + last.KeyHash
		if err := support.SetNextLink(c, d.PaginationCodec, d.Issuer, actor.TenantID, listAuthenticationEventBucketsQuery, last.WindowStart.UTC().Format(time.RFC3339Nano), afterKey, hasMore); err != nil {
			return err
		}
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"buckets": response})
}
