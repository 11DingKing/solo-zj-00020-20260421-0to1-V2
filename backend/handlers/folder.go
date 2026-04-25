package handlers

import (
	"database/sql"
	"fmt"

	"cloud-disk/database"

	"github.com/gofiber/fiber/v2"
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
		err := database.DB.QueryRow(`
			SELECT path FROM folders WHERE id = $1 AND user_id = $2
		`, *req.ParentID, userID).Scan(&parentFolderPath)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Parent folder not found",
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
	err := database.DB.QueryRow(`
		SELECT name FROM folders WHERE id = $1 AND user_id = $2
	`, folderID, userID).Scan(&folderName)

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

	var hasFiles bool
	err = database.DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM files WHERE folder_id = $1)
	`, folderID).Scan(&hasFiles)

	if err == nil && hasFiles {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot delete non-empty folder",
		})
	}

	var hasSubFolders bool
	err = database.DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM folders WHERE parent_id = $1)
	`, folderID).Scan(&hasSubFolders)

	if err == nil && hasSubFolders {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot delete non-empty folder",
		})
	}

	result, err := database.DB.Exec(`
		DELETE FROM folders WHERE id = $1 AND user_id = $2
	`, folderID, userID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete folder",
		})
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Folder not found",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Folder deleted successfully",
	})
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
	err := database.DB.QueryRow(`
		SELECT path FROM folders WHERE id = $1 AND user_id = $2
	`, folderID, userID).Scan(&path)

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

	return c.JSON(fiber.Map{
		"path": path,
	})
}
