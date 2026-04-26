package handlers

import (
	"database/sql"
	"fmt"
	"time"

	"cloud-disk/database"

	"github.com/gofiber/fiber/v2"
	"github.com/lib/pq"
)

type FolderHandler struct{}

func NewFolderHandler() *FolderHandler {
	return &FolderHandler{}
}

type CreateFolderRequest struct {
	Name     string `json:"name"`
	ParentID *int   `json:"parentId,omitempty"`
}

func (h *FolderHandler) CreateFolder(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)

	var req CreateFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Folder name is required",
		})
	}

	var parentPath string = "/"
	if req.ParentID != nil && *req.ParentID > 0 {
		var parentFolderPath string
		var parentDeletedAt *time.Time
		err := database.DB.QueryRow(`
			SELECT path, deleted_at FROM folders WHERE id = $1 AND user_id = $2
		`, *req.ParentID, userID).Scan(&parentFolderPath, &parentDeletedAt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Parent folder not found",
			})
		}
		if parentDeletedAt != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Parent folder has been deleted",
			})
		}
		if parentFolderPath == "/" {
			parentPath = "/" + req.Name
		} else {
			parentPath = parentFolderPath + "/" + req.Name
		}
	} else {
		parentPath = "/" + req.Name
		req.ParentID = nil
	}

	var existingID int
	err := database.DB.QueryRow(`
		SELECT id FROM folders 
		WHERE user_id = $1 AND parent_id IS NOT DISTINCT FROM $2 AND name = $3
	`, userID, req.ParentID, req.Name).Scan(&existingID)

	if err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Folder with this name already exists",
		})
	}

	var folderID int
	if req.ParentID != nil {
		err = database.DB.QueryRow(`
			INSERT INTO folders (user_id, parent_id, name, path, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
			RETURNING id
		`, userID, *req.ParentID, req.Name, parentPath).Scan(&folderID)
	} else {
		err = database.DB.QueryRow(`
			INSERT INTO folders (user_id, parent_id, name, path, created_at, updated_at)
			VALUES ($1, NULL, $2, $3, NOW(), NOW())
			RETURNING id
		`, userID, req.Name, parentPath).Scan(&folderID)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create folder",
		})
	}

	return c.JSON(fiber.Map{
		"message":  "Folder created successfully",
		"folderId": folderID,
	})
}

func (h *FolderHandler) DeleteFolder(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	folderID, _ := c.ParamsInt("id")

	if folderID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid folder ID",
		})
	}

	var folderName string
	var deletedAt *time.Time
	err := database.DB.QueryRow(`
		SELECT name, deleted_at FROM folders WHERE id = $1 AND user_id = $2
	`, folderID, userID).Scan(&folderName, &deletedAt)

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

	if deletedAt != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Folder already deleted",
		})
	}

	allFolderIDs, err := getAllSubFolderIDs(folderID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get subfolders",
		})
	}
	allFolderIDs = append(allFolderIDs, folderID)

	var totalSize int64
	if len(allFolderIDs) > 0 {
		err = database.DB.QueryRow(`
			SELECT COALESCE(SUM(size), 0) FROM files 
			WHERE folder_id = ANY($1) AND deleted_at IS NULL
		`, pq.Array(allFolderIDs)).Scan(&totalSize)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to calculate folder size",
			})
		}
	}

	_, err = database.DB.Exec(`
		UPDATE files SET deleted_at = NOW(), updated_at = NOW() 
		WHERE folder_id = ANY($1) AND deleted_at IS NULL
	`, pq.Array(allFolderIDs))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete files in folder",
		})
	}

	_, err = database.DB.Exec(`
		UPDATE folders SET deleted_at = NOW(), updated_at = NOW() 
		WHERE id = ANY($1) AND deleted_at IS NULL
	`, pq.Array(allFolderIDs))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete folders",
		})
	}

	if totalSize > 0 {
		_, err = database.DB.Exec(`
			UPDATE users SET storage_used = storage_used - $1 WHERE id = $2
		`, totalSize, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update storage",
			})
		}
	}

	return c.JSON(fiber.Map{
		"message": "Folder deleted successfully",
	})
}

func getAllSubFolderIDs(parentID int, userID int) ([]int, error) {
	var allIDs []int
	var toProcess []int
	toProcess = append(toProcess, parentID)

	for len(toProcess) > 0 {
		currentID := toProcess[0]
		toProcess = toProcess[1:]

		rows, err := database.DB.Query(`
			SELECT id FROM folders 
			WHERE parent_id = $1 AND user_id = $2 AND deleted_at IS NULL
		`, currentID, userID)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err == nil {
				allIDs = append(allIDs, id)
				toProcess = append(toProcess, id)
			}
		}
		rows.Close()
	}

	return allIDs, nil
}

func (h *FolderHandler) GetFolderPath(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	folderID, _ := c.ParamsInt("id")

	if folderID <= 0 {
		return c.JSON(fiber.Map{
			"path": "/",
		})
	}

	var path string
	var deletedAt *time.Time
	err := database.DB.QueryRow(`
		SELECT path, deleted_at FROM folders WHERE id = $1 AND user_id = $2
	`, folderID, userID).Scan(&path, &deletedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Folder not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get folder path: %v", err),
		})
	}

	if deletedAt != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Folder has been deleted",
		})
	}

	return c.JSON(fiber.Map{
		"path": path,
	})
}
