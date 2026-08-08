// Package handlers_http は SharedSignals bounded context の管理 API (ADR-057,
// wi-58 T005) を所有する。SsfStream (transmitter/receiver) の CRUD と
// SecurityEventDelivery の一覧 (delivery health) を、テナント解決済みグループに
// 登録する。
package handlers_http

import (
	"errors"
	"io"
	"net/http"
	"time"

	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
	"github.com/ambi/idmagic/backend/sharedsignals/usecases"

	"github.com/labstack/echo/v5"
)

// maxSecurityEventTokenBytes bounds the inbound SET body on the public
// (unauthenticated) ReceiveSecurityEvent endpoint. Real SETs are a few KB;
// this is defense-in-depth against abuse, not a functional limit.
const maxSecurityEventTokenBytes = 64 * 1024

type Deps struct {
	support.Deps
	*support.Authenticator
	StreamRepo            ssports.SsfStreamRepository
	TransmitterConfigRepo ssports.SsfTransmitterConfigRepository
	ReceiverConfigRepo    ssports.SsfReceiverConfigRepository
	DeliveryRepo          ssports.SecurityEventDeliveryRepository
	ReceivedEventRepo     ssports.ReceivedSecurityEventRepository
	EpochRepo             ssports.AgentRevocationEpochRepository
	AgentRepo             agentports.AgentRepository
	Verifier              ssports.SecurityEventTokenVerifier
	Emit                  func(spec.DomainEvent) error
}

func (d Deps) adminDeps() usecases.AdminStreamDeps {
	return usecases.AdminStreamDeps{
		StreamRepo: d.StreamRepo, TransmitterConfigRepo: d.TransmitterConfigRepo,
		ReceiverConfigRepo: d.ReceiverConfigRepo, DeliveryRepo: d.DeliveryRepo, Emit: d.Emit,
	}
}

func (d Deps) receiveDeps() usecases.ReceiveDeps {
	return usecases.ReceiveDeps{
		StreamRepo: d.StreamRepo, ReceiverConfigRepo: d.ReceiverConfigRepo, ReceivedEventRepo: d.ReceivedEventRepo,
		EpochRepo: d.EpochRepo, AgentRepo: d.AgentRepo, Verifier: d.Verifier, Emit: d.Emit,
	}
}

// RegisterRoutes はテナント解決済みグループに SharedSignals 管理 API と、
// 外部 transmitter からの inbound SET を受理する public エンドポイントを登録する。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/api/admin/shared-signals/streams", d.handleListStreams)
	g.POST("/api/admin/shared-signals/streams/transmitter", d.handleRegisterTransmitterStream)
	g.POST("/api/admin/shared-signals/streams/receiver", d.handleRegisterReceiverStream)
	g.GET("/api/admin/shared-signals/streams/:stream_id", d.handleGetStream)
	g.PATCH("/api/admin/shared-signals/streams/:stream_id", d.handleUpdateStream)
	g.POST("/api/admin/shared-signals/streams/:stream_id/disable", d.handleDisableStream)
	g.POST("/api/admin/shared-signals/streams/:stream_id/enable", d.handleEnableStream)
	g.DELETE("/api/admin/shared-signals/streams/:stream_id", d.handleDeleteStream)
	g.GET("/api/admin/shared-signals/streams/:stream_id/deliveries", d.handleListDeliveries)
	g.POST("/ssf/streams/:stream_id/events", d.handleReceiveSecurityEvent)
}

// handleReceiveSecurityEvent は SCL ReceiveSecurityEvent (access: public):
// 外部 transmitter が push する Security Event Token をそのまま request body
// (compact JWT, RFC 8417) として受理する。管理 API と異なり RequireAdmin/CSRF は
// 課さない — 認可は SsfReceiverConfig の trusted_issuer/JWKS 検証そのものが担う。
func (d Deps) handleReceiveSecurityEvent(c *echo.Context) error {
	body, err := io.ReadAll(io.LimitReader(c.Request().Body, maxSecurityEventTokenBytes+1))
	if err != nil {
		return err
	}
	if len(body) > maxSecurityEventTokenBytes {
		return support.WriteBrowserError(c, http.StatusRequestEntityTooLarge, "security_event_token_too_large", "The Security Event Token exceeds the accepted size.")
	}
	if err := usecases.ReceiveSecurityEvent(c.Request().Context(), d.receiveDeps(), c.Param("stream_id"), string(body), time.Now().UTC()); err != nil {
		return writeAdminSharedSignalsError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusAccepted)
}

