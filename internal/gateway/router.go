// Package gateway provides the API gateway for CYP-Docker-Registry.
package gateway

import (
	"cyp-docker-registry/internal/accelerator"
	"cyp-docker-registry/internal/common"
	"cyp-docker-registry/internal/detector"
	"cyp-docker-registry/internal/handler"
	"cyp-docker-registry/internal/middleware"
	"cyp-docker-registry/internal/registry"
	"cyp-docker-registry/internal/service"
	"cyp-docker-registry/internal/updater"
	"cyp-docker-registry/internal/version"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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
	p2pHandler         *handler.P2PHandler
	authService        *service.AuthService
	lockService        *service.LockService
	intrusionService   *service.IntrusionService
	auditService       *service.AuditService
	orgService         *service.OrgService
	shareService       *service.ShareService
	tokenService       *service.TokenService
	signatureService   *service.SignatureService
	sbomService        *service.SBOMService
	dnsService         *service.DNSService
	dnsHandler         *handler.DNSHandler
	p2pService         *service.P2PService
	globalService      *service.GlobalServiceManager
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

	// Initialize DNS service
	r.dnsService = service.NewDNSService(logger)

	// Initialize P2P service - 修复问题4
	p2pConfig := r.config.P2P
	if p2pConfig != nil && p2pConfig.Enabled {
		p2pSvc, err := service.NewP2PService(p2pConfig, r.config.Storage.BlobPath, logger)
		if err != nil {
			logger.Warn("P2P服务初始化失败", zap.Error(err))
		} else {
			r.p2pService = p2pSvc
			// 自动启动P2P服务
			if err := r.p2pService.Start(); err != nil {
				logger.Warn("P2P服务启动失败", zap.Error(err))
			} else {
				logger.Info("P2P服务已启动")
			}
		}
	}

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
	r.dnsHandler = handler.NewDNSHandler(r.dnsService)

	// Initialize P2P handler
	if r.p2pService != nil {
		r.p2pHandler = handler.NewP2PHandler(r.p2pService)
	}

	// Initialize global service manager and apply configurations
	r.globalService = service.NewGlobalServiceManager(logger)
	r.initGlobalServices()
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
	config := updater.DefaultConfig()

	// 从配置文件读取更新设置
	if r.config.Update.UpdateURL != "" {
		config.GitHubRepo = r.config.Update.UpdateURL
	}
	if r.config.Update.AutoUpdate {
		config.AutoUpdate = r.config.Update.AutoUpdate
	}
	if r.config.Update.CheckInterval != "" {
		// 解析检查间隔，如 "1h", "30m"
		if interval, err := time.ParseDuration(r.config.Update.CheckInterval); err == nil {
			config.CheckInterval = interval
		}
	}

	downloadPath := "./data/updates"
	service := updater.NewUpdaterService(config, downloadPath)

	// 启动后台更新检查
	service.Start()

	r.updaterHandler = updater.NewHandler(service)
}

