package local

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"os"
	"path/filepath"
)

type LocalFilesystem struct {
	Root string // 根目录
}

func NewStoragte(root string) *LocalFilesystem {
	fs := &LocalFilesystem{
		Root: root,
	}
	return fs
}

func (fs *LocalFilesystem) Put(ctx context.Context, path string, data []byte) error {
	// TODO 待完善功能,创建文件对应的目录
	fullPath := filepath.Join(fs.Root, path)
	return os.WriteFile(fullPath, data, 0644)
}

func (fs *LocalFilesystem) PutWithoutContext(path string, data []byte) error {
	return fs.Put(context.Background(), path, data)
}

func (fs *LocalFilesystem) Get(path string) ([]byte, error) {
	fullPath := filepath.Join(fs.Root, path)
	return os.ReadFile(fullPath)
}

func (fs *LocalFilesystem) GetUrl(path string) string {
	return fmt.Sprintf("file://%s", filepath.Join(fs.Root, path))
}

// GetSignedUrl 获取签名URL
func (fs *LocalFilesystem) GetSignedUrl(path string, expires int64) (string, error) {
	// 这里我们简单返回一个 URL，实际应用中可能需要生成签名
	return fs.GetUrl(path), nil
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
