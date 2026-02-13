package controller

import (
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v2/web/service"
	"github.com/mhsanaei/3x-ui/v2/web/session"

	"github.com/gin-gonic/gin"
)

// APIController handles the main API routes for the 3x-ui panel, including inbounds and server management.
type APIController struct {
	BaseController
	inboundController *InboundController
	serverController  *ServerController
	inboundService    service.InboundService
	Tgbot             service.Tgbot
}

// NewAPIController creates a new APIController instance and initializes its routes.
func NewAPIController(g *gin.RouterGroup) *APIController {
	a := &APIController{}
	a.initRouter(g)
	return a
}

// checkAPIAuth is a middleware that returns 404 for unauthenticated API requests
// to hide the existence of API endpoints from unauthorized users
func (a *APIController) checkAPIAuth(c *gin.Context) {
	if !session.IsLogin(c) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Next()
}

// initRouter sets up the API routes for inbounds, server, and other endpoints.
func (a *APIController) initRouter(g *gin.RouterGroup) {
	// Public API group (no authentication required)
	public := g.Group("/panel/api/public")
	public.GET("/client-expiry", a.getClientExpiryByUUID)
	public.GET("/client-expiry/:uuid", a.getClientExpiryByUUID)

	// Main API group
	api := g.Group("/panel/api")
	api.Use(a.checkAPIAuth)

	// Inbounds API
	inbounds := api.Group("/inbounds")
	a.inboundController = NewInboundController(inbounds)

	// Server API
	server := api.Group("/server")
	a.serverController = NewServerController(server)

	// Extra routes
	api.GET("/backuptotgbot", a.BackuptoTgbot)
}

// getClientExpiryByUUID returns only expiration information for a client UUID.
// This endpoint is public and read-only.
func (a *APIController) getClientExpiryByUUID(c *gin.Context) {
	uuid := strings.TrimSpace(c.Param("uuid"))
	if uuid == "" {
		uuid = strings.TrimSpace(c.Query("uuid"))
	}
	if uuid == "" {
		pureJsonMsg(c, http.StatusBadRequest, false, "uuid is required")
		return
	}

	expiryTime, found, err := a.inboundService.GetClientExpiryByUUID(uuid)
	if err != nil {
		pureJsonMsg(c, http.StatusInternalServerError, false, "failed to query expiry time")
		return
	}
	if !found {
		pureJsonMsg(c, http.StatusNotFound, false, "uuid not found")
		return
	}

	response := gin.H{
		"uuid":       uuid,
		"expiryTime": expiryTime,
	}
	if expiryTime > 0 {
		expiryDateTime := time.UnixMilli(expiryTime).UTC()
		response["expiryDate"] = expiryDateTime.Format("2006-01-02")

		remainingMs := expiryTime - time.Now().UnixMilli()
		response["daysRemaining"] = int64(math.Ceil(float64(remainingMs) / 86400000.0))
		response["expired"] = remainingMs <= 0
	} else {
		response["expired"] = false
	}

	jsonObj(c, response, nil)
}

// BackuptoTgbot sends a backup of the panel data to Telegram bot admins.
func (a *APIController) BackuptoTgbot(c *gin.Context) {
	a.Tgbot.SendBackupToAdmins()
}
