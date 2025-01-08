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

	mac              *auth.Credentials
	bucketManager    *storage.BucketManager
	operationManager *storage.OperationManager
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
	qnFs.operationManager = storage.NewOperationManager(qnFs.mac, &cfg)
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

// GetAntileechSignedUrl 获取防盗链签名URL
// path: 文件路径
// expires: 过期时间 单位/秒
func (bucket Bucket) GetAntileechSignedUrl(path string, expires int64) (string, error) {
	var restURL string
	if strings.HasPrefix(path, "http") {
		restURL = path
	} else {
		restURL = bucket.GetUrl(path)
		parsedUrl, err := url.Parse(restURL)
		if err != nil {
			return "", err
		}

		restURL = parsedUrl.Scheme + "://" + parsedUrl.Host + parsedUrl.Path

		qs := removeQuerySignParams(parsedUrl.RawQuery)
		if qs != "" {
			restURL += "?" + qs
		}
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

// GetBucketManager 获取BucketManager
func (qn *QiniuFilesystem) GetBucketManager() *storage.BucketManager {
	return qn.bucketManager
}

// NewCensor 创建审查器
func (qn *QiniuFilesystem) NewCensor() *Censor {
	return NewCensor(qn)
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
	resURL, err := qn.GetSignedUrl(path, 180)
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
		return qn.Bucket.GetAntileechSignedUrl(path, expires)
	}
}

// MustGetSignedUrl 获取签名URL
func (qn *QiniuFilesystem) MustGetSignedUrl(path string, expires int64) string {
	url, err := qn.GetSignedUrl(path, expires)
	if err != nil {
		panic(err)
	}
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
		Width  int `json:"width"`
		Height int `json:"height"`
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

	qs := removeQuerySignParams(uri.RawQuery)

	if qs != "" {
		privateUrl = storage.MakePrivateURLv2WithQueryString(qn.mac, qn.Bucket.Domain, key, qs, deadline)
	} else {
		privateUrl = storage.MakePrivateURLv2(qn.mac, qn.Bucket.Domain, key, deadline)
	}
	return privateUrl, nil
}

// Delete 删除文件
func (qn *QiniuFilesystem) Delete(path string) error {
	return qn.bucketManager.Delete(qn.Bucket.Name, path)
}

// Exists 判断文件是否存在
func (qn *QiniuFilesystem) Exists(path string) bool {
	signedUrl := qn.MustGetSignedUrl(path, 180)

	// 只请求头信息，判断文件是否存在
	resp, err := http.Head(signedUrl)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode != http.StatusNotFound
}

type ZipOptions struct {
	SaveAs    *SaveAs
	Pipeline  string
	NotifyURL string
	Force     bool
	IsWait    bool
}

// Zip 打包资源
// mkzipArgs: 打包参数
// saveAs: 保存参数
func (qn *QiniuFilesystem) Zip(mkzipArgs *MkZipArgs, opts *ZipOptions) (string, error) {
	mkzipArgsStr, err := mkzipArgs.ToString()
	if err != nil {
		return "", fmt.Errorf("failed to get fop string, %w", err)
	}

	bucket := qn.Bucket.Name

	// 此处的key是打包索引文件的key
	key := mkzipArgs.GetIndexFileKey()
	pipeline := ""
	notifyURL := ""
	force := true
	fops := mkzipArgsStr

	if !qn.Exists(key) {
		indexContents := []byte(mkzipArgs.GetUrlsStr())

		// 如果打包索引文件不存在，则先上传
		err := qn.Put(context.Background(), key, indexContents)
		if err != nil {
			return "", fmt.Errorf("failed to put index file, %w", err)
		}
	}

	// 延迟删除打包索引文件
	defer func() {
		_ = qn.Delete(key)
	}()

	if opts != nil {
		if opts.Pipeline != "" {
			pipeline = opts.Pipeline
		}
		if opts.NotifyURL != "" {
			notifyURL = opts.NotifyURL
		}
		if opts.Force {
			force = opts.Force
		}

		if opts.SaveAs != nil {
			saveAsStr, err := opts.SaveAs.ToString()
			if err != nil {
				return "", fmt.Errorf("failed to get save key, %w", err)
			}
			fops += "|" + saveAsStr
		}
	}

	// client.DebugMode = true
	// client.DeepDebugInfo = true
	persistentID, err := qn.operationManager.Pfop(
		bucket,
		key,
		fops,
		pipeline,
		notifyURL,
		force,
	)

	if err != nil {
		return "", fmt.Errorf("failed to pfop, %w", err)
	}

	// 等待完成
	if opts.IsWait {
		for {
			// 等待500ms
			time.Sleep(500 * time.Millisecond)
			ret, err := qn.Prefop(persistentID)
			if err != nil {
				return "", fmt.Errorf("failed to prefop, %w", err)
			}

			if ret.ID != persistentID {
				return "", fmt.Errorf("persistentID not match, %s != %s", ret.ID, persistentID)
			}

			if ret.Code == 3 {
				return "", fmt.Errorf("failed to zip, %s", ret.Desc)
			}

			if ret.Code == 0 {
				return ret.ID, nil
			}
		}
	}

	return persistentID, nil
}

// Prefop 查询任务状态
func (qn *QiniuFilesystem) Prefop(persistentID string) (storage.PrefopRet, error) {
	ret, err := qn.operationManager.Prefop(persistentID)
	return ret, err
}

// removeQuerySignParams 移除查询参数中的签名参数
func removeQuerySignParams(qs string) string {
	if qs == "" {
		return ""
	}

	parts := strings.Split(qs, "&")
	var result []string
	needRemoveSignParams := []string{"sign=", "t=", "e=", "token="}

	for _, part := range parts {
		skip := false
		for _, param := range needRemoveSignParams {
			if strings.HasPrefix(part, param) {
				skip = true
				break
			}
		}
		if !skip {
			result = append(result, part)
		}
	}
	return strings.Join(result, "&")
}
