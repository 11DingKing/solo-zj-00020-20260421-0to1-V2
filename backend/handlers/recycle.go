package handlers

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"cloud-disk/config"
	"cloud-disk/database"
	"cloud-disk/models"

	"github.com/gofiber/fiber/v2"
)

type RecycleHandler struct {
	cfg *config.Config
}

func NewRecycleHandler(cfg *config.Config) *RecycleHandler {
	return &RecycleHandler{cfg: cfg}
}

type RecycleItem struct {
	ID               int       `json:"id"`
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	Size             int64     `json:"size,omitempty"`
	SizeFormatted    string    `json:"sizeFormatted,omitempty"`
	DeletedAt        time.Time `json:"deletedAt"`
	RemainingDays    int       `json:"remainingDays"`
	OriginalParentID *int      `json:"originalParentId,omitempty"`
	OriginalFolderID *int      `json:"originalFolderId,omitempty"`
}

func (h *RecycleHandler) GetRecycleList(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)

	var items []RecycleItem

	fileRows, err := database.DB.Query(`
		SELECT id, name, size, folder_id, deleted_at 
		FROM files 
		WHERE user_id = $1 AND deleted_at IS NOT NULL
		AND (folder_id IS NULL OR folder_id NOT IN (
			SELECT id FROM folders WHERE user_id = $1 AND deleted_at IS NOT NULL
		))
		ORDER BY deleted_at DESC
	`, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get recycle files",
		})
	}
	defer fileRows.Close()

	for fileRows.Next() {
		var item RecycleItem
		var folderID *int
		var size sql.NullInt64
		if err := fileRows.Scan(&item.ID, &item.Name, &size, &folderID, &item.DeletedAt); err == nil {
			item.Type = "file"
			item.Size = size.Int64
			item.SizeFormatted = formatSize(size.Int64)
			item.OriginalFolderID = folderID
			item.RemainingDays = calculateRemainingDays(item.DeletedAt)
			items = append(items, item)
		}
	}

	folderRows, err := database.DB.Query(`
		SELECT id, name, parent_id, deleted_at 
		FROM folders 
		WHERE user_id = $1 AND deleted_at IS NOT NULL
		AND (parent_id IS NULL OR parent_id NOT IN (
			SELECT id FROM folders WHERE user_id = $1 AND deleted_at IS NOT NULL
		))
		ORDER BY deleted_at DESC
	`, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get recycle folders",
		})
	}
	defer folderRows.Close()

	for folderRows.Next() {
		var item RecycleItem
		var parentID *int
		if err := folderRows.Scan(&item.ID, &item.Name, &parentID, &item.DeletedAt); err == nil {
			item.Type = "folder"
			item.OriginalParentID = parentID
			item.RemainingDays = calculateRemainingDays(item.DeletedAt)
			items = append(items, item)
		}
	}

	return c.JSON(fiber.Map{
		"items": items,
	})
}

func (h *RecycleHandler) RestoreFile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	fileID, _ := strconv.Atoi(c.Params("id"))

	var file models.File
	var folderID *int
	var size int64
	err := database.DB.QueryRow(`
		SELECT id, user_id, name, folder_id, size, deleted_at FROM files WHERE id = $1
	`, fileID).Scan(&file.ID, &file.UserID, &file.Name, &folderID, &size, &file.DeletedAt)

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

	if file.DeletedAt == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File is not in recycle bin",
		})
	}

	restoreToRoot := false
	if folderID != nil {
		var parentDeletedAt *time.Time
		err = database.DB.QueryRow(`
			SELECT deleted_at FROM folders WHERE id = $1
		`, *folderID).Scan(&parentDeletedAt)
		if err == nil && parentDeletedAt != nil {
			restoreToRoot = true
		} else if err != nil && err != sql.ErrNoRows {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to check parent folder",
			})
		}
	}

	_, err = database.DB.Exec(`
		UPDATE files SET deleted_at = NULL, updated_at = NOW() WHERE id = $1
	`, fileID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to restore file",
		})
	}

	_, err = database.DB.Exec(`
		UPDATE users SET storage_used = storage_used + $1 WHERE id = $2
	`, size, userID)
	if err != nil {
		database.DB.Exec(`UPDATE files SET deleted_at = NOW() WHERE id = $1`, fileID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update storage",
		})
	}

	response := fiber.Map{
		"message": "File restored successfully",
	}
	if restoreToRoot {
		response["warning"] = "原文件夹已被删除，文件已恢复到根目录"
	}

	return c.JSON(response)
}

