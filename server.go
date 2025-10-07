package main

import (
	"context"
	"fmt"
	"go_server/storage"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	_ "github.com/joho/godotenv/autoload"
)

var r2Client *storage.R2Client

func init() {
	// Initialize R2 client
	accountID := os.Getenv("R2_ACCOUNT_ID")
	accessKeyID := os.Getenv("R2_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	bucketName := os.Getenv("R2_BUCKET_NAME")

	if accountID == "" || accessKeyID == "" || secretAccessKey == "" || bucketName == "" {
		fmt.Println("Warning: R2 credentials not configured. Please set environment variables.")
		return
	}

	var err error
	r2Client, err = storage.NewR2Client(accountID, accessKeyID, secretAccessKey, bucketName)
	if err != nil {
		fmt.Printf("Failed to initialize R2 client: %v\n", err)
	}
}

// UploadResponse represents the response after successful upload
type UploadResponse struct {
	Key     string `json:"key"`
	URL     string `json:"url,omitempty"`
	Message string `json:"message"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// uploadImage handles image upload
func uploadImage(c echo.Context) error {
	if r2Client == nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Storage service not configured",
		})
	}

	// Parse multipart form
	file, err := c.FormFile("image")
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "No image file provided",
		})
	}

	// Validate file type (optional)
	allowedTypes := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedTypes[ext] {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid file type. Only images are allowed",
		})
	}

	// Generate unique key for the file
	timestamp := time.Now().Unix()
	key := fmt.Sprintf("images/%d_%s", timestamp, file.Filename)

	// Upload to R2
	ctx := context.Background()
	err = r2Client.UploadMultipartFile(ctx, key, file)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: fmt.Sprintf("Failed to upload image: %v", err),
		})
	}

	// Generate presigned URL (optional - for direct access)
	url, _ := r2Client.GetPresignedURL(ctx, key, 60) // 60 minutes expiry

	return c.JSON(http.StatusOK, UploadResponse{
		Key:     key,
		URL:     url,
		Message: "Image uploaded successfully",
	})
}

// downloadImage handles image download
func downloadImage(c echo.Context) error {
	if r2Client == nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Storage service not configured",
		})
	}

	key := c.Param("key")
	if key == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Image key is required",
		})
	}

	ctx := context.Background()
	data, err := r2Client.DownloadFile(ctx, key)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "Image not found",
		})
	}

	// Determine content type based on extension
	contentType := "application/octet-stream"
	ext := strings.ToLower(filepath.Ext(key))
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	}

	return c.Blob(http.StatusOK, contentType, data)
}

// deleteImage handles image deletion
func deleteImage(c echo.Context) error {
	if r2Client == nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Storage service not configured",
		})
	}

	key := c.Param("key")
	if key == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Image key is required",
		})
	}

	ctx := context.Background()
	err := r2Client.DeleteFile(ctx, key)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: fmt.Sprintf("Failed to delete image: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Image deleted successfully",
	})
}

// listImages handles listing all images
func listImages(c echo.Context) error {
	if r2Client == nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Storage service not configured",
		})
	}

	ctx := context.Background()
	files, err := r2Client.ListFiles(ctx, "images/")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: fmt.Sprintf("Failed to list images: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"images": files,
		"count":  len(files),
	})
}

func basicAuthMiddleware(username, password string) echo.MiddlewareFunc {
	return middleware.BasicAuth(func(u, p string, ctx echo.Context) (bool, error) {
		if u == username && p == password {
			return true, nil
		}
		return false, nil
	})
}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	authUsername := os.Getenv("AUTH_USERNAME")
	authPassword := os.Getenv("AUTH_PASSWORD")
	if authUsername == "" || authPassword == "" {
		fmt.Println("Warning: Basic Auth credentials not set. Please configure AUTH_USERNAME and AUTH_PASSWORD.")
	} else {
		e.Use(basicAuthMiddleware(authUsername, authPassword))
	}

	// Routes
	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Image Storage API",
			"status":  "running",
		})
	})

	// Image endpoints
	api := e.Group("/api")
	api.POST("/images/upload", uploadImage)
	api.GET("/images/:key", downloadImage)
	api.DELETE("/images/:key", deleteImage)
	api.GET("/images", listImages)

	// Get port from environment variable or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	e.Logger.Fatal(e.Start(":" + port))
}
