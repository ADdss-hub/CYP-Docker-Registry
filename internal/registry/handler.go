// Package registry provides container image registry functionality.
package registry

import (
	"cyp-docker-registry/internal/common"
	"cyp-docker-registry/internal/service"
	"cyp-docker-registry/pkg/compression"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler provides HTTP handlers for registry operations.
type Handler struct {
	service          *Service
	signatureService *service.SignatureService
	sbomService      *service.SBOMService
	compressor       *compression.Compressor
	logger           *zap.Logger

	// 配置选项
	autoSign         bool
	autoGenerateSBOM bool
	autoCompress     bool
}

// HandlerConfig 配置选项
type HandlerConfig struct {
	AutoSign         bool
	AutoGenerateSBOM bool
	AutoCompress     bool
	CompressionLevel int
}

// NewHandler creates a new registry handler.
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// SetSignatureService 设置签名服务
func (h *Handler) SetSignatureService(svc *service.SignatureService) {
	h.signatureService = svc
}

// SetSBOMService 设置SBOM服务
func (h *Handler) SetSBOMService(svc *service.SBOMService) {
	h.sbomService = svc
}

// SetCompressor 设置压缩服务
func (h *Handler) SetCompressor(c *compression.Compressor) {
	h.compressor = c
}

// SetLogger 设置日志
func (h *Handler) SetLogger(logger *zap.Logger) {
	h.logger = logger
}

// Configure 配置Handler选项
func (h *Handler) Configure(config *HandlerConfig) {
	if config != nil {
		h.autoSign = config.AutoSign
		h.autoGenerateSBOM = config.AutoGenerateSBOM
		h.autoCompress = config.AutoCompress
	}
}

// RegisterRoutes registers registry routes on the given router groups.
func (h *Handler) RegisterRoutes(v2Group, apiGroup *gin.RouterGroup) {
	// Docker Registry V2 API routes
	h.registerV2Routes(v2Group)

	// Web API routes
	h.registerAPIRoutes(apiGroup)
}

// registerV2Routes registers Docker Registry V2 API routes.
func (h *Handler) registerV2Routes(v2 *gin.RouterGroup) {
	// Base endpoint - version check
	v2.GET("/", h.v2Base)

	// Manifest operations
	v2.GET("/:name/manifests/:reference", h.getManifest)
	v2.PUT("/:name/manifests/:reference", h.putManifest)
	v2.DELETE("/:name/manifests/:reference", h.deleteManifest)
	v2.HEAD("/:name/manifests/:reference", h.headManifest)

	// Blob operations
	v2.GET("/:name/blobs/:digest", h.getBlob)
	v2.HEAD("/:name/blobs/:digest", h.headBlob)
	v2.DELETE("/:name/blobs/:digest", h.deleteBlob)

	// Blob upload operations
	v2.POST("/:name/blobs/uploads/", h.startBlobUpload)
	v2.PATCH("/:name/blobs/uploads/:uuid", h.patchBlobUpload)
	v2.PUT("/:name/blobs/uploads/:uuid", h.completeBlobUpload)

	// Tags list
	v2.GET("/:name/tags/list", h.listTags)
}

// registerAPIRoutes registers Web API routes.
func (h *Handler) registerAPIRoutes(api *gin.RouterGroup) {
	images := api.Group("/images")
	{
		images.GET("", h.listImages)
		images.GET("/search", h.searchImages)
		images.GET("/:name", h.getImageDetails)
		images.GET("/:name/:tag", h.getImageByTag)
		images.DELETE("/:name/:tag", h.deleteImage)
	}
}

// ============================================================================
// Docker Registry V2 API Handlers
// ============================================================================

// v2Base handles the V2 API base endpoint.
func (h *Handler) v2Base(c *gin.Context) {
	c.Header("Docker-Distribution-API-Version", "registry/2.0")
	c.JSON(http.StatusOK, gin.H{})
}

// getManifest handles GET /v2/:name/manifests/:reference
func (h *Handler) getManifest(c *gin.Context) {
	name := c.Param("name")
	reference := c.Param("reference")

	data, manifest, err := h.service.PullManifest(name, reference)
	if err != nil {
		h.v2Error(c, "MANIFEST_UNKNOWN", err.Error(), http.StatusNotFound)
		return
	}

	imageRef := name + ":" + reference

	// 验证签名（如果签名服务启用且要求签名）
	if h.signatureService != nil && h.signatureService.IsSignatureRequired(imageRef) {
		req := &service.VerifyRequest{
			ImageRef: imageRef,
		}
		result, _ := h.signatureService.VerifyImage(req)
		if result != nil && !result.Verified {
			if h.logger != nil {
				h.logger.Warn("镜像签名验证失败", zap.String("image", imageRef), zap.String("error", result.Error))
			}
			// 根据配置决定是否阻止拉取
			// 当前仅记录警告，不阻止拉取
		}
	}

	c.Header("Docker-Distribution-API-Version", "registry/2.0")
	c.Header("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
	c.Header("Docker-Content-Digest", manifest.Digest)
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Data(http.StatusOK, "application/vnd.docker.distribution.manifest.v2+json", data)
}

// putManifest handles PUT /v2/:name/manifests/:reference
func (h *Handler) putManifest(c *gin.Context) {
	name := c.Param("name")
	reference := c.Param("reference")

	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.v2Error(c, "MANIFEST_INVALID", "读取清单数据失败", http.StatusBadRequest)
		return
	}

	manifest, err := h.service.PushManifest(name, reference, data)
	if err != nil {
		h.v2Error(c, "MANIFEST_INVALID", err.Error(), http.StatusBadRequest)
		return
	}

	imageRef := name + ":" + reference

	// 自动签名（如果启用）
	if h.autoSign && h.signatureService != nil {
		go func() {
			req := &service.SignRequest{
				ImageRef: imageRef,
				KeyID:    "default",
			}
			if _, err := h.signatureService.SignImage(req, 0, "system"); err != nil {
				if h.logger != nil {
					h.logger.Warn("自动签名失败", zap.String("image", imageRef), zap.Error(err))
				}
			} else {
				if h.logger != nil {
					h.logger.Info("镜像已自动签名", zap.String("image", imageRef))
				}
			}
		}()
	}

	// 自动生成SBOM（如果启用）
	if h.autoGenerateSBOM && h.sbomService != nil {
		go func() {
			req := &service.GenerateSBOMRequest{
				ImageRef: imageRef,
			}
			if _, err := h.sbomService.GenerateSBOM(req); err != nil {
				if h.logger != nil {
					h.logger.Warn("自动生成SBOM失败", zap.String("image", imageRef), zap.Error(err))
				}
			} else {
				if h.logger != nil {
					h.logger.Info("SBOM已自动生成", zap.String("image", imageRef))
				}
			}
		}()
	}

	c.Header("Docker-Distribution-API-Version", "registry/2.0")
	c.Header("Docker-Content-Digest", manifest.Digest)
	c.Header("Location", "/v2/"+name+"/manifests/"+manifest.Digest)
	c.Status(http.StatusCreated)
}

// deleteManifest handles DELETE /v2/:name/manifests/:reference
func (h *Handler) deleteManifest(c *gin.Context) {
	name := c.Param("name")
	reference := c.Param("reference")

	if err := h.service.DeleteImage(name, reference); err != nil {
		h.v2Error(c, "MANIFEST_UNKNOWN", err.Error(), http.StatusNotFound)
		return
	}

	c.Header("Docker-Distribution-API-Version", "registry/2.0")
	c.Status(http.StatusAccepted)
}

// headManifest handles HEAD /v2/:name/manifests/:reference
func (h *Handler) headManifest(c *gin.Context) {
	name := c.Param("name")
	reference := c.Param("reference")

	data, manifest, err := h.service.PullManifest(name, reference)
	if err != nil {
		h.v2Error(c, "MANIFEST_UNKNOWN", err.Error(), http.StatusNotFound)
		return
	}

	c.Header("Docker-Distribution-API-Version", "registry/2.0")
	c.Header("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
	c.Header("Docker-Content-Digest", manifest.Digest)
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Status(http.StatusOK)
}

// getBlob handles GET /v2/:name/blobs/:digest
func (h *Handler) getBlob(c *gin.Context) {
	digest := c.Param("digest")

	reader, size, err := h.service.PullBlob(digest)
	if err != nil {
		h.v2Error(c, "BLOB_UNKNOWN", err.Error(), http.StatusNotFound)
		return
	}
	defer reader.Close()

	c.Header("Docker-Distribution-API-Version", "registry/2.0")
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Docker-Content-Digest", digest)
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.DataFromReader(http.StatusOK, size, "application/octet-stream", reader, nil)
}

// headBlob handles HEAD /v2/:name/blobs/:digest
func (h *Handler) headBlob(c *gin.Context) {
	digest := c.Param("digest")

	reader, size, err := h.service.PullBlob(digest)
	if err != nil {
		h.v2Error(c, "BLOB_UNKNOWN", err.Error(), http.StatusNotFound)
		return
	}
	reader.Close()

	c.Header("Docker-Distribution-API-Version", "registry/2.0")
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Docker-Content-Digest", digest)
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.Status(http.StatusOK)
}

// deleteBlob handles DELETE /v2/:name/blobs/:digest
func (h *Handler) deleteBlob(c *gin.Context) {
	digest := c.Param("digest")

	if err := h.service.DeleteBlob(digest); err != nil {
		h.v2Error(c, "BLOB_UNKNOWN", err.Error(), http.StatusNotFound)
		return
	}

	c.Header("Docker-Distribution-API-Version", "registry/2.0")
	c.Status(http.StatusAccepted)
}

// startBlobUpload handles POST /v2/:name/blobs/uploads/
func (h *Handler) startBlobUpload(c *gin.Context) {
	name := c.Param("name")

	// Check for single POST upload with digest
	digest := c.Query("digest")
	if digest != "" {
		// Monolithic upload
		var reader io.Reader = c.Request.Body

		// 自动压缩（如果启用且数据未压缩）
		if h.autoCompress && h.compressor != nil {
			// 读取数据进行压缩
			data, err := io.ReadAll(c.Request.Body)
			if err != nil {
				h.v2Error(c, "BLOB_UPLOAD_INVALID", err.Error(), http.StatusBadRequest)
				return
			}

			// 检查是否已压缩
			if !compression.IsCompressed(data) {
				compressedData, err := h.compressor.Compress(data)
				if err == nil && len(compressedData) < len(data) {
					data = compressedData
					if h.logger != nil {
						h.logger.Debug("Blob已压缩", zap.String("digest", digest))
					}
				}
			}

			reader = strings.NewReader(string(data))
		}

		size, err := h.service.PushBlobWithDigest(digest, reader)
		if err != nil {
			h.v2Error(c, "BLOB_UPLOAD_INVALID", err.Error(), http.StatusBadRequest)
			return
		}

		c.Header("Docker-Distribution-API-Version", "registry/2.0")
		c.Header("Docker-Content-Digest", digest)
		c.Header("Content-Length", strconv.FormatInt(size, 10))
		c.Header("Location", "/v2/"+name+"/blobs/"+digest)
		c.Status(http.StatusCreated)
		return
	}

	// Start chunked upload - generate UUID
	uuid := generateUUID()
	c.Header("Docker-Distribution-API-Version", "registry/2.0")
	c.Header("Location", "/v2/"+name+"/blobs/uploads/"+uuid)
	c.Header("Docker-Upload-UUID", uuid)
	c.Header("Range", "0-0")
	c.Status(http.StatusAccepted)
}

// patchBlobUpload handles PATCH /v2/:name/blobs/uploads/:uuid
func (h *Handler) patchBlobUpload(c *gin.Context) {
	name := c.Param("name")
	uuid := c.Param("uuid")

	// For simplicity, we'll store the entire blob on PATCH
	// A full implementation would support chunked uploads
	digest, size, err := h.service.PushBlob(c.Request.Body)
	if err != nil {
		h.v2Error(c, "BLOB_UPLOAD_INVALID", err.Error(), http.StatusBadRequest)
		return
	}

	c.Header("Docker-Distribution-API-Version", "registry/2.0")
	c.Header("Location", "/v2/"+name+"/blobs/uploads/"+uuid)
	c.Header("Docker-Upload-UUID", uuid)
	c.Header("Range", "0-"+strconv.FormatInt(size-1, 10))
	c.Header("Docker-Content-Digest", digest)
	c.Status(http.StatusAccepted)
}

// completeBlobUpload handles PUT /v2/:name/blobs/uploads/:uuid
func (h *Handler) completeBlobUpload(c *gin.Context) {
	name := c.Param("name")
	digest := c.Query("digest")

	if digest == "" {
		h.v2Error(c, "DIGEST_INVALID", "缺少摘要参数", http.StatusBadRequest)
		return
	}

	// If there's body content, save it
	if c.Request.ContentLength > 0 {
		_, err := h.service.PushBlobWithDigest(digest, c.Request.Body)
		if err != nil {
			h.v2Error(c, "BLOB_UPLOAD_INVALID", err.Error(), http.StatusBadRequest)
			return
		}
	}

	c.Header("Docker-Distribution-API-Version", "registry/2.0")
	c.Header("Docker-Content-Digest", digest)
	c.Header("Location", "/v2/"+name+"/blobs/"+digest)
	c.Status(http.StatusCreated)
}

// listTags handles GET /v2/:name/tags/list
func (h *Handler) listTags(c *gin.Context) {
	name := c.Param("name")

	// Get all images for this name
	images, _, err := h.service.GetStorage().ListImages(1, 1000)
	if err != nil {
		h.v2Error(c, "NAME_UNKNOWN", err.Error(), http.StatusNotFound)
		return
	}

	var tags []string
	for _, img := range images {
		if img.Name == name {
			tags = append(tags, img.Tag)
		}
	}

	c.Header("Docker-Distribution-API-Version", "registry/2.0")
	c.JSON(http.StatusOK, gin.H{
		"name": name,
		"tags": tags,
	})
}

// ============================================================================
// Web API Handlers
// ============================================================================

// listImages handles GET /api/images
func (h *Handler) listImages(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	list, err := h.service.ListImages(page, pageSize)
	if err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"error": err.Error(),
		})
		return
	}

	common.SuccessResponse(c, list)
}

