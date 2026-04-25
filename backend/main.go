package main

import (
	"log"
	"os"

	"cloud-disk/cache"
	"cloud-disk/config"
	"cloud-disk/database"
	"cloud-disk/handlers"
	"cloud-disk/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	cfg := config.Load()

	if err := database.Connect(cfg); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.DB.Close()

	if err := cache.Connect(cfg); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer cache.Redis.Close()

	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	app := fiber.New(fiber.Config{
		BodyLimit: 100 * 1024 * 1024,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	app.Use(logger.New())

	authHandler := handlers.NewAuthHandler(cfg)
	fileHandler := handlers.NewFileHandler(cfg)
	folderHandler := handlers.NewFolderHandler()
	shareHandler := handlers.NewShareHandler(cfg)

	api := app.Group("/api")

	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Get("/me", middleware.JWTMiddleware(cfg), authHandler.GetMe)

	api.Get("/shares/:code", middleware.OptionalJWTMiddleware(cfg), shareHandler.GetShareInfo)
	api.Post("/shares/:code/download", middleware.OptionalJWTMiddleware(cfg), shareHandler.DownloadShare)

	files := api.Group("/files", middleware.JWTMiddleware(cfg))
	files.Post("/upload/init", fileHandler.UploadInit)
	files.Post("/upload/chunk", fileHandler.UploadChunk)
	files.Post("/upload/complete", fileHandler.UploadComplete)
	files.Get("/", fileHandler.GetFiles)
	files.Get("/storage", fileHandler.GetStorageInfo)
	files.Get("/breadcrumb", fileHandler.GetBreadcrumb)
	files.Get("/preview/:id", fileHandler.PreviewFile)
	files.Get("/download/:id", fileHandler.DownloadFile)
	files.Delete("/:id", fileHandler.DeleteFile)

	folders := api.Group("/folders", middleware.JWTMiddleware(cfg))
	folders.Post("/", folderHandler.CreateFolder)
	folders.Delete("/:id", folderHandler.DeleteFolder)

	shares := api.Group("/shares", middleware.JWTMiddleware(cfg))
	shares.Post("/", shareHandler.CreateShare)
	shares.Get("/", shareHandler.GetShares)
	shares.Delete("/:code", shareHandler.DeleteShare)

	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := app.Listen(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
