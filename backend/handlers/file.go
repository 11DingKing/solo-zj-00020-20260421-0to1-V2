package handlers

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cloud-disk/config"
	"cloud-disk/database"
	"cloud-disk/models"

	"github.com/gofiber/fiber/v2"
)

const DefaultChunkSize = 2 * 1024 * 1024

type FileHandler struct {
	cfg *config.Config
}

func NewFileHandler(cfg *config.Config) *FileHandler {
	return &FileHandler{cfg: cfg}
}

func (h *FileHandler) UploadInit(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	filename := c.FormValue("filename")
	fileSize, _ := strconv.ParseInt(c.FormValue("fileSize"), 10, 64)
	folderID, _ := strconv.Atoi(c.FormValue("folderId"))

	if filename == "" || fileSize <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file parameters",
		})
	}

	var currentStorage int64
	err := database.DB.QueryRow(
		`SELECT storage_used FROM users WHERE id = $1`, userID,
	).Scan(&currentStorage)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check storage",
		})
	}

	maxStorage := h.cfg.MaxStorageMB * 1024 * 1024
	if currentStorage+fileSize > maxStorage {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":       "Storage limit exceeded",
			"currentUsed": formatSize(currentStorage),
			"maxStorage":  formatSize(maxStorage),
		})
	}

	totalChunks := (fileSize + DefaultChunkSize - 1) / DefaultChunkSize
	sessionID := generateSessionID(filename, userID)

	uploadDir := filepath.Join(h.cfg.UploadDir, "chunks", sessionID)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create upload directory",
		})
	}

	var existingFolderID interface{}
	if folderID > 0 {
		existingFolderID = folderID
	}

	var fileID int
	err = database.DB.QueryRow(`
		INSERT INTO files (user_id, folder_id, name, original_name, file_path, size, upload_status, upload_session_id, total_chunks, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'uploading', $7, $8, NOW(), NOW())
		RETURNING id
	`, userID, existingFolderID, filename, filename, "", fileSize, sessionID, totalChunks).Scan(&fileID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to initialize upload",
		})
	}

	return c.JSON(fiber.Map{
		"sessionId":   sessionID,
		"fileId":      fileID,
		"chunkSize":   DefaultChunkSize,
		"totalChunks": totalChunks,
	})
}

func (h *FileHandler) UploadChunk(c *fiber.Ctx) error {
	sessionID := c.FormValue("sessionId")
	chunkIndex, _ := strconv.Atoi(c.FormValue("chunkIndex"))

	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing session ID",
		})
	}

	file, err := c.FormFile("chunk")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to get chunk file",
		})
	}

	chunkDir := filepath.Join(h.cfg.UploadDir, "chunks", sessionID)
	chunkPath := filepath.Join(chunkDir, fmt.Sprintf("chunk_%d", chunkIndex))

	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to open chunk",
		})
	}
	defer src.Close()

	dst, err := os.Create(chunkPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save chunk",
		})
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to write chunk",
		})
	}

	return c.JSON(fiber.Map{
		"message":   "Chunk uploaded successfully",
		"chunkIndex": chunkIndex,
	})
}

func (h *FileHandler) UploadComplete(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	sessionID := c.FormValue("sessionId")
	fileID, _ := strconv.Atoi(c.FormValue("fileId"))

	if sessionID == "" || fileID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid parameters",
		})
	}

	var file models.File
	err := database.DB.QueryRow(`
		SELECT id, user_id, name, size, total_chunks FROM files WHERE id = $1 AND upload_session_id = $2
	`, fileID, sessionID).Scan(&file.ID, &file.UserID, &file.Name, &file.Size, &file.TotalChunks)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	if file.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	chunkDir := filepath.Join(h.cfg.UploadDir, "chunks", sessionID)
	finalDir := filepath.Join(h.cfg.UploadDir, "files", fmt.Sprintf("%d", userID))
	os.MkdirAll(finalDir, 0755)

	fileExt := filepath.Ext(file.Name)
	storedFileName := fmt.Sprintf("%d_%d%s", userID, time.Now().UnixNano(), fileExt)
	finalPath := filepath.Join(finalDir, storedFileName)

	outFile, err := os.Create(finalPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create final file",
		})
	}
	defer outFile.Close()

	for i := 0; i < int(file.TotalChunks.Int64); i++ {
		chunkPath := filepath.Join(chunkDir, fmt.Sprintf("chunk_%d", i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Missing chunk %d", i),
			})
		}

		if _, err = io.Copy(outFile, chunkFile); err != nil {
			chunkFile.Close()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to merge chunks",
			})
		}
		chunkFile.Close()
	}

	_, err = database.DB.Exec(`
		UPDATE files SET file_path = $1, upload_status = 'completed', updated_at = NOW() WHERE id = $2
	`, finalPath, fileID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update file status",
		})
	}

	_, err = database.DB.Exec(`
		UPDATE users SET storage_used = storage_used + $1 WHERE id = $2
	`, file.Size, userID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update storage",
		})
	}

	os.RemoveAll(chunkDir)

	return c.JSON(fiber.Map{
		"message": "File uploaded successfully",
		"fileId":  fileID,
	})
}

