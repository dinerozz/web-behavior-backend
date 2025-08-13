package download_extension

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/dinerozz/web-behavior-backend/internal/model/response/wrapper"
	"github.com/dinerozz/web-behavior-backend/internal/repository"
	"github.com/dinerozz/web-behavior-backend/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type ExtensionHandler struct {
	userRepo *repository.UserRepository
}

type ExtensionInfo struct {
	Version    string `json:"version"`
	FullCommit string `json:"full_commit"`
	BuildDate  string `json:"build_date"`
	Branch     string `json:"branch"`
	Repository string `json:"repository"`
	DeployedAt string `json:"deployed_at,omitempty"`
	SizeBytes  int64  `json:"size_bytes,omitempty"`
	SizeHuman  string `json:"size_human,omitempty"`
}

type DeployRequest struct {
	Version      string `json:"version" binding:"required"`
	ExtensionZip string `json:"extension_zip" binding:"required"` // base64 encoded
	InfoJSON     string `json:"info_json" binding:"required"`     // base64 encoded
}

type ExtensionStats struct {
	TotalDownloads int     `json:"total_downloads"`
	LastDownload   *string `json:"last_download"`
	ExtensionSize  int64   `json:"extension_size"`
	DeploymentDate string  `json:"deployment_date"`
	IsAvailable    bool    `json:"is_available"`
}

const (
	ExtensionDir      = "/var/lib/chrome-extension"
	ExtensionZipPath  = ExtensionDir + "/extension.zip"
	ExtensionInfoPath = ExtensionDir + "/info.json"
)

func NewExtensionHandler(userRepo *repository.UserRepository) *ExtensionHandler {
	return &ExtensionHandler{
		userRepo: userRepo,
	}
}

// VerifyAdmin - для nginx auth_request
// @Summary Verify admin access for nginx auth_request
// @Description Internal endpoint for nginx auth_request to verify admin access
// @Tags Chrome Extension
// @Accept json
// @Produce json
// @Success 200 "Admin access granted"
// @Failure 401 {object} wrapper.ErrorWrapper
// @Router /api/auth/verify-admin [get]
func (h *ExtensionHandler) VerifyAdmin(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{
			Message: "Missing authorization header",
			Success: false,
		})
		return
	}

	tokenString := authHeader
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		tokenString = authHeader[7:]
	}

	claims, err := utils.ValidateToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{
			Message: "Invalid token",
			Success: false,
		})
		return
	}

	userUUID, err := uuid.FromString(claims["user_id"].(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{
			Message: "Invalid user ID",
			Success: false,
		})
		return
	}

	isSuperAdmin, err := h.userRepo.IsUserSuperAdmin(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: "Failed to check admin status",
			Success: false,
		})
		return
	}

	if !isSuperAdmin {
		c.JSON(http.StatusForbidden, wrapper.ErrorWrapper{
			Message: "Super admin access required",
			Success: false,
		})
		return
	}

	c.Status(http.StatusOK)
}

// GetExtensionInfo - публичная информация о расширении
// @Summary Get Chrome Extension information
// @Description Get current Chrome Extension version and metadata (public endpoint)
// @Tags Chrome Extension
// @Accept json
// @Produce json
// @Success 200 {object} wrapper.ResponseWrapper{data=ExtensionInfo}
// @Failure 404 {object} wrapper.ErrorWrapper
// @Failure 500 {object} wrapper.ErrorWrapper
// @Router /api/extension/info [get]
func (h *ExtensionHandler) GetExtensionInfo(c *gin.Context) {
	if _, err := os.Stat(ExtensionInfoPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, wrapper.ErrorWrapper{
			Message: "Chrome extension not deployed yet",
			Success: false,
		})
		return
	}

	infoData, err := ioutil.ReadFile(ExtensionInfoPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: "Cannot read extension info",
			Success: false,
		})
		return
	}

	var info ExtensionInfo
	if err := json.Unmarshal(infoData, &info); err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: "Cannot parse extension info",
			Success: false,
		})
		return
	}

	if stat, err := os.Stat(ExtensionZipPath); err == nil {
		info.SizeBytes = stat.Size()
		info.SizeHuman = fmt.Sprintf("%.2f KB", float64(stat.Size())/1024)
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data:    info,
		Success: true,
	})
}

