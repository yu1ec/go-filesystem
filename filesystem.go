package filesystem

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/yu1ec/go-filesystem/config"
	"github.com/yu1ec/go-filesystem/driver/local"
	"github.com/yu1ec/go-filesystem/driver/qiniu"
	"github.com/yu1ec/go-filesystem/driver/webdav"

	"gopkg.in/yaml.v3"
)

// Filesystem 文件系统接口
type Filesystem interface {
	Put(ctx context.Context, path string, data []byte) error // 将数据写入文件
	PutWithoutContext(path string, data []byte) error        // 将数据写入文件不带上下文
	Get(path string) ([]byte, error)                         // 获取文件内容
	GetUrl(path string) string                               // 获取文件的URL

	// GetSignedUrl 获取签名URL
	// path: 文件路径
	// expires: 过期时间 单位/秒
	GetSignedUrl(path string, expires int64) (string, error)
	GetImageWidthHeight(path string) (int, int, error) // 获取图片的宽高

	MustGetSignedUrl(path string, expires int64) string // 获取签名URL

	Delete(path string) error // 删除文件
}

// NewStorage 创建文件系统
func NewStorage(driver config.FilesystemDriver) Filesystem {
	fs, _ := NewStorageWithError(driver)
	return fs
}

// NewStorageWithError 带错误信息的文件系统创建
func NewStorageWithError(driver config.FilesystemDriver) (Filesystem, error) {
	var fs Filesystem
	var err error
	switch driver.Name {
	case "local":
		var cfg config.LocalDriverConfig
		mapToStruct(driver.Config, &cfg)
		fs = local.NewStorage(cfg.Root, cfg.BaseUrl)
	case "qiniu":
		var cfg config.QiniuDriverConfig
		mapToStruct(driver.Config, &cfg)

		bucket := qiniu.Bucket{
			Name:            cfg.Bucket,
			Domain:          cfg.Domain,
			TimestampEncKey: cfg.TimestampEncKey,
			Private:         cfg.Private,
		}
		fs = qiniu.NewStorage(cfg.AccessKey, cfg.AccessSecret, bucket)
	case "webdav":
		var cfg config.WebdavDriverConfig
		mapToStruct(driver.Config, &cfg)
		fs, err = webdav.NewStorage(cfg.Uri, cfg.Username, cfg.Password)
	case "":
		panic("请正确选择文件系统配置")
	default:
		panic(driver.Name + "为不支持的文件系统")
	}

	return fs, err
}

// mapToStruct 手动将 map[string]any 转换为结构体
func mapToStruct(input any, output any) {
	data, _ := yaml.Marshal(input)
	yaml.Unmarshal(data, output)
}

// BuildUploadKey 生成上传文件的key
func BuildUploadKey(uploadDir, fileExt string) string {
	uploadDir = strings.Trim(uploadDir, "/")
	fileExt = strings.Trim(fileExt, ".")

	// 生成随机字节
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return ""
	}
	uniqueKey := hex.EncodeToString(randomBytes)
	datePath := time.Now().Format("2006/01/02")

	return fmt.Sprintf("%s/%s/%s.%s", uploadDir, datePath, uniqueKey, fileExt)
}

// AsQiniu 将通用文件系统转换为七牛云文件系统
// 如果不是七牛云文件系统，第二个返回值为 false
func AsQiniu(fs Filesystem) (*qiniu.QiniuFilesystem, bool) {
	if qn, ok := fs.(*qiniu.QiniuFilesystem); ok {
		return qn, true
	}
	return nil, false
}

func MustAsQiniu(fs Filesystem) *qiniu.QiniuFilesystem {
	qn, ok := AsQiniu(fs)
	if !ok {
		panic("文件系统不是七牛云文件系统")
	}
	return qn
}