func (h *FileHandler) GetFiles(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	folderIDParam := c.Query("folderId")

	var folderID interface{}
	if folderIDParam != "" && folderIDParam != "0" {
		id, _ := strconv.Atoi(folderIDParam)
		folderID = id
	}

	rows, err := database.DB.Query(`
		SELECT id, name, size, download_count, created_at FROM files 
		WHERE user_id = $1 AND folder_id IS NOT DISTINCT FROM $2 AND upload_status = 'completed'
		ORDER BY created_at DESC
	`, userID, folderID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get files",
		})
	}
	defer rows.Close()

	var files []models.FileResponse
	for rows.Next() {
		var f models.File
		if err := rows.Scan(&f.ID, &f.Name, &f.Size, &f.DownloadCount, &f.CreatedAt); err != nil {
			continue
		}
		files = append(files, models.FileResponse{
			ID:           f.ID,
			Name:         f.Name,
			Size:         f.Size,
			SizeFormatted: formatSize(f.Size),
			DownloadCount: f.DownloadCount,
			CreatedAt:    f.CreatedAt.Format(time.RFC3339),
			Type:         getFileType(f.Name),
		})
	}

	var folders []models.FolderResponse
	folderRows, err := database.DB.Query(`
		SELECT id, name, created_at FROM folders WHERE user_id = $1 AND parent_id IS NOT DISTINCT FROM $2
		ORDER BY created_at DESC
	`, userID, folderID)

	if err == nil {
		defer folderRows.Close()
		for folderRows.Next() {
			var folder models.Folder
			if err := folderRows.Scan(&folder.ID, &folder.Name, &folder.CreatedAt); err != nil {
				continue
			}
			folders = append(folders, models.FolderResponse{
				ID:        folder.ID,
				Name:      folder.Name,
				CreatedAt: folder.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	return c.JSON(fiber.Map{
		"folders": folders,
		"files":   files,
	})
}

func (h *FileHandler) DownloadFile(c *fiber.Ctx) error {
	fileID, _ := strconv.Atoi(c.Params("id"))
	userID := c.Locals("user_id").(int)

	var file models.File
	err := database.DB.QueryRow(`
		SELECT id, user_id, name, file_path FROM files WHERE id = $1
	`, fileID).Scan(&file.ID, &file.UserID, &file.Name, &file.FilePath)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	if file.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	_, err = database.DB.Exec(`
		UPDATE files SET download_count = download_count + 1 WHERE id = $1
	`, fileID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update download count",
		})
	}

	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", file.Name))
	return c.SendFile(file.FilePath)
}

func (h *FileHandler) DeleteFile(c *fiber.Ctx) error {
	fileID, _ := strconv.Atoi(c.Params("id"))
	userID := c.Locals("user_id").(int)

	var file models.File
	err := database.DB.QueryRow(`
		SELECT id, user_id, file_path, size FROM files WHERE id = $1
	`, fileID).Scan(&file.ID, &file.UserID, &file.FilePath, &file.Size)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	if file.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	_, err = database.DB.Exec(`DELETE FROM files WHERE id = $1`, fileID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete file",
		})
	}

	_, err = database.DB.Exec(`
		UPDATE users SET storage_used = storage_used - $1 WHERE id = $2
	`, file.Size, userID)

	if file.FilePath != "" {
		os.Remove(file.FilePath)
	}

	return c.JSON(fiber.Map{
		"message": "File deleted successfully",
	})
}

