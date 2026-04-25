package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID           int
	Username     string
	PasswordHash string
	Email        sql.NullString
	StorageUsed  int64
	StorageLimit int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Folder struct {
	ID        int
	UserID    int
	ParentID  *int
	Name      string
	Path      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type File struct {
	ID               int
	UserID           int
	FolderID         *int
	Name             string
	OriginalName     string
	FilePath         string
	Size             int64
	MimeType         sql.NullString
	DownloadCount    int
	UploadStatus     string
	UploadSessionID  sql.NullString
	TotalChunks      sql.NullInt64
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
}

type ShareLink struct {
	ID            int
	Code          string
	FileID        int
	UserID        int
	AccessCode    sql.NullString
	ExpiresAt     time.Time
	DownloadCount int
	IsActive      bool
	CreatedAt     time.Time
}

type FileResponse struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Size          int64  `json:"size"`
	SizeFormatted string `json:"sizeFormatted"`
	DownloadCount int    `json:"downloadCount"`
	CreatedAt     string `json:"createdAt"`
	Type          string `json:"type"`
}

type FolderResponse struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

type BreadcrumbItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
