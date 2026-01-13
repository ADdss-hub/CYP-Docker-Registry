// Package gateway provides the API gateway for CYP-Registry.
package gateway

import (
	"cyp-registry/internal/accelerator"
	"cyp-registry/internal/common"
	"cyp-registry/internal/detector"
	"cyp-registry/internal/handler"
	"cyp-registry/internal/middleware"
	"cyp-registry/internal/registry"
	"cyp-registry/internal/service"
	"cyp-registry/internal/updater"
	"cyp-registry/internal/version"

	"github.com/gin-gonic/gin"
)

// Router represents the API gateway router.
type Router struct {
	engine             *gin.Engine
	config             *common.Config
	registryHandler    *registry.Handler
	acceleratorHandler *accelerator.Handler
	detectorHandler    *detector.Handler
	updaterHandler     *updater.Handler
	authHandler        *handler.AuthHandler
	lockHandler        *handler.LockHandler
	auditHandler       *handler.AuditHandler
	orgHandler         *handler.OrgHandler
	shareHandler       *handler.ShareHandler
	tokenHandler       *handler.TokenHandler
	wsHandler          *handler.WSHandler
	signatureHandler   *handler.SignatureHandler
	sbomHandler        *handler.SBOMHandler
	authService        *service.AuthService
	lockService        *service.LockService
	intrusionService   *service.IntrusionService
	auditService       *service.AuditService
	orgService         *service.OrgService
	shareService       *service.ShareService
	tokenService       *service.TokenService
	signatureService   *service.SignatureService
	sbomService        *service.SBOMService
}

// NewRouter creates a new Router instance.
func NewRouter(config *common.Config) *Router {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	r := &Router{
		engine: engine,
		config: config,
	}

	// Initialize security services
	r.initSecurityServices()

	// Initialize registry
	storage, err := registry.NewStorage(config.Storage.BlobPath, config.Storage.MetaPath)
	if err == nil {
		service := registry.NewService(storage)
		r.registryHandler = registry.NewHandler(service)
	}

	// Initialize accelerator
	if config.Accelerator.Enabled {
		r.initAccelerator()
	}

	// Initialize detector
	r.initDetector()

	// Initialize updater
	r.initUpdater()

	r.setupMiddleware()
	r.setupRoutes()

	return r
}

// initSecurityServices initializes security-related services.
func (r *Router) initSecurityServices() {
	// Initialize lock service
	r.lockService = service.NewLockService(logger)

	// Initialize intrusion service
	intrusionConfig := &service.IntrusionConfig{
		Enabled:          true,
		MaxLoginAttempts: 3,
		MaxTokenAttempts: 5,
		MaxAPIAttempts:   10,
		ProgressiveDelay: true,
	}
	r.intrusionService = service.NewIntrusionService(intrusionConfig, r.lockService, logger)

	// Initialize audit service
	auditConfig := &service.AuditConfig{
		LogAllRequests: true,
		LogFailedAuth:  true,
		LogLockEvents:  true,
		BlockchainHash: true,
	}
	r.auditService, _ = service.NewAuditService(auditConfig, logger)

	// Initialize auth service
	jwtSecret := "cyp-registry-secret-key" // TODO: Load from config
	r.authService = service.NewAuthService(jwtSecret)

	// Initialize org service
	r.orgService = service.NewOrgService(logger)

	// Initialize share service
	r.shareService = service.NewShareService(logger)

	// Initialize token service
	r.tokenService = service.NewTokenService(logger)

	// Initialize signature service
	signatureConfig := &service.SignatureConfig{
		Enabled:          true,
		Mode:             "warn",
		AutoSign:         false,
		RequireSignature: false,
		KeyPath:          "./data/signatures",
	}
	r.signatureService = service.NewSignatureService(signatureConfig, logger)

	// Initialize SBOM service
	sbomConfig := &service.SBOMConfig{
		Enabled:     true,
		Generator:   "syft",
		Format:      "spdx-json",
		VulnScan:    true,
		VulnScanner: "trivy",
		StoragePath: "./data/sboms",
	}
	r.sbomService = service.NewSBOMService(sbomConfig, logger)

	// Initialize handlers
	r.authHandler = handler.NewAuthHandler(r.authService, r.lockService, r.intrusionService, r.auditService)
	r.lockHandler = handler.NewLockHandler(r.lockService, r.auditService)
	r.auditHandler = handler.NewAuditHandler()
	r.orgHandler = handler.NewOrgHandler(r.orgService, r.auditService)
	r.shareHandler = handler.NewShareHandler(r.shareService, r.auditService)
	r.tokenHandler = handler.NewTokenHandler(r.tokenService, r.auditService)
	r.wsHandler = handler.NewWSHandler(logger)
	r.signatureHandler = handler.NewSignatureHandler(r.signatureService, r.auditService)
	r.sbomHandler = handler.NewSBOMHandler(r.sbomService, r.auditService)
}