func (h *FileHandler) GetStorageInfo(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)

	var storageUsed int64
	err := database.DB.QueryRow(`
		SELECT storage_used FROM users WHERE id = $1
	`, userID).Scan(&storageUsed)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get storage info",
		})
	}

	maxStorage := h.cfg.MaxStorageMB * 1024 * 1024

	return c.JSON(fiber.Map{
		"used":      storageUsed,
		"max":       maxStorage,
		"usedFormatted": formatSize(storageUsed),
		"maxFormatted":  formatSize(maxStorage),
		"percentage": float64(storageUsed) / float64(maxStorage) * 100,
	})
}

func (h *FileHandler) PreviewFile(c *fiber.Ctx) error {
	fileID, _ := strconv.Atoi(c.Params("id"))
	userID := c.Locals("user_id").(int)

	var file models.File
	err := database.DB.QueryRow(`
		SELECT id, user_id, name, file_path, size, created_at FROM files WHERE id = $1
	`, fileID).Scan(&file.ID, &file.UserID, &file.Name, &file.FilePath, &file.Size, &file.CreatedAt)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	if file.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	fileType := getFileType(file.Name)
	response := fiber.Map{
		"id":           file.ID,
		"name":         file.Name,
		"size":         file.Size,
		"sizeFormatted": formatSize(file.Size),
		"type":         fileType,
		"createdAt":    file.CreatedAt.Format(time.RFC3339),
	}

	if fileType == "image" {
		c.Set("Content-Type", getMimeType(file.Name))
		return c.SendFile(file.FilePath)
	} else if fileType == "text" {
		content, err := os.ReadFile(file.FilePath)
		if err != nil {
			response["content"] = "Unable to read file"
		} else {
			response["content"] = string(content)
		}
	}

	return c.JSON(response)
}

func (h *FileHandler) GetBreadcrumb(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	folderIDParam := c.Query("folderId")

	breadcrumb := []models.BreadcrumbItem{
		{ID: 0, Name: "根目录"},
	}

	if folderIDParam == "" || folderIDParam == "0" {
		return c.JSON(breadcrumb)
	}

	folderID, _ := strconv.Atoi(folderIDParam)

	for folderID > 0 {
		var folder models.Folder
		err := database.DB.QueryRow(`
			SELECT id, name, parent_id FROM folders WHERE id = $1 AND user_id = $2
		`, folderID, userID).Scan(&folder.ID, &folder.Name, &folder.ParentID)

		if err != nil {
			break
		}

		item := models.BreadcrumbItem{
			ID:   folder.ID,
			Name: folder.Name,
		}

		breadcrumb = append([]models.BreadcrumbItem{item}, breadcrumb[1:]...)
		breadcrumb = append([]models.BreadcrumbItem{{ID: 0, Name: "根目录"}}, breadcrumb...)

		if folder.ParentID == nil {
			break
		}
		folderID = *folder.ParentID
	}

	return c.JSON(breadcrumb)
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func generateSessionID(filename string, userID int) string {
	data := fmt.Sprintf("%s_%d_%d", filename, userID, time.Now().UnixNano())
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func getFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	imageExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true, ".webp": true, ".svg": true}
	textExts := map[string]bool{".txt": true, ".html": true, ".htm": true, ".css": true, ".js": true, ".json": true, ".xml": true, ".md": true, ".go": true, ".py": true, ".java": true, ".c": true, ".cpp": true, ".h": true, ".rs": true, ".ts": true, ".tsx": true, ".jsx": true, ".vue": true}
	docExts := map[string]bool{".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true}
	videoExts := map[string]bool{".mp4": true, ".avi": true, ".mkv": true, ".mov": true, ".webm": true}
	audioExts := map[string]bool{".mp3": true, ".wav": true, ".flac": true, ".aac": true, ".ogg": true}
	archiveExts := map[string]bool{".zip": true, ".rar": true, ".7z": true, ".tar": true, ".gz": true}

	if imageExts[ext] {
		return "image"
	} else if textExts[ext] {
		return "text"
	} else if docExts[ext] {
		return "document"
	} else if videoExts[ext] {
		return "video"
	} else if audioExts[ext] {
		return "audio"
	} else if archiveExts[ext] {
		return "archive"
	}
	return "other"
}

func getMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".bmp":  "image/bmp",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
	}
	if mimeType, ok := mimeTypes[ext]; ok {
		return mimeType
	}
	return "application/octet-stream"
}