func (h *RecycleHandler) RestoreFolder(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	folderID, _ := strconv.Atoi(c.Params("id"))

	var folderName string
	var parentID *int
	var deletedAt *time.Time
	err := database.DB.QueryRow(`
		SELECT name, parent_id, deleted_at FROM folders WHERE id = $1 AND user_id = $2
	`, folderID, userID).Scan(&folderName, &parentID, &deletedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Folder not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check folder",
		})
	}

	if deletedAt == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Folder is not in recycle bin",
		})
	}

	restoreToRoot := false
	if parentID != nil {
		var parentDeletedAt *time.Time
		err = database.DB.QueryRow(`
			SELECT deleted_at FROM folders WHERE id = $1
		`, *parentID).Scan(&parentDeletedAt)
		if err == nil && parentDeletedAt != nil {
			restoreToRoot = true
		} else if err != nil && err != sql.ErrNoRows {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to check parent folder",
			})
		}
	}

	allFolderIDs, err := getAllSubFolderIDsIncludingSelf(folderID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get subfolders",
		})
	}

	var totalSize int64
	if len(allFolderIDs) > 0 {
		err = database.DB.QueryRow(`
			SELECT COALESCE(SUM(size), 0) FROM files 
			WHERE folder_id = ANY($1) AND deleted_at IS NOT NULL
		`, allFolderIDs).Scan(&totalSize)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to calculate folder size",
			})
		}
	}

	_, err = database.DB.Exec(`
		UPDATE files SET deleted_at = NULL, updated_at = NOW() 
		WHERE folder_id = ANY($1) AND deleted_at IS NOT NULL
	`, allFolderIDs)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to restore files in folder",
		})
	}

	if restoreToRoot {
		_, err = database.DB.Exec(`
			UPDATE folders SET deleted_at = NULL, parent_id = NULL, updated_at = NOW() 
			WHERE id = $1 AND deleted_at IS NOT NULL
		`, folderID)
	} else {
		_, err = database.DB.Exec(`
			UPDATE folders SET deleted_at = NULL, updated_at = NOW() 
			WHERE id = ANY($1) AND deleted_at IS NOT NULL
		`, allFolderIDs)
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to restore folders",
		})
	}

	if totalSize > 0 {
		_, err = database.DB.Exec(`
			UPDATE users SET storage_used = storage_used + $1 WHERE id = $2
		`, totalSize, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update storage",
			})
		}
	}

	response := fiber.Map{
		"message": "Folder restored successfully",
	}
	if restoreToRoot {
		response["warning"] = "原父文件夹已被删除，文件夹已恢复到根目录"
	}

	return c.JSON(response)
}

func (h *RecycleHandler) PermanentlyDeleteFile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	fileID, _ := strconv.Atoi(c.Params("id"))

	var filePath string
	var size int64
	var fileUserID int
	var deletedAt *time.Time
	err := database.DB.QueryRow(`
		SELECT file_path, size, user_id, deleted_at FROM files WHERE id = $1
	`, fileID).Scan(&filePath, &size, &fileUserID, &deletedAt)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	if fileUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	if deletedAt == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File is not in recycle bin",
		})
	}

	_, err = database.DB.Exec(`DELETE FROM files WHERE id = $1`, fileID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete file from database",
		})
	}

	if filePath != "" {
		os.Remove(filePath)
	}

	return c.JSON(fiber.Map{
		"message": "File permanently deleted successfully",
	})
}