// initAccelerator initializes the accelerator service.
func (r *Router) initAccelerator() {
	// Parse max cache size (default 10GB)
	maxCacheSize := parseSize(r.config.Storage.MaxCacheSize)
	if maxCacheSize == 0 {
		maxCacheSize = 10 * 1024 * 1024 * 1024 // 10GB default
	}

	cache, err := accelerator.NewLRUCache(r.config.Storage.CachePath, maxCacheSize)
	if err != nil {
		return
	}

	proxy, err := accelerator.NewProxyService(cache, r.config.Storage.CachePath)
	if err != nil {
		return
	}

	// Set upstreams from config
	var upstreams []accelerator.UpstreamSource
	for _, u := range r.config.Accelerator.Upstreams {
		upstreams = append(upstreams, accelerator.UpstreamSource{
			Name:     u.Name,
			URL:      u.URL,
			Priority: u.Priority,
			Enabled:  true,
		})
	}
	if len(upstreams) > 0 {
		proxy.SetUpstreams(upstreams)
	}

	r.acceleratorHandler = accelerator.NewHandler(proxy)
}

// initDetector initializes the detector service.
func (r *Router) initDetector() {
	service := detector.NewDetectorService()
	r.detectorHandler = detector.NewHandler(service)
}

// initUpdater initializes the updater service.
func (r *Router) initUpdater() {
	updateURL := ""
	if r.config.Update.UpdateURL != "" {
		updateURL = r.config.Update.UpdateURL
	}

	downloadPath := "./data/updates"
	service := updater.NewUpdaterService(updateURL, downloadPath)
	r.updaterHandler = updater.NewHandler(service)
}

// parseSize parses a size string like "10GB" into bytes.
func parseSize(s string) int64 {
	if s == "" {
		return 0
	}

	var multiplier int64 = 1
	numStr := s

	if len(s) >= 2 {
		suffix := s[len(s)-2:]
		switch suffix {
		case "GB", "gb":
			multiplier = 1024 * 1024 * 1024
			numStr = s[:len(s)-2]
		case "MB", "mb":
			multiplier = 1024 * 1024
			numStr = s[:len(s)-2]
		case "KB", "kb":
			multiplier = 1024
			numStr = s[:len(s)-2]
		}
	}

	var num int64
	for _, c := range numStr {
		if c >= '0' && c <= '9' {
			num = num*10 + int64(c-'0')
		}
	}

	return num * multiplier
}

// setupMiddleware configures middleware for the router.
func (r *Router) setupMiddleware() {
	r.engine.Use(LoggingMiddleware())
	r.engine.Use(ErrorHandlingMiddleware())
	r.engine.Use(gin.Recovery())
	r.engine.Use(CORSMiddleware())

	// Security middleware
	securityMw := middleware.NewSecurityMiddleware(false)
	r.engine.Use(securityMw.SecurityHeaders())

	// Lock check middleware
	lockMw := middleware.NewLockMiddleware(r.lockService)
	r.engine.Use(lockMw.CheckLock())
}

