package qiniu

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/cdn"
	"github.com/qiniu/go-sdk/v7/storage"
)

type QiniuFilesystem struct {
	AccessKey    string
	AccessSecret string
	Bucket       Bucket

	mac           *auth.Credentials
	bucketManager *storage.BucketManager
}

// Bucket 存储桶
type Bucket struct {
	Name            string // 存储桶名称
	Domain          string // 存储桶域名
	TimestampEncKey string // 时间戳加密key
	Private         bool   // 是否私有
}

// NewStorage 创建七牛云存储
func NewStorage(accessKey, accessSecret string, bucket Bucket) *QiniuFilesystem {
	qnFs := &QiniuFilesystem{
		AccessKey:    accessKey,
		AccessSecret: accessSecret,
		Bucket:       bucket,
	}

	// 初始化七牛云存储
	qnFs.mac = auth.New(qnFs.AccessKey, qnFs.AccessSecret)

	cfg := storage.Config{
		UseHTTPS: true,
	}
	qnFs.bucketManager = storage.NewBucketManager(qnFs.mac, &cfg)

	return qnFs
}

// GetScope 获取存储桶的作用域
func (bucket Bucket) GetScope(saveKey string) string {
	return bucket.Name + ":" + saveKey
}

// GetUrl 获取文件的URL
func (bucket Bucket) GetUrl(path string) string {
	return bucket.Domain + "/" + path
}

// GetSignedUrl 获取签名URL
// path: 文件路径
// expires: 过期时间 单位/秒
func (bucket Bucket) GetSignedUrl(path string, expires int64) (string, error) {
	var restURL string
	if strings.HasPrefix(path, "http") {
		restURL = path
	} else {
		restURL = bucket.GetUrl(path)
	}

	if bucket.TimestampEncKey == "" {
		return restURL, nil
	} else {
		url, err := cdn.CreateTimestampAntileechURL(restURL, bucket.TimestampEncKey, expires)
		if err != nil {
			return "", err
		}
		return url, nil
	}

}

// SimpleUploadToken 生成简单上传凭证
func (qn *QiniuFilesystem) SimpleUploadToken(saveKey string, expires uint64) string {
	return qn.UploadTokenWithPolicy(&storage.PutPolicy{
		Scope:   qn.Bucket.GetScope(saveKey),
		Expires: expires,
	})
}

// UploadTokenWithPolicy 生成上传凭证
func (qn *QiniuFilesystem) UploadTokenWithPolicy(putPolicy *storage.PutPolicy) string {
	return putPolicy.UploadToken(qn.mac)
}

func (qn *QiniuFilesystem) PutWithoutContext(path string, data []byte) error {
	return qn.Put(context.Background(), path, data)
}

func (qn *QiniuFilesystem) Put(ctx context.Context, path string, data []byte) error {
	uploadToken := qn.SimpleUploadToken(path, 180)
	cfg := storage.Config{}
	formUpload := storage.NewFormUploader(&cfg)

	ret := storage.PutRet{}

	putExtra := storage.PutExtra{}

	dataLen := int64(len(data))

	err := formUpload.Put(ctx, &ret, uploadToken, path, bytes.NewReader(data), dataLen, &putExtra)
	if err != nil {
		return fmt.Errorf("upload data failed, %w", err)
	}

	return nil
}

// Get 获取文件
func (qn *QiniuFilesystem) Get(path string) ([]byte, error) {
	resURL, err := qn.Bucket.GetSignedUrl(path, 180)
	if err != nil {
		return nil, fmt.Errorf("fail to get signed url, %w", err)
	}

	httpCli := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := httpCli.Get(resURL)
	if err != nil {
		return nil, fmt.Errorf("fail to get file, %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fail to get file, status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body, %w", err)
	}

	return body, nil
}

// GetUrl 获取文件的URL
func (qn *QiniuFilesystem) GetUrl(path string) string {
	return qn.Bucket.GetUrl(path)
}

// GetSignedUrl 获取签名URL
func (qn *QiniuFilesystem) GetSignedUrl(path string, expires int64) (string, error) {
	if qn.Bucket.Private {
		return qn.getPrivateUrl(path, expires)
	} else {
		return qn.Bucket.GetSignedUrl(path, expires)
	}
}

// MustGetSignedUrl 获取签名URL
func (qn *QiniuFilesystem) MustGetSignedUrl(path string, expires int64) string {
	url, _ := qn.GetSignedUrl(path, expires)
	return url
}

// GetImageWidthHeight 获取图片的宽高
func (qn *QiniuFilesystem) GetImageWidthHeight(path string) (width int, height int, err error) {
	path = path + "?imageInfo"
	url, err := qn.GetSignedUrl(path, 180)
	if err != nil {
		return 0, 0, err
	}

	httpCli := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := httpCli.Get(url)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get image info, %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("failed to get image info, status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read body, %w", err)
	}

	var imageInfoResp struct {
		Size        int64  `json:"size"`
		Format      string `json:"format"`
		Width       int    `json:"width"`
		Height      int    `json:"height"`
		ColorModel  string `json:"colorModel"`
		Orientation int    `json:"orientation"`
	}
	err = json.Unmarshal(body, &imageInfoResp)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to unmarshal image info, %w", err)
	}

	return imageInfoResp.Width, imageInfoResp.Height, nil
}

// getPrivateUrl 获取私有URL
func (qn *QiniuFilesystem) getPrivateUrl(path string, expires int64) (string, error) {
	var privateUrl string
	deadline := time.Now().Add(time.Duration(expires) * time.Second).Unix()
	// 从path中解析query,
	uri, err := url.Parse(path)
	if err != nil {
		return "", errors.New("path is invalid")
	}
	key := strings.TrimLeft(uri.Path, "/")

	var qs string

	// 要在签名前对管道操作符需要进行url编码，否则浏览器会自动编码，导致签名失效
	if uri.RawQuery != "" {
		qs = strings.ReplaceAll(uri.RawQuery, "|", "%7C")
	}

	if qs != "" {
		privateUrl = storage.MakePrivateURLv2WithQueryString(qn.mac, qn.Bucket.Domain, key, qs, deadline)
	} else {
		privateUrl = storage.MakePrivateURLv2(qn.mac, qn.Bucket.Domain, key, deadline)
	}
	return privateUrl, nil
}
