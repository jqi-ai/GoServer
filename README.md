# GoServer with Cloudflare R2 Storage

A Go server with image storage capabilities using Cloudflare R2.

## Features

- Upload images to Cloudflare R2
- Download images from storage
- Delete images
- List all stored images
- Generate presigned URLs for temporary access

## Setup

### 1. Prerequisites

- Go 1.23+ installed
- Cloudflare account with R2 enabled

### 2. Cloudflare R2 Configuration

1. Log in to your [Cloudflare Dashboard](https://dash.cloudflare.com/)
2. Navigate to R2 Storage
3. Create a new bucket (e.g., `my-images`)
4. Go to "Manage R2 API Tokens" and create a new API token with:
   - Permissions: Object Read & Write
   - Specify your bucket or allow all buckets
5. Save your credentials:
   - Account ID (found in your dashboard URL or R2 overview)
   - Access Key ID
   - Secret Access Key
   - Bucket Name

### 3. Environment Variables

Create a `.env` file in the project root (copy from `.env.example`):

```bash
cp .env.example .env
```

Edit `.env` with your R2 credentials:

```env
PORT=8080
R2_ACCOUNT_ID=your_cloudflare_account_id
R2_ACCESS_KEY_ID=your_r2_access_key_id
R2_SECRET_ACCESS_KEY=your_r2_secret_access_key
R2_BUCKET_NAME=your_bucket_name
```

### 4. Install Dependencies

```bash
go mod download
```

### 5. Run the Server

For development with environment variables:

```bash
# Windows PowerShell
$env:R2_ACCOUNT_ID="your_account_id"
$env:R2_ACCESS_KEY_ID="your_access_key"
$env:R2_SECRET_ACCESS_KEY="your_secret_key"
$env:R2_BUCKET_NAME="your_bucket_name"
go run server.go

# Linux/Mac
export R2_ACCOUNT_ID="your_account_id"
export R2_ACCESS_KEY_ID="your_access_key"
export R2_SECRET_ACCESS_KEY="your_secret_key"
export R2_BUCKET_NAME="your_bucket_name"
go run server.go
```

Or use a tool like [godotenv](https://github.com/joho/godotenv) to load `.env` files automatically.

## API Endpoints

### Health Check
```
GET /
```

### Upload Image
```
POST /api/images/upload
Content-Type: multipart/form-data
Body: image=<file>
```

Response:
```json
{
  "key": "images/1234567890_photo.jpg",
  "url": "https://...",  // Presigned URL (temporary)
  "message": "Image uploaded successfully"
}
```

### Download Image
```
GET /api/images/:key
```
Example: `GET /api/images/images/1234567890_photo.jpg`

### Delete Image
```
DELETE /api/images/:key
```

### List All Images
```
GET /api/images
```

Response:
```json
{
  "images": ["images/photo1.jpg", "images/photo2.png"],
  "count": 2
}
```

## Testing with cURL

### Upload an image:
```bash
curl -X POST http://localhost:8080/api/images/upload \
  -F "image=@/path/to/your/image.jpg"
```

### List images:
```bash
curl http://localhost:8080/api/images
```

### Download an image:
```bash
curl http://localhost:8080/api/images/images/1234567890_photo.jpg \
  --output downloaded.jpg
```

### Delete an image:
```bash
curl -X DELETE http://localhost:8080/api/images/images/1234567890_photo.jpg
```

## Deployment to Fly.io

1. Install Fly CLI: https://fly.io/docs/hands-on/install-flyctl/

2. Create `fly.toml`:
```toml
app = "your-app-name"

[env]
  PORT = "8080"

[http_service]
  internal_port = 8080
  force_https = true
```

3. Set secrets:
```bash
fly secrets set R2_ACCOUNT_ID="your_account_id"
fly secrets set R2_ACCESS_KEY_ID="your_access_key"
fly secrets set R2_SECRET_ACCESS_KEY="your_secret_key"
fly secrets set R2_BUCKET_NAME="your_bucket_name"
```

4. Deploy:
```bash
fly deploy
```

## Security Notes

- Never commit your `.env` file or credentials to version control
- Consider implementing authentication for production use
- Add rate limiting for upload endpoints
- Validate file sizes and implement upload limits
- Consider using CDN URLs instead of presigned URLs for public images

## License

MIT