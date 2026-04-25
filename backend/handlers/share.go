package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"cloud-disk/cache"
	"cloud-disk/config"
	"cloud-disk/database"

	"github.com/gofiber/fiber/v2"
)

type ShareHandler struct {
	cfg *config.Config
}

func NewShareHandler(cfg *config.Config) *ShareHandler {
	return &ShareHandler{cfg: cfg}
}

type CreateShareRequest struct {
	FileID     int    `json:"fileId"`
	ExpireDays int    `json:"expireDays"`
	AccessCode string `json:"accessCode,omitempty"`
}

type CreateShareResponse struct {
	Code       string    `json:"code"`
	ExpiresAt  time.Time `json:"expiresAt"`
	AccessCode string    `json:"accessCode,omitempty"`
}

func (h *ShareHandler) CreateShare(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)

	var req CreateShareRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.FileID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File ID is required",
		})
	}

	var fileExists bool
	var fileName string
	var fileSize int64
	err := database.DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM files WHERE id = $1 AND user_id = $2 AND upload_status = 'completed'),
		       name, size
		FROM files WHERE id = $1
	`, req.FileID, userID).Scan(&fileExists, &fileName, &fileSize)

	if err != nil || !fileExists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "File not found or access denied",
		})
	}

	expireDays := req.ExpireDays
	if expireDays <= 0 {
		expireDays = 7
	}

	code, err := generateShareCode()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate share code",
		})
	}

	expiresAt := time.Now().Add(time.Duration(expireDays) * 24 * time.Hour)

	var accessCode interface{}
	if req.AccessCode != "" {
		accessCode = req.AccessCode
	}

	_, err = database.DB.Exec(`
		INSERT INTO share_links (code, file_id, user_id, access_code, expires_at, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, true, NOW())
	`, code, req.FileID, userID, accessCode, expiresAt)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create share link",
		})
	}

	ctx := context.Background()
	cacheKey := fmt.Sprintf("share:%s", code)
	cacheData := map[string]interface{}{
		"fileId":     req.FileID,
		"userId":     userID,
		"fileName":   fileName,
		"fileSize":   fileSize,
		"accessCode": req.AccessCode,
		"expiresAt":  expiresAt.Format(time.RFC3339),
	}

	for k, v := range cacheData {
		if v != "" && v != nil {
			cache.Redis.HSet(ctx, cacheKey, k, v)
		}
	}
	cache.Redis.Expire(ctx, cacheKey, time.Until(expiresAt))

	response := CreateShareResponse{
		Code:      code,
		ExpiresAt: expiresAt,
	}
	if req.AccessCode != "" {
		response.AccessCode = req.AccessCode
	}

	return c.JSON(response)
}

func (h *ShareHandler) GetShares(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)

	rows, err := database.DB.Query(`
		SELECT s.code, s.file_id, f.name, f.size, s.access_code, s.expires_at, 
		       s.download_count, s.is_active, s.created_at
		FROM share_links s
		JOIN files f ON s.file_id = f.id
		WHERE s.user_id = $1
		ORDER BY s.created_at DESC
	`, userID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get share links",
		})
	}
	defer rows.Close()

	var shares []fiber.Map
	for rows.Next() {
		var code string
		var fileID int
		var fileName string
		var fileSize int64
		var accessCode interface{}
		var expiresAt time.Time
		var downloadCount int
		var isActive bool
		var createdAt time.Time

		if err := rows.Scan(&code, &fileID, &fileName, &fileSize, &accessCode,
			&expiresAt, &downloadCount, &isActive, &createdAt); err != nil {
			continue
		}

		share := fiber.Map{
			"code":          code,
			"fileId":        fileID,
			"fileName":      fileName,
			"fileSize":      fileSize,
			"expiresAt":     expiresAt,
			"downloadCount": downloadCount,
			"isActive":      isActive,
			"createdAt":     createdAt,
		}
		if accessCode != nil {
			share["hasAccessCode"] = true
		}

		shares = append(shares, share)
	}

	return c.JSON(shares)
}

func (h *ShareHandler) DeleteShare(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	code := c.Params("code")

	result, err := database.DB.Exec(`
		DELETE FROM share_links WHERE code = $1 AND user_id = $2
	`, code, userID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete share link",
		})
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Share link not found",
		})
	}

	ctx := context.Background()
	cache.Redis.Del(ctx, fmt.Sprintf("share:%s", code))

	return c.JSON(fiber.Map{
		"message": "Share link deleted successfully",
	})
}

func (h *ShareHandler) GetShareInfo(c *fiber.Ctx) error {
	code := c.Params("code")

	ctx := context.Background()
	cacheKey := fmt.Sprintf("share:%s", code)

	cachedData, err := cache.Redis.HGetAll(ctx, cacheKey).Result()
	if err == nil && len(cachedData) > 0 {
		fileID, _ := strconv.Atoi(cachedData["fileId"])
		fileSize, _ := strconv.ParseInt(cachedData["fileSize"], 10, 64)
		expiresAt, _ := time.Parse(time.RFC3339, cachedData["expiresAt"])

		hasAccessCode := cachedData["accessCode"] != ""

		return c.JSON(fiber.Map{
			"share": fiber.Map{
				"code":         code,
				"fileId":       fileID,
				"fileName":     cachedData["fileName"],
				"fileSize":     fileSize,
				"expiresAt":    expiresAt,
				"isActive":     time.Now().Before(expiresAt),
			},
			"needsAccessCode": hasAccessCode,
		})
	}

	var shareCode string
	var fileID int
	var fileName string
	var fileSize int64
	var accessCode interface{}
	var expiresAt time.Time

	err = database.DB.QueryRow(`
		SELECT s.code, s.file_id, f.name, f.size, s.access_code, s.expires_at
		FROM share_links s
		JOIN files f ON s.file_id = f.id
		WHERE s.code = $1 AND s.is_active = true
	`, code).Scan(&shareCode, &fileID, &fileName, &fileSize, &accessCode, &expiresAt)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Share link not found or expired",
		})
	}

	if time.Now().After(expiresAt) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Share link has expired",
		})
	}

	hasAccessCode := accessCode != nil

	return c.JSON(fiber.Map{
		"share": fiber.Map{
			"code":         shareCode,
			"fileId":       fileID,
			"fileName":     fileName,
			"fileSize":     fileSize,
			"expiresAt":    expiresAt,
			"isActive":     true,
		},
		"needsAccessCode": hasAccessCode,
	})
}

func (h *ShareHandler) DownloadShare(c *fiber.Ctx) error {
	code := c.Params("code")

	var req struct {
		AccessCode string `json:"accessCode,omitempty"`
	}
	c.BodyParser(&req)

	ctx := context.Background()
	cacheKey := fmt.Sprintf("share:%s", code)

	var fileID int
	var fileName string
	var filePath string
	var storedAccessCode interface{}
	var expiresAt time.Time

	cachedData, cacheErr := cache.Redis.HGetAll(ctx, cacheKey).Result()
	if cacheErr == nil && len(cachedData) > 0 {
		fileID, _ = strconv.Atoi(cachedData["fileId"])
		fileName = cachedData["fileName"]
		storedAccessCode = cachedData["accessCode"]
		expiresAt, _ = time.Parse(time.RFC3339, cachedData["expiresAt"])

		err := database.DB.QueryRow(`
			SELECT file_path FROM files WHERE id = $1
		`, fileID).Scan(&filePath)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "File not found",
			})
		}
	} else {
		err := database.DB.QueryRow(`
			SELECT s.file_id, f.name, f.file_path, s.access_code, s.expires_at
			FROM share_links s
			JOIN files f ON s.file_id = f.id
			WHERE s.code = $1 AND s.is_active = true
		`, code).Scan(&fileID, &fileName, &filePath, &storedAccessCode, &expiresAt)

		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Share link not found or expired",
			})
		}
	}

	if time.Now().After(expiresAt) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Share link has expired",
		})
	}

	if storedAccessCode != nil {
		var storedCode string
		switch v := storedAccessCode.(type) {
		case string:
			storedCode = v
		case []byte:
			storedCode = string(v)
		}

		if storedCode != "" && storedCode != req.AccessCode {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid access code",
			})
		}
	}

	_, err := database.DB.Exec(`
		UPDATE share_links SET download_count = download_count + 1 WHERE code = $1
	`, code)
	if err == nil {
		database.DB.Exec(`
			UPDATE files SET download_count = download_count + 1 WHERE id = $1
		`, fileID)
	}

	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	return c.SendFile(filePath)
}

func generateShareCode() (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:12], nil
}