func (h *RecycleHandler) PermanentlyDeleteFolder(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	folderID, _ := strconv.Atoi(c.Params("id"))

	var deletedAt *time.Time
	err := database.DB.QueryRow(`
		SELECT deleted_at FROM folders WHERE id = $1 AND user_id = $2
	`, folderID, userID).Scan(&deletedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Folder not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check folder",
		})
	}

	if deletedAt == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Folder is not in recycle bin",
		})
	}

	allFolderIDs, err := getAllSubFolderIDsIncludingSelf(folderID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get subfolders",
		})
	}

	fileRows, err := database.DB.Query(`
		SELECT file_path FROM files WHERE folder_id = ANY($1) AND deleted_at IS NOT NULL
	`, allFolderIDs)
	if err == nil {
		defer fileRows.Close()
		for fileRows.Next() {
			var filePath string
			if fileRows.Scan(&filePath) == nil && filePath != "" {
				os.Remove(filePath)
			}
		}
	}

	_, err = database.DB.Exec(`
		DELETE FROM files WHERE folder_id = ANY($1) AND deleted_at IS NOT NULL
	`, allFolderIDs)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete files from database",
		})
	}

	_, err = database.DB.Exec(`
		DELETE FROM folders WHERE id = ANY($1)
	`, allFolderIDs)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete folders from database",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Folder permanently deleted successfully",
	})
}

func (h *RecycleHandler) EmptyRecycleBin(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)

	fileRows, err := database.DB.Query(`
		SELECT file_path FROM files WHERE user_id = $1 AND deleted_at IS NOT NULL
	`, userID)
	if err == nil {
		defer fileRows.Close()
		for fileRows.Next() {
			var filePath string
			if fileRows.Scan(&filePath) == nil && filePath != "" {
				os.Remove(filePath)
			}
		}
	}

	_, err = database.DB.Exec(`
		DELETE FROM files WHERE user_id = $1 AND deleted_at IS NOT NULL
	`, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete files from recycle bin",
		})
	}

	_, err = database.DB.Exec(`
		DELETE FROM folders WHERE user_id = $1 AND deleted_at IS NOT NULL
	`, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete folders from recycle bin",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Recycle bin emptied successfully",
	})
}

func calculateRemainingDays(deletedAt time.Time) int {
	expiryTime := deletedAt.Add(30 * 24 * time.Hour)
	remaining := time.Until(expiryTime)
	if remaining <= 0 {
		return 0
	}
	return int(remaining.Hours() / 24)
}

func getAllSubFolderIDsIncludingSelf(parentID int, userID int) ([]int, error) {
	var allIDs []int
	var toProcess []int
	toProcess = append(toProcess, parentID)

	for len(toProcess) > 0 {
		currentID := toProcess[0]
		toProcess = toProcess[1:]
		allIDs = append(allIDs, currentID)

		rows, err := database.DB.Query(`
			SELECT id FROM folders 
			WHERE parent_id = $1 AND user_id = $2 AND deleted_at IS NOT NULL
		`, currentID, userID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err == nil {
				toProcess = append(toProcess, id)
			}
		}
	}

	return allIDs, nil
}

func StartCleanupTask(cfg *config.Config) {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			cleanupExpiredItems(cfg)
		}
	}()
}

func cleanupExpiredItems(cfg *config.Config) {
	cutoffTime := time.Now().Add(-30 * 24 * time.Hour)

	fileRows, err := database.DB.Query(`
		SELECT id, file_path FROM files WHERE deleted_at IS NOT NULL AND deleted_at < $1
	`, cutoffTime)
	if err != nil {
		fmt.Printf("Failed to query expired files: %v\n", err)
		return
	}
	defer fileRows.Close()

	var expiredFileIDs []int
	for fileRows.Next() {
		var id int
		var filePath string
		if err := fileRows.Scan(&id, &filePath); err == nil {
			expiredFileIDs = append(expiredFileIDs, id)
			if filePath != "" {
				os.Remove(filePath)
			}
		}
	}

	if len(expiredFileIDs) > 0 {
		_, err = database.DB.Exec(`
			DELETE FROM files WHERE id = ANY($1)
		`, expiredFileIDs)
		if err != nil {
			fmt.Printf("Failed to delete expired files: %v\n", err)
		}
	}

	_, err = database.DB.Exec(`
		DELETE FROM folders WHERE deleted_at IS NOT NULL AND deleted_at < $1
	`, cutoffTime)
	if err != nil {
		fmt.Printf("Failed to delete expired folders: %v\n", err)
	}
}