// GetExtensionStats - статистика для админов
// @Summary Get Chrome Extension statistics
// @Description Get Chrome Extension download statistics (admin only)
// @Tags Chrome Extension
// @Accept json
// @Produce json
// @Success 200 {object} wrapper.ResponseWrapper{data=ExtensionStats}
// @Failure 401 {object} wrapper.ErrorWrapper
// @Failure 500 {object} wrapper.ErrorWrapper
// @Router /api/extension/stats [get]
func (h *ExtensionHandler) GetExtensionStats(c *gin.Context) {
	stats := ExtensionStats{
		TotalDownloads: 0, // TODO: можно считать из логов nginx
		LastDownload:   nil,
		ExtensionSize:  0,
		IsAvailable:    false,
		DeploymentDate: time.Now().UTC().Format(time.RFC3339),
	}

	if stat, err := os.Stat(ExtensionZipPath); err == nil {
		stats.ExtensionSize = stat.Size()
		stats.IsAvailable = true
		stats.DeploymentDate = stat.ModTime().UTC().Format(time.RFC3339)
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data:    stats,
		Success: true,
	})
}

// DeployExtension - деплой через API (альтернатива SSH)
// @Summary Deploy Chrome Extension via API
// @Description Deploy Chrome Extension files via API (admin only)
// @Tags Chrome Extension
// @Accept json
// @Produce json
// @Param deployment body DeployRequest true "Deployment data"
// @Success 200 {object} wrapper.ResponseWrapper{data=ExtensionInfo}
// @Failure 400 {object} wrapper.ErrorWrapper
// @Failure 401 {object} wrapper.ErrorWrapper
// @Failure 500 {object} wrapper.ErrorWrapper
// @Router /api/extension/deploy [post]
func (h *ExtensionHandler) DeployExtension(c *gin.Context) {
	var req DeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: err.Error(),
			Success: false,
		})
		return
	}

	if err := os.MkdirAll(ExtensionDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: "Cannot create extension directory",
			Success: false,
		})
		return
	}

	zipData, err := base64.StdEncoding.DecodeString(req.ExtensionZip)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Invalid base64 data for extension zip",
			Success: false,
		})
		return
	}

	infoData, err := base64.StdEncoding.DecodeString(req.InfoJSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Invalid base64 data for info json",
			Success: false,
		})
		return
	}

	if err := ioutil.WriteFile(ExtensionZipPath, zipData, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: "Cannot save extension zip",
			Success: false,
		})
		return
	}

	var info ExtensionInfo
	if err := json.Unmarshal(infoData, &info); err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Invalid info.json format",
			Success: false,
		})
		return
	}

	info.DeployedAt = time.Now().UTC().Format(time.RFC3339)
	info.SizeBytes = int64(len(zipData))
	info.SizeHuman = fmt.Sprintf("%.2f KB", float64(len(zipData))/1024)

	updatedInfo, _ := json.MarshalIndent(info, "", "  ")

	if err := ioutil.WriteFile(ExtensionInfoPath, updatedInfo, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: "Cannot save extension info",
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data:    info,
		Success: true,
	})
}

// GetExtensionHealth - проверка состояния
// @Summary Check Chrome Extension health
// @Description Check if Chrome Extension files are available and valid
// @Tags Chrome Extension
// @Accept json
// @Produce json
// @Success 200 {object} wrapper.ResponseWrapper{data=map[string]interface{}}
// @Failure 503 {object} wrapper.ErrorWrapper
// @Router /api/extension/health [get]
func (h *ExtensionHandler) GetExtensionHealth(c *gin.Context) {
	health := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"status":    "unknown",
		"files": map[string]bool{
			"info.json":     false,
			"extension.zip": false,
		},
		"details": map[string]interface{}{},
	}

	filesOk := 0
	totalFiles := 2

	if _, err := os.Stat(ExtensionInfoPath); err == nil {
		health["files"].(map[string]bool)["info.json"] = true
		filesOk++
	}

	if stat, err := os.Stat(ExtensionZipPath); err == nil {
		health["files"].(map[string]bool)["extension.zip"] = true
		health["details"].(map[string]interface{})["zip_size"] = stat.Size()
		health["details"].(map[string]interface{})["zip_modified"] = stat.ModTime().UTC().Format(time.RFC3339)
		filesOk++
	}

	if filesOk == totalFiles {
		health["status"] = "healthy"
		c.JSON(http.StatusOK, wrapper.ResponseWrapper{
			Data:    health,
			Success: true,
		})
	} else if filesOk > 0 {
		health["status"] = "partial"
		c.JSON(http.StatusServiceUnavailable, wrapper.ErrorWrapper{
			Message: "Extension partially available",
			Success: false,
		})
	} else {
		health["status"] = "unavailable"
		c.JSON(http.StatusServiceUnavailable, wrapper.ErrorWrapper{
			Message: "Extension not available",
			Success: false,
		})
	}
}
