// Package registry provides container image registry functionality.
package registry

import (
	"cyp-docker-registry/internal/common"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for registry operations.
type Handler struct {
	service *Service
}

// NewHandler creates a new registry handler.
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
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
		h.v2Error(c, "MANIFEST_INVALID", "failed to read manifest", http.StatusBadRequest)
		return
	}

	manifest, err := h.service.PushManifest(name, reference, data)
	if err != nil {
		h.v2Error(c, "MANIFEST_INVALID", err.Error(), http.StatusBadRequest)
		return
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
		size, err := h.service.PushBlobWithDigest(digest, c.Request.Body)
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
		h.v2Error(c, "DIGEST_INVALID", "digest parameter required", http.StatusBadRequest)
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
