package s3aws

import (
	"bytes"
	"context"
	"fmt"
	"go-boilerplate/internal/pkg/redis"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3Config struct {
	AWSRegion          string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
}

type S3Client struct {
	Client     *s3.S3
	BucketName string
	cancel     context.CancelFunc
	ctx        context.Context
	redis      redis.IRedis
}

type Is3 interface {
	GetBucketName() string
	UploadFile(fileName string, fileBytes []byte, contentType string) error
	GetPresignedURL(key string) (string, error)
}

func newSession(cfg S3Config) (*session.Session, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(cfg.AWSRegion),
		Credentials: credentials.NewStaticCredentials(
			cfg.AWSAccessKeyID,
			cfg.AWSSecretAccessKey,
			"",
		),
	})

	if err != nil {
		return nil, err
	}

	return sess, nil
}

func NewS3Client(ctx context.Context, cfg S3Config, bucketName string, redis redis.IRedis) (*S3Client, error) {
	sess, err := newSession(cfg)
	if err != nil {
		return nil, err
	}

	client := s3.New(sess)

	s3Client := &S3Client{
		Client:     client,
		BucketName: bucketName,
		ctx:        ctx,
		redis:      redis,
	}

	isBucketExists, err := CheckBucketExists(s3Client)
	if err != nil {
		return nil, err
	}

	if !isBucketExists {
		err = CreateBucket(s3Client)
		if err != nil {
			return nil, err
		}
	}

	return s3Client, nil

}
func CheckBucketExists(client *S3Client) (bool, error) {
	_, err := client.Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(client.BucketName),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket, "NotFound":
				return false, nil
			default:
				return false, err
			}
		}
		return false, err
	}

	return true, nil
}

func CreateBucket(client *S3Client) error {
	fmt.Println("Creating bucket:", client.BucketName)
	_, err := client.Client.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(client.BucketName),
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *S3Client) GetBucketName() string {
	return s.BucketName
}

func (s *S3Client) UploadFile(fileName string, fileBytes []byte, contentType string) error {
	_, err := s.Client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(s.BucketName),
		Key:         aws.String(fileName),
		Body:        bytes.NewReader(fileBytes),
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return nil
}
func (s *S3Client) GetPresignedURL(key string) (string, error) {
	keyCahce := fmt.Sprintf("s3:%s:%s", s.BucketName, key)
	cache, err := s.redis.Get(keyCahce)
	if err == nil && cache != "" && strings.HasPrefix(cache, "http") {
		return cache, nil
	}

	expired := 3 * 24 * time.Hour

	// Dapatkan content type secara dinamis
	contentType := getContentTypeFromKey(key)

	req, _ := s.Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket:                     aws.String(s.BucketName),
		Key:                        aws.String(key),
		ResponseContentType:        aws.String(contentType),
		ResponseContentDisposition: aws.String("inline"),
	})

	urlStr, err := req.Presign(expired)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	err = s.redis.Set(keyCahce, urlStr, expired)
	if err != nil {
		return "", fmt.Errorf("failed to cache presigned URL: %w", err)
	}

	return urlStr, nil
}

func getContentTypeFromKey(key string) string {
	ext := strings.ToLower(filepath.Ext(key))

	contentTypes := map[string]string{
		".pdf":  "application/pdf",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
		".txt":  "text/plain",
		".csv":  "text/csv",
		".json": "application/json",
		".xml":  "application/xml",
		".zip":  "application/zip",
		".rar":  "application/x-rar-compressed",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	}

	if contentType, exists := contentTypes[ext]; exists {
		return contentType
	}

	// Default fallback
	return "application/octet-stream"
}