// ---- SsfStream ----

type ssfStreamResponse struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	Direction  string     `json:"direction"`
	EventTypes []string   `json:"event_types"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
}

func toSsfStreamResponse(s *ssdomain.SsfStream) ssfStreamResponse {
	return ssfStreamResponse{
		ID: s.ID, TenantID: s.TenantID, Direction: string(s.Direction), EventTypes: caepEventTypesToStrings(s.EventTypes),
		Status: string(s.Status), CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

func caepEventTypesToStrings(types []ssdomain.CaepEventType) []string {
	out := make([]string, len(types))
	for i, t := range types {
		out[i] = string(t)
	}
	return out
}

func caepEventTypesFromStrings(values []string) []ssdomain.CaepEventType {
	out := make([]ssdomain.CaepEventType, len(values))
	for i, v := range values {
		out[i] = ssdomain.CaepEventType(v)
	}
	return out
}

type registerTransmitterStreamRequest struct {
	EventTypes            []string `json:"event_types"`
	DeliveryEndpoint      string   `json:"delivery_endpoint"`
	Audience              string   `json:"audience"`
	DeliveryAuthorization *string  `json:"delivery_authorization"`
	MaxDeliveryAttempts   *int     `json:"max_delivery_attempts"`
}

type registerReceiverStreamRequest struct {
	EventTypes        []string       `json:"event_types"`
	TrustedIssuer     string         `json:"trusted_issuer"`
	JWKSURI           *string        `json:"jwks_uri"`
	JWKS              map[string]any `json:"jwks"`
	AcceptedAudiences []string       `json:"accepted_audiences"`
}

type updateStreamRequest struct {
	EventTypes *[]string `json:"event_types"`
}

func (d Deps) handleListStreams(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	streams, err := usecases.ListSsfStreams(c.Request().Context(), d.adminDeps())
	if err != nil {
		return writeAdminSharedSignalsError(c, err)
	}
	out := make([]ssfStreamResponse, len(streams))
	for i, s := range streams {
		out[i] = toSsfStreamResponse(s)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"streams": out})
}

func (d Deps) handleGetStream(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	stream, err := usecases.GetSsfStream(c.Request().Context(), d.adminDeps(), c.Param("stream_id"))
	if err != nil {
		return writeAdminSharedSignalsError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toSsfStreamResponse(stream))
}

func (d Deps) handleRegisterTransmitterStream(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req registerTransmitterStreamRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	stream, err := usecases.RegisterSsfTransmitterStream(c.Request().Context(), d.adminDeps(), usecases.RegisterSsfTransmitterStreamInput{
		EventTypes: caepEventTypesFromStrings(req.EventTypes), DeliveryEndpoint: req.DeliveryEndpoint,
		Audience: req.Audience, DeliveryAuthorization: req.DeliveryAuthorization, MaxDeliveryAttempts: req.MaxDeliveryAttempts,
	}, time.Now().UTC())
	if err != nil {
		return writeAdminSharedSignalsError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusCreated, toSsfStreamResponse(stream))
}

func (d Deps) handleRegisterReceiverStream(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req registerReceiverStreamRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	stream, err := usecases.RegisterSsfReceiverStream(c.Request().Context(), d.adminDeps(), usecases.RegisterSsfReceiverStreamInput{
		EventTypes: caepEventTypesFromStrings(req.EventTypes), TrustedIssuer: req.TrustedIssuer,
		JWKSURI: req.JWKSURI, JWKS: req.JWKS, AcceptedAudiences: req.AcceptedAudiences,
	}, time.Now().UTC())
	if err != nil {
		return writeAdminSharedSignalsError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusCreated, toSsfStreamResponse(stream))
}

func (d Deps) handleUpdateStream(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req updateStreamRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	var eventTypes *[]ssdomain.CaepEventType
	if req.EventTypes != nil {
		converted := caepEventTypesFromStrings(*req.EventTypes)
		eventTypes = &converted
	}
	stream, err := usecases.UpdateSsfStream(c.Request().Context(), d.adminDeps(), c.Param("stream_id"), usecases.UpdateSsfStreamInput{
		EventTypes: eventTypes,
	}, time.Now().UTC())
	if err != nil {
		return writeAdminSharedSignalsError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toSsfStreamResponse(stream))
}

func (d Deps) handleDisableStream(c *echo.Context) error {
	return d.changeStreamStatus(c, func(id string, now time.Time) error {
		_, err := usecases.DisableSsfStream(c.Request().Context(), d.adminDeps(), id, now)
		return err
	})
}

func (d Deps) handleEnableStream(c *echo.Context) error {
	return d.changeStreamStatus(c, func(id string, now time.Time) error {
		_, err := usecases.EnableSsfStream(c.Request().Context(), d.adminDeps(), id, now)
		return err
	})
}

func (d Deps) handleDeleteStream(c *echo.Context) error {
	return d.changeStreamStatus(c, func(id string, now time.Time) error {
		return usecases.DeleteSsfStream(c.Request().Context(), d.adminDeps(), id, now)
	})
}

func (d Deps) changeStreamStatus(c *echo.Context, action func(id string, now time.Time) error) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := action(c.Param("stream_id"), time.Now().UTC()); err != nil {
		return writeAdminSharedSignalsError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

// ---- SecurityEventDelivery ----

type securityEventDeliveryResponse struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	StreamID      string     `json:"stream_id"`
	SetJTI        string     `json:"set_jti"`
	Status        string     `json:"status"`
	AttemptCount  int        `json:"attempt_count"`
	NextAttemptAt *time.Time `json:"next_attempt_at,omitempty"`
	LastError     *string    `json:"last_error,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	DeliveredAt   *time.Time `json:"delivered_at,omitempty"`
}

