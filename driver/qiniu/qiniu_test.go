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
		Name:    os.Getenv("QINIU_PRIVATE_BUCKET_NAME"),
		Domain:  os.Getenv("QINIU_PRIVATE_BUCKET_DOMAIN"),
		Private: true,
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

func TestQiniuFilesystem_GetSignedUrl1(t *testing.T) {
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

func TestQiniuFilesystem_GetSignedUrl(t *testing.T) {
	testCases := []struct {
		name     string
		fs       *qiniu.QiniuFilesystem
		key      string
		query    string
		checkUrl func(t *testing.T, url string)
	}{
		{
			name: "安全时间戳签名",
			fs:   qnFs,
			key:  os.Getenv("QINIU_SECURE_TEST_REMOTE_KEY"),
			checkUrl: func(t *testing.T, url string) {
				// 检查是否包含时间戳签名参数
				if !strings.Contains(url, "sign=") || !strings.Contains(url, "t=") {
					t.Error("URL缺少时间戳签名参数")
				}
			},
		},
		{
			name: "私有空间签名-无参数",
			fs:   qnFsPrivate,
			key:  os.Getenv("QINIU_PRIVATE_TEST_REMOTE_KEY"),
			checkUrl: func(t *testing.T, rawUrl string) {
				decodedUrl, err := url.QueryUnescape(rawUrl)
				if err != nil {
					t.Errorf("URL解码失败: %v", err)
					return
				}
				// 解码
				pattern := "^" + regexp.QuoteMeta(qnFsPrivate.Bucket.Domain) + "/" +
					"[^?]+\\?e=\\d+&token=[^:]+:[^&]+$"
				matched, _ := regexp.MatchString(pattern, decodedUrl)
				if !matched {
					t.Errorf("私有空间URL格式不匹配\n期望格式: %s\n实际URL: %s", pattern, decodedUrl)
				}
			},
		},
		{
			name:  "私有空间签名-带查询参数",
			fs:    qnFsPrivate,
			key:   os.Getenv("QINIU_PRIVATE_TEST_REMOTE_KEY"),
			query: "?imageInfo",
			checkUrl: func(t *testing.T, rawUrl string) {
				decodedUrl, err := url.QueryUnescape(rawUrl)
				if err != nil {
					t.Errorf("URL解码失败: %v", err)
					return
				}
				// 检查基本格式
				if !strings.Contains(decodedUrl, "imageInfo") {
					t.Error("URL缺少imageInfo参数")
				}
				if !strings.Contains(decodedUrl, "e=") || !strings.Contains(decodedUrl, "token=") {
					t.Error("URL缺少签名参数")
				}

				// 验证token格式
				tokenParts := strings.Split(decodedUrl, "token=")
				if len(tokenParts) != 2 {
					t.Error("URL token格式错误")
					return
				}
				token := tokenParts[1]
				if !strings.Contains(token, ":") {
					t.Error("token格式错误，缺少':'分隔符")
				}
			},
		},
		{
			name: "私有空间-完整url签名",
			fs:   qnFsPrivate,
			key:  os.Getenv("QINIU_PRIVATE_TEST_REMOTE_URL"),
			checkUrl: func(t *testing.T, rawUrl string) {
				decodedUrl, err := url.QueryUnescape(rawUrl)
				if err != nil {
					t.Errorf("URL解码失败: %v", err)
					return
				}
				if !strings.Contains(decodedUrl, "e=") || !strings.Contains(decodedUrl, "token=") {
					t.Error("URL缺少签名参数")
				}

				// 验证token格式
				tokenParts := strings.Split(decodedUrl, "token=")
				if len(tokenParts) != 2 {
					t.Error("URL token格式错误")
					return
				}
				token := tokenParts[1]
				if !strings.Contains(token, ":") {
					t.Error("token格式错误，缺少':'分隔符")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := tc.key
			if tc.query != "" {
				key = key + tc.query
			}

			signedUrl, err := tc.fs.GetSignedUrl(key, 30)
			if err != nil {
				t.Fatalf("获取签名URL失败: %v", err)
			}

			t.Logf("签名URL: %s", signedUrl)
			tc.checkUrl(t, signedUrl)

			// 验证URL可访问性
			resp, err := http.Get(signedUrl)
			if err != nil {
				t.Logf("警告：无法访问生成的URL: %v", err)
			} else {
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					t.Errorf("URL访问失败，状态码: %d", resp.StatusCode)
				} else {
					t.Logf("URL访问成功，状态码: %d", resp.StatusCode)
				}
			}
		})
	}
}