// initGlobalServices 初始化全局服务并应用配置
// 修复问题3、4：DNS和P2P服务自动应用到系统
func (r *Router) initGlobalServices() {
	// 收集镜像加速源
	var acceleratorMirrors []string
	if r.config.Accelerator.Enabled {
		for _, u := range r.config.Accelerator.Upstreams {
			acceleratorMirrors = append(acceleratorMirrors, u.URL)
		}
	}

	// P2P配置
	p2pEnabled := false
	p2pPort := 4001
	if r.config.P2P != nil {
		p2pEnabled = r.config.P2P.Enabled
		if r.config.P2P.ListenPort > 0 {
			p2pPort = r.config.P2P.ListenPort
		}
	}

	// 初始化全局服务配置
	globalConfig := &service.GlobalServiceConfig{
		DataPath:           r.config.Storage.BlobPath,
		ConfigPath:         "./configs",
		AcceleratorEnabled: r.config.Accelerator.Enabled,
		AcceleratorMirrors: acceleratorMirrors,
		DNSEnabled:         true, // DNS服务默认启用
		DNSServers:         []string{"8.8.8.8", "8.8.4.4", "114.114.114.114"},
		P2PEnabled:         p2pEnabled,
		P2PListenPort:      p2pPort,
	}

	// 应用全局服务配置
	if err := r.globalService.Initialize(globalConfig); err != nil {
		logger.Warn("全局服务初始化失败", zap.Error(err))
	} else {
		logger.Info("全局服务已初始化并应用到系统")
	}
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
	r.engine.GET("/api/version/full", r.versionFullHandler)

	// Auth routes (no auth required)
	authGroup := r.engine.Group("/api/v1/auth")
	if r.authHandler != nil {
		r.authHandler.RegisterRoutes(authGroup)
	}

	// Lock management routes (no auth required for status check)
	lockGroup := r.engine.Group("/api/v1/system/lock")
	if r.lockHandler != nil {
		r.lockHandler.RegisterRoutes(lockGroup)
	}

	// Create simple auth check middleware for protected routes
	authCheckMiddleware := r.createAuthCheckMiddleware()

	// Audit routes (requires auth)
	auditGroup := r.engine.Group("/api/v1/audit")
	auditGroup.Use(authCheckMiddleware)
	if r.auditHandler != nil {
		r.auditHandler.RegisterRoutes(auditGroup)
	}

	// Organization routes (requires auth) - 修复问题1
	orgGroup := r.engine.Group("/api/v1/orgs")
	orgGroup.Use(authCheckMiddleware)
	if r.orgHandler != nil {
		r.orgHandler.RegisterRoutes(orgGroup)
	}

	// Share routes (requires auth) - 修复问题1
	shareGroup := r.engine.Group("/api/v1/share")
	shareGroup.Use(authCheckMiddleware)
	if r.shareHandler != nil {
		r.shareHandler.RegisterRoutes(shareGroup)
	}

	// Token routes (requires auth) - 修复问题1
	tokenGroup := r.engine.Group("/api/v1/tokens")
	tokenGroup.Use(authCheckMiddleware)
	if r.tokenHandler != nil {
		r.tokenHandler.RegisterRoutes(tokenGroup)
	}

	// WebSocket routes
	wsGroup := r.engine.Group("/api/v1")
	if r.wsHandler != nil {
		r.wsHandler.RegisterRoutes(wsGroup)
	}

	// Signature routes (requires auth)
	signatureGroup := r.engine.Group("/api/v1/signatures")
	signatureGroup.Use(authCheckMiddleware)
	if r.signatureHandler != nil {
		r.signatureHandler.RegisterRoutes(signatureGroup)
	}

	// SBOM routes (requires auth)
	sbomGroup := r.engine.Group("/api/v1/sbom")
	sbomGroup.Use(authCheckMiddleware)
	if r.sbomHandler != nil {
		r.sbomHandler.RegisterRoutes(sbomGroup)
	}

	// DNS routes (no auth required for DNS resolution)
	dnsGroup := r.engine.Group("/api/v1")
	if r.dnsHandler != nil {
		r.dnsHandler.RegisterRoutes(dnsGroup)
	}

	// P2P routes - 修复问题4
	p2pGroup := r.engine.Group("/api/v1")
	if r.p2pHandler != nil {
		r.p2pHandler.RegisterRoutes(p2pGroup)
	}

	// Global service status route
	r.engine.GET("/api/v1/global/status", r.globalServiceStatusHandler)
	r.engine.POST("/api/v1/global/apply/accelerator", authCheckMiddleware, r.applyAcceleratorHandler)
	r.engine.POST("/api/v1/global/apply/dns", authCheckMiddleware, r.applyDNSHandler)
	r.engine.POST("/api/v1/global/apply/p2p", authCheckMiddleware, r.applyP2PHandler)

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

	// Setup static file serving for frontend (must be last)
	r.setupStaticFiles()
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

// versionFullHandler handles full version API requests.
func (r *Router) versionFullHandler(c *gin.Context) {
	common.SuccessResponse(c, gin.H{
		"version":    version.GetVersion(),
		"build_time": version.BuildTime,
		"git_commit": version.GitCommit,
	})
}

// createAuthCheckMiddleware creates a simple authentication check middleware.
// 修复问题1：为组织管理、分享管理、访问令牌等路由添加认证检查
func (r *Router) createAuthCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if system is locked
		if r.lockService != nil && r.lockService.IsSystemLocked() {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":       "系统已锁定",
				"details":     "system_locked",
				"lock_reason": r.lockService.GetLockReason(),
			})
			return
		}

		// Check authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "缺少认证信息",
				"code":  "no_auth_header",
			})
			return
		}

		// Validate JWT token
		if r.authService != nil && strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			user, err := r.authService.ValidateJWT(tokenStr)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "JWT令牌无效",
					"code":  "invalid_jwt",
				})
				return
			}

			// Check if user is active
			if !user.IsActive {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "用户已被禁用",
					"code":  "inactive_user",
				})
				return
			}

			// Set user info in context
			c.Set("currentUser", user)
			c.Next()
			return
		}

		// Invalid authorization format
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "认证格式无效",
			"code":  "invalid_format",
		})
	}
}

