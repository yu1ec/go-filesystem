package webdav

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/studio-b12/gowebdav"
)

type WebdavFilesystem struct {
	uri      string
	username string
	password string
	client   *gowebdav.Client
}

func NewStorage(uri, username, password string) (*WebdavFilesystem, error) {
	fs := &WebdavFilesystem{
		uri:      uri,
		username: username,
		password: password,
	}
	fs.client = gowebdav.NewClient(uri, username, password)

	if err := fs.client.Connect(); err != nil {
		return nil, err
	}

	return fs, nil
}

func (fs *WebdavFilesystem) Put(ctx context.Context, path string, data []byte) error {
	// path包含了文件名，所以需要提取出路径的文件夹路径,然后进行创建
	dir := filepath.Dir(path)
	if err := fs.client.MkdirAll(dir, 0644); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return fs.client.Write(path, data, 0644)
}

func (fs *WebdavFilesystem) PutWithoutContext(path string, data []byte) error {
	return fs.Put(context.Background(), path, data)
}

func (fs *WebdavFilesystem) Get(path string) ([]byte, error) {
	return fs.client.Read(path)
}

// GetUrl 获取文件完整路径 获取不带用户名密码的URL
func (fs *WebdavFilesystem) GetUrl(path string) string {
	return strings.TrimRight(fs.uri, "/") + "/" + strings.TrimLeft(path, "/")
}

// GetSignedUrl 获取文件完整路径 获取带用户名密码的URL
func (fs *WebdavFilesystem) GetSignedUrl(filePath string, expires int64) (string, error) {
	// 获取当前uri的协议
	u, err := url.Parse(fs.uri)
	if err != nil {
		return "", err
	}

	u.User = url.UserPassword(fs.username, fs.password)
	u.Path = path.Join(u.Path, filePath)

	return u.String(), nil
}

// MustGetSignedUrl 获取签名URL
func (fs *WebdavFilesystem) MustGetSignedUrl(path string, expires int64) string {
	url, _ := fs.GetSignedUrl(path, expires)
	return url
}

func (fs *WebdavFilesystem) GetImageWidthHeight(path string) (int, int, error) {
	data, err := fs.client.Read(path)
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

func (fs *WebdavFilesystem) GetPrivateUrl(path string, expires int64, query any) string {
	url, _ := fs.GetSignedUrl(path, expires)
	return url
}