func toSecurityEventDeliveryResponse(d *ssdomain.SecurityEventDelivery) securityEventDeliveryResponse {
	return securityEventDeliveryResponse{
		ID: d.ID, TenantID: d.TenantID, StreamID: d.StreamID, SetJTI: d.SetJTI, Status: string(d.Status),
		AttemptCount: d.AttemptCount, NextAttemptAt: d.NextAttemptAt, LastError: d.LastError,
		CreatedAt: d.CreatedAt, DeliveredAt: d.DeliveredAt,
	}
}

func (d Deps) handleListDeliveries(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	deliveries, err := usecases.ListSecurityEventDeliveries(c.Request().Context(), d.adminDeps(), c.Param("stream_id"))
	if err != nil {
		return writeAdminSharedSignalsError(c, err)
	}
	out := make([]securityEventDeliveryResponse, len(deliveries))
	for i, del := range deliveries {
		out[i] = toSecurityEventDeliveryResponse(del)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"deliveries": out})
}

func writeAdminSharedSignalsError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, usecases.ErrStreamNotFound):
		return support.WriteBrowserError(c, http.StatusNotFound, "ssf_stream_not_found", "The SSF stream does not exist.")
	case errors.Is(err, usecases.ErrEventTypesEmpty):
		return support.WriteBrowserError(c, http.StatusBadRequest, "ssf_stream_event_types_required", "event_types must not be empty.")
	case errors.Is(err, usecases.ErrEventTypeInvalid):
		return support.WriteBrowserError(c, http.StatusBadRequest, "ssf_stream_event_type_invalid", "event_types contains an unknown CAEP event type.")
	case errors.Is(err, usecases.ErrDeliveryEndpointInvalid):
		return support.WriteBrowserError(c, http.StatusBadRequest, "ssf_transmitter_delivery_endpoint_invalid", "delivery_endpoint is required and must be an https URL.")
	case errors.Is(err, usecases.ErrAudienceRequired):
		return support.WriteBrowserError(c, http.StatusBadRequest, "ssf_transmitter_audience_required", "audience is required.")
	case errors.Is(err, usecases.ErrTrustedIssuerInvalid):
		return support.WriteBrowserError(c, http.StatusBadRequest, "ssf_receiver_trusted_issuer_invalid", "trusted_issuer is required and must be an https URL.")
	case errors.Is(err, usecases.ErrJWKSSourceRequired):
		return support.WriteBrowserError(c, http.StatusBadRequest, "ssf_receiver_jwks_required", "jwks_uri or jwks is required.")
	case errors.Is(err, usecases.ErrAcceptedAudiencesEmpty):
		return support.WriteBrowserError(c, http.StatusBadRequest, "ssf_receiver_accepted_audiences_required", "accepted_audiences must not be empty.")
	case errors.Is(err, usecases.ErrSecurityEventRejected):
		return support.WriteBrowserError(c, http.StatusBadRequest, "security_event_rejected", "The Security Event Token was rejected.")
	default:
		return err
	}
}
