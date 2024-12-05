package local

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
)

type LocalFilesystem struct {
	Root    string // 根目录
	BaseUrl string // 基础URL
}

func NewStorage(root string, baseUrl string) *LocalFilesystem {
	fs := &LocalFilesystem{
		Root:    root,
		BaseUrl: baseUrl,
	}
	return fs
}

func (fs *LocalFilesystem) Put(ctx context.Context, path string, data []byte) error {
	// path包含了文件名，所以需要提取出路径的文件夹路径,然后进行创建
	fullPath := filepath.Join(fs.Root, path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return os.WriteFile(fullPath, data, 0644)
}

func (fs *LocalFilesystem) PutWithoutContext(path string, data []byte) error {
	return fs.Put(context.Background(), path, data)
}

func (fs *LocalFilesystem) Get(path string) ([]byte, error) {
	fullPath := filepath.Join(fs.Root, path)
	return os.ReadFile(fullPath)
}

// GetUrl 获取文件完整路径
// 当路径是绝对路径时，忽略Root配置
func (fs *LocalFilesystem) GetUrl(path string) string {
	if fs.BaseUrl != "" {
		return strings.TrimRight(fs.BaseUrl, "/") + "/" + strings.TrimPrefix(filepath.ToSlash(filepath.Clean(path)), "./")
	} else {
		fullPath, err := filepath.Abs(filepath.Join(fs.Root, path))
		if err != nil {
			return path
		}
		return fullPath
	}

}

// GetSignedUrl 获取签名URL
func (fs *LocalFilesystem) GetSignedUrl(path string, expires int64) (string, error) {
	// 这里我们简单返回一个 URL，实际应用中可能需要生成签名
	return fs.GetUrl(path), nil
}

// MustGetSignedUrl 获取签名URL
func (fs *LocalFilesystem) MustGetSignedUrl(path string, expires int64) string {
	url, err := fs.GetSignedUrl(path, expires)
	if err != nil {
		panic(err)
	}
	return url
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

// Delete 删除文件
func (fs *LocalFilesystem) Delete(path string) error {
	fullPath := filepath.Join(fs.Root, path)
	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}