// setupRoutes configures all routes for the API gateway.
func (r *Router) setupRoutes() {
	// Health check endpoint (no auth required)
	r.engine.GET("/health", r.healthHandler)

	// Version API endpoint (no auth required)
	r.engine.GET("/api/version", r.versionHandler)

	// Auth routes (no auth required)
	authGroup := r.engine.Group("/api/v1/auth")
	if r.authHandler != nil {
		r.authHandler.RegisterRoutes(authGroup)
	}

	// Lock management routes
	lockGroup := r.engine.Group("/api/v1/system/lock")
	if r.lockHandler != nil {
		r.lockHandler.RegisterRoutes(lockGroup)
	}

	// Audit routes
	auditGroup := r.engine.Group("/api/v1/audit")
	if r.auditHandler != nil {
		r.auditHandler.RegisterRoutes(auditGroup)
	}

	// Organization routes
	orgGroup := r.engine.Group("/api/v1/orgs")
	if r.orgHandler != nil {
		r.orgHandler.RegisterRoutes(orgGroup)
	}

	// Share routes
	shareGroup := r.engine.Group("/api/v1/share")
	if r.shareHandler != nil {
		r.shareHandler.RegisterRoutes(shareGroup)
	}

	// Token routes
	tokenGroup := r.engine.Group("/api/v1/tokens")
	if r.tokenHandler != nil {
		r.tokenHandler.RegisterRoutes(tokenGroup)
	}

	// WebSocket routes
	wsGroup := r.engine.Group("/api/v1")
	if r.wsHandler != nil {
		r.wsHandler.RegisterRoutes(wsGroup)
	}

	// Signature routes
	signatureGroup := r.engine.Group("/api/v1/signatures")
	if r.signatureHandler != nil {
		r.signatureHandler.RegisterRoutes(signatureGroup)
	}

	// SBOM routes
	sbomGroup := r.engine.Group("/api/v1/sbom")
	if r.sbomHandler != nil {
		r.sbomHandler.RegisterRoutes(sbomGroup)
	}

	// Docker Registry V2 API routes
	v2 := r.engine.Group("/v2")
	{
		// Register registry routes if handler is available
		if r.registryHandler != nil {
			r.registryHandler.RegisterRoutes(v2, r.engine.Group("/api"))
		} else {
			v2.GET("/", r.v2BaseHandler)
			v2.Any("/*path", r.v2PlaceholderHandler)
		}
	}

	// Web API routes (for non-registry endpoints)
	api := r.engine.Group("/api")
	{
		// Accelerator management
		accel := api.Group("/accel")
		if r.acceleratorHandler != nil {
			r.acceleratorHandler.RegisterRoutes(accel)
		} else {
			accel.Any("/*path", r.apiPlaceholderHandler)
		}

		// System information
		system := api.Group("/system")
		if r.detectorHandler != nil {
			r.detectorHandler.RegisterRoutes(system)
		} else {
			system.Any("/*path", r.apiPlaceholderHandler)
		}

		// Update management
		update := api.Group("/update")
		if r.updaterHandler != nil {
			r.updaterHandler.RegisterRoutes(update)
		} else {
			update.Any("/*path", r.apiPlaceholderHandler)
		}
	}
}

// Engine returns the underlying gin engine.
func (r *Router) Engine() *gin.Engine {
	return r.engine
}

// healthHandler handles health check requests.
func (r *Router) healthHandler(c *gin.Context) {
	common.SuccessResponse(c, gin.H{
		"status":  "healthy",
		"version": version.GetVersion(),
	})
}

// versionHandler handles version API requests.
func (r *Router) versionHandler(c *gin.Context) {
	common.SuccessResponse(c, gin.H{
		"version":      version.GetVersion(),
		"full_version": version.GetFullVersion(),
	})
}

// v2BaseHandler handles Docker Registry V2 base endpoint.
func (r *Router) v2BaseHandler(c *gin.Context) {
	c.JSON(200, gin.H{})
}

// v2PlaceholderHandler is a placeholder for V2 registry routes.
func (r *Router) v2PlaceholderHandler(c *gin.Context) {
	common.ErrorResponse(c, common.ErrNotFound, gin.H{
		"path": c.Param("path"),
	})
}

// apiPlaceholderHandler is a placeholder for API routes.
func (r *Router) apiPlaceholderHandler(c *gin.Context) {
	common.ErrorResponse(c, common.ErrNotFound, gin.H{
		"path": c.FullPath(),
	})
}