// globalServiceStatusHandler 获取全局服务状态
func (r *Router) globalServiceStatusHandler(c *gin.Context) {
	if r.globalService == nil {
		common.ErrorResponse(c, common.ErrNotFound, gin.H{
			"message": "全局服务未初始化",
		})
		return
	}

	status := r.globalService.GetStatus()
	common.SuccessResponse(c, status)
}

// ApplyAcceleratorRequest 应用镜像加速请求
type ApplyAcceleratorRequest struct {
	Mirrors []string `json:"mirrors"`
}

// applyAcceleratorHandler 手动应用镜像加速配置
func (r *Router) applyAcceleratorHandler(c *gin.Context) {
	if r.globalService == nil {
		common.ErrorResponse(c, common.ErrNotFound, gin.H{
			"message": "全局服务未初始化",
		})
		return
	}

	var req ApplyAcceleratorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, common.ErrInvalidRequest, gin.H{
			"message": "无效的请求参数",
		})
		return
	}

	if err := r.globalService.ApplyAccelerator(req.Mirrors); err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"message": "应用镜像加速配置失败: " + err.Error(),
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "镜像加速配置已应用",
		"mirrors": req.Mirrors,
	})
}

// ApplyDNSRequest 应用DNS请求
type ApplyDNSRequest struct {
	Servers []string `json:"servers"`
}

// applyDNSHandler 手动应用DNS配置
func (r *Router) applyDNSHandler(c *gin.Context) {
	if r.globalService == nil {
		common.ErrorResponse(c, common.ErrNotFound, gin.H{
			"message": "全局服务未初始化",
		})
		return
	}

	var req ApplyDNSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, common.ErrInvalidRequest, gin.H{
			"message": "无效的请求参数",
		})
		return
	}

	if err := r.globalService.ApplyDNS(req.Servers); err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"message": "应用DNS配置失败: " + err.Error(),
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "DNS配置已应用",
		"servers": req.Servers,
	})
}

// ApplyP2PRequest 应用P2P请求
type ApplyP2PRequest struct {
	ListenPort int `json:"listen_port"`
}

// applyP2PHandler 手动应用P2P配置
func (r *Router) applyP2PHandler(c *gin.Context) {
	if r.globalService == nil {
		common.ErrorResponse(c, common.ErrNotFound, gin.H{
			"message": "全局服务未初始化",
		})
		return
	}

	var req ApplyP2PRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, common.ErrInvalidRequest, gin.H{
			"message": "无效的请求参数",
		})
		return
	}

	if err := r.globalService.ApplyP2P(req.ListenPort); err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"message": "应用P2P配置失败: " + err.Error(),
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"message":     "P2P配置已应用",
		"listen_port": req.ListenPort,
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

// setupStaticFiles configures static file serving for the frontend.
func (r *Router) setupStaticFiles() {
	// Try multiple possible paths for static files
	staticPaths := []string{
		"./web/dist",    // Development path
		"/app/web/dist", // Docker container path
		"web/dist",      // Relative path
	}

	var staticPath string
	for _, p := range staticPaths {
		if _, err := os.Stat(p); err == nil {
			staticPath = p
			break
		}
	}

	if staticPath == "" {
		logger.Warn("Static files directory not found, frontend will not be served")
		return
	}

	// Serve static assets (js, css, images, etc.)
	r.engine.Static("/assets", filepath.Join(staticPath, "assets"))

	// Serve favicon and other root static files
	r.engine.StaticFile("/favicon.ico", filepath.Join(staticPath, "favicon.ico"))
	r.engine.StaticFile("/vite.svg", filepath.Join(staticPath, "vite.svg"))

	// Serve robots.txt if exists
	robotsPath := filepath.Join(staticPath, "robots.txt")
	if _, err := os.Stat(robotsPath); err == nil {
		r.engine.StaticFile("/robots.txt", robotsPath)
	} else {
		// Provide default robots.txt
		r.engine.GET("/robots.txt", func(c *gin.Context) {
			c.String(http.StatusOK, "User-agent: *\nAllow: /")
		})
	}

	// Serve index.html for SPA routing
	indexPath := filepath.Join(staticPath, "index.html")
	r.engine.GET("/", func(c *gin.Context) {
		c.File(indexPath)
	})

	// Handle SPA routes - serve index.html for any unmatched routes
	r.engine.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip API and V2 routes
		if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/v2") || strings.HasPrefix(path, "/health") {
			common.ErrorResponse(c, common.ErrNotFound, gin.H{
				"path": path,
			})
			return
		}

		// Serve index.html for SPA routes
		c.File(indexPath)
	})

	logger.Info("Static files configured", zap.String("path", staticPath))
}
