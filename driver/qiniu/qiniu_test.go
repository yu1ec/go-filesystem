package qiniu_test

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/yu1ec/go-filesystem"
	"github.com/yu1ec/go-filesystem/driver/qiniu"
)

var qnFs *qiniu.QiniuFilesystem
var qnFsPrivate *qiniu.QiniuFilesystem

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}

	accessKey := os.Getenv("QINIU_ACCESS_KEY")
	accessSecret := os.Getenv("QINIU_SECRET_KEY")
	secureBucket := qiniu.Bucket{
		Name:            os.Getenv("QINIU_SECURE_BUCKET_NAME"),
		Domain:          os.Getenv("QINIU_SECURE_BUCKET_DOMAIN"),
		TimestampEncKey: os.Getenv("QINIU_TIMESTAMP_ENC_KEY"),
	}

	qnFs = qiniu.NewStorage(accessKey, accessSecret, secureBucket)

	privateBucket := qiniu.Bucket{
		Name:   os.Getenv("QINIU_PRIVATE_BUCKET_NAME"),
		Domain: os.Getenv("QINIU_PRIVATE_BUCKET_DOMAIN"),
	}
	qnFsPrivate = qiniu.NewStorage(accessKey, accessSecret, privateBucket)
}

func TestQiniuFilesystem_SimpleUploadToken(t *testing.T) {
	token := qnFs.SimpleUploadToken("test", 3600)
	t.Log("upload token:", token)
}

func TestQiniuFilesystem_Put(t *testing.T) {
	uploadData := []byte("测试文件哈哈哈哈哈哈哈哈哈哈哈哈")
	remoteKey := filesystem.BuildUploadKey("test", "txt")
	err := qnFs.Put(context.Background(), remoteKey, []byte(uploadData))
	if err != nil {
		t.Error(err)
	}

	t.Log("upload success, remote url:", qnFs.Bucket.Domain+"/"+remoteKey)
}

func TestQiniuFilesystem_Get(t *testing.T) {
	remoteUrl := os.Getenv("QINIU_SECURE_TEST_REMOTE_KEY")
	data, err := qnFs.Get(remoteUrl)
	if err != nil {
		t.Error(err)
	}
	t.Log("get data:", string(data))
}

func TestQiniuFilesystem_GetSignedUrl(t *testing.T) {
	remoteKey := os.Getenv("QINIU_SECURE_TEST_REMOTE_KEY")
	signedUrl, err := qnFs.GetSignedUrl(remoteKey, 30)
	if err != nil {
		t.Error(err)
	}
	t.Log("signed url:", signedUrl)
}

func TestQiniuFilesystem_GetImageWidthSize(t *testing.T) {
	remoteKey := os.Getenv("QINIU_SECURE_TEST_IMAGE_KEY")
	width, height, err := qnFs.GetImageWidthHeight(remoteKey)
	if err != nil {
		t.Error(err)
	}

	if width != 512 || height != 512 {
		t.Error("image size error")
	}

	t.Log("image width:", width, "height:", height)
	//Output: image width: 512 height: 512
}

func TestQiniuFilesystem_GetPrivateUrl(t *testing.T) {
	// 初始化测试环境
	err := godotenv.Load()
	if err != nil {
		t.Logf("Error loading .env file: %v", err)
	}

	key := os.Getenv("QINIU_PRIVATE_TEST_PRIVATE_KEY")
	if key == "" {
		t.Fatal("QINIU_PRIVATE_TEST_PRIVATE_KEY environment variable is not set")
	}

	qnFs := qnFsPrivate
	bucket := qnFsPrivate.Bucket

	testCases := []struct {
		name    string
		query   any
		checker func(t *testing.T, rawUrl string)
	}{
		{
			name:  "无查询参数",
			query: nil,
			checker: func(t *testing.T, rawUrl string) {
				decodedUrl, err := url.QueryUnescape(rawUrl)
				if err != nil {
					t.Errorf("URL解码失败: %v", err)
					return
				}

				pattern := "^" + regexp.QuoteMeta(bucket.Domain) + "/" +
					regexp.QuoteMeta(key) +
					"\\?e=\\d+&token=[^:]+:[^&]+$"
				matched, _ := regexp.MatchString(pattern, decodedUrl)
				if !matched {
					t.Errorf("URL格式不匹配\n期望格式: %s\n实际URL: %s", pattern, decodedUrl)
				}
			},
		},
		{
			name:  "字符串查询参数",
			query: "foo=bar&baz=qux",
			checker: func(t *testing.T, rawUrl string) {
				decodedUrl, err := url.QueryUnescape(rawUrl)
				if err != nil {
					t.Errorf("URL解码失败: %v", err)
					return
				}

				// 检查是否包含必要的参数
				if !strings.Contains(decodedUrl, "foo=bar") || !strings.Contains(decodedUrl, "baz=qux") {
					t.Error("URL缺少必要的查询参数")
				}
				// 检查基本格式
				if !strings.Contains(decodedUrl, "e=") || !strings.Contains(decodedUrl, "token=") {
					t.Error("URL缺少e或token参数")
				}
			},
		},
		{
			name: "url.Values查询参数",
			query: func() url.Values {
				v := url.Values{}
				v.Set("foo", "bar")
				v.Set("baz", "qux")
				return v
			}(),
			checker: func(t *testing.T, rawUrl string) {
				decodedUrl, err := url.QueryUnescape(rawUrl)
				if err != nil {
					t.Errorf("URL解码失败: %v", err)
					return
				}

				// 检查是否包含必要的参数
				if !strings.Contains(decodedUrl, "foo=bar") || !strings.Contains(decodedUrl, "baz=qux") {
					t.Error("URL缺少必要的查询参数")
				}
				// 检查基本格式
				if !strings.Contains(decodedUrl, "e=") || !strings.Contains(decodedUrl, "token=") {
					t.Error("URL缺少e或token参数")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			privateUrl := qnFs.GetPrivateUrl(key, 30, tc.query)
			t.Logf("生成的私有URL: %s", privateUrl)

			// 使用自定义检查器验证URL
			tc.checker(t, privateUrl)

			// 验证URL可访问性
			resp, err := http.Get(privateUrl)
			if err != nil {
				t.Logf("警告：无法访问生成的URL: %v", err)
			} else {
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					t.Logf("警告：URL返回非200状态码: %d", resp.StatusCode)
				} else {
					t.Logf("成功访问URL，状态码: %d", resp.StatusCode)
				}
			}
		})
	}
}