// searchImages handles GET /api/images/search
func (h *Handler) searchImages(c *gin.Context) {
	keyword := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	list, err := h.service.SearchImages(keyword, page, pageSize)
	if err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"error": err.Error(),
		})
		return
	}

	common.SuccessResponse(c, list)
}

// getImageDetails handles GET /api/images/:name
func (h *Handler) getImageDetails(c *gin.Context) {
	name := c.Param("name")

	// Get all tags for this image
	images, _, err := h.service.GetStorage().ListImages(1, 1000)
	if err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"error": err.Error(),
		})
		return
	}

	var tags []*ImageManifest
	for _, img := range images {
		if img.Name == name {
			tags = append(tags, img)
		}
	}

	if len(tags) == 0 {
		common.ErrorResponse(c, common.ErrImageNotFound, gin.H{
			"name": name,
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"name": name,
		"tags": tags,
	})
}

// getImageByTag handles GET /api/images/:name/:tag
func (h *Handler) getImageByTag(c *gin.Context) {
	name := c.Param("name")
	tag := c.Param("tag")

	manifest, err := h.service.GetImage(name, tag)
	if err != nil {
		common.ErrorResponse(c, common.ErrImageNotFound, gin.H{
			"name": name,
			"tag":  tag,
		})
		return
	}

	// Generate pull command
	pullCmd := "docker pull localhost:8080/" + name + ":" + tag

	common.SuccessResponse(c, gin.H{
		"image":    manifest,
		"pull_cmd": pullCmd,
	})
}

// deleteImage handles DELETE /api/images/:name/:tag
func (h *Handler) deleteImage(c *gin.Context) {
	name := c.Param("name")
	tag := c.Param("tag")

	if err := h.service.DeleteImage(name, tag); err != nil {
		if strings.Contains(err.Error(), "not found") {
			common.ErrorResponse(c, common.ErrImageNotFound, gin.H{
				"name": name,
				"tag":  tag,
			})
			return
		}
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"error": err.Error(),
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "镜像删除成功",
		"name":    name,
		"tag":     tag,
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

// v2Error sends a Docker Registry V2 API error response.
func (h *Handler) v2Error(c *gin.Context, code string, message string, status int) {
	c.Header("Docker-Distribution-API-Version", "registry/2.0")
	c.JSON(status, gin.H{
		"errors": []gin.H{
			{
				"code":    code,
				"message": message,
			},
		},
	})
}

// generateUUID generates a simple UUID for upload tracking.
func generateUUID() string {
	// Simple UUID generation - in production use a proper UUID library
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}
