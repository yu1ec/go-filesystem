package local

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"net/url"
	"os"
	"path/filepath"
)

type LocalFilesystem struct {
	Root string // 根目录
}

func NewStorage(root string) *LocalFilesystem {
	fs := &LocalFilesystem{
		Root: root,
	}
	return fs
}

func (fs *LocalFilesystem) Put(ctx context.Context, path string, data []byte) error {
	// path包含了文件名，所以需要提取出路径的文件夹路径,然后进行创建
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func (fs *LocalFilesystem) PutWithoutContext(path string, data []byte) error {
	return fs.Put(context.Background(), path, data)
}

func (fs *LocalFilesystem) Get(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// GetUrl 获取文件完整路径
// 当路径是绝对路径时，忽略Root配置
func (fs *LocalFilesystem) GetUrl(path string) string {
	var fullPath string

	if filepath.IsAbs(path) {
		fullPath = path
	} else {
		fullPath = filepath.Join(fs.Root, path)
	}

	absolutePath, err := filepath.Abs(fullPath)
	if err != nil {
		// 如果无法获取绝对路径，则使用原始路径
		absolutePath = fullPath
	}

	return absolutePath
}

// GetSignedUrl 获取签名URL
func (fs *LocalFilesystem) GetSignedUrl(path string, expires int64) (string, error) {
	// 这里我们简单返回一个 URL，实际应用中可能需要生成签名
	return fs.GetUrl(path), nil
}

// MustGetSignedUrl 获取签名URL
func (fs *LocalFilesystem) MustGetSignedUrl(path string, expires int64) string {
	url, _ := fs.GetSignedUrl(path, expires)
	return url
}

func (fs *LocalFilesystem) GetPrivateUrl(path string, expires int64, query *url.Values) string {
	return path
}

func (fs *LocalFilesystem) GetImageWidthHeight(path string) (int, int, error) {
	data, err := fs.Get(path)
	if err != nil {
		return 0, 0, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, 0, err
	}

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	return width, height, nil
}
