package qiniu_test

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
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

func TestQiniuFilesystem_GetPrivateImageWidthSize(t *testing.T) {
	remoteKey := os.Getenv("QINIU_PRIVATE_TEST_REMOTE_KEY")
	width, height, err := qnFsPrivate.GetImageWidthHeight(remoteKey)
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
		{
			name: "私有空间-带完整水印参数的完整url签名",
			fs:   qnFsPrivate,
			key:  os.Getenv("QINIU_PRIVATE_TEST_REMOTE_URL") + "?imageslim|imageMogr2/auto-orient/thumbnail/500x/blur/1x0/quality/85/interlace/1/ignore-error/1|watermark/4/text/Qm9vbUFp/fontsize/480/rotate/45/uw/255/uh/255/dissolve/20/fill/IzgwODA4MA==",
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

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Errorf("读取响应体失败: %v", err)
				}
				t.Logf("响应体: %s", string(body))
			}
		})
	}
}

func TestQiniuFilesystem_Delete(t *testing.T) {
	// 先上传一个文件用于测试删除
	uploadData := []byte("test file for deletion")
	remoteKey := filesystem.BuildUploadKey("test_delete", "txt")
	err := qnFs.Put(context.Background(), remoteKey, uploadData)
	if err != nil {
		t.Fatalf("Failed to upload test file: %v", err)
	}

	// 测试删除文件
	err = qnFs.Delete(remoteKey)
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	// 验证文件已被删除
	_, err = qnFs.Get(remoteKey)
	if err == nil {
		t.Error("Expected error when getting deleted file, got nil")
	}
}

func TestCensor_CheckImageByURI(t *testing.T) {
	tests := []struct {
		fs   filesystem.Filesystem
		name string

		uri      string
		wantPass bool
		wantErr  bool
	}{
		{
			fs:       qnFsPrivate,
			name:     "bad qiniu image",
			uri:      "qiniu:///90sheji-download/aimodel_test/2024/12/05/1a9e0539c9b6ae2a7f8969d2eb6948bc.jpeg",
			wantPass: false,
			wantErr:  true,
		},
		{
			fs:       qnFsPrivate,
			name:     "bad url image",
			uri:      qnFsPrivate.MustGetSignedUrl("aimodel_test/2024/12/05/1a9e0539c9b6ae2a7f8969d2eb6948bc.jpeg", 1800),
			wantPass: false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qnFs := filesystem.MustAsQiniu(tt.fs)
			// Test image check
			suggestion, reasons, err := qnFs.NewCensor().CheckImageByURI(tt.uri)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantPass, suggestion == qiniu.SuggestionPass, "CheckImageByURI result mismatch"+strings.Join(reasons, ";"))
			}
		})
	}
}

func TestCensor_CheckImageData(t *testing.T) {
	data, err := qnFsPrivate.Get("aimodel_test/2024/12/05/1a9e0539c9b6ae2a7f8969d2eb6948bc.jpeg")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		fs   filesystem.Filesystem
		name string

		data     []byte
		wantPass bool
		wantErr  bool
	}{
		{
			fs:       qnFsPrivate,
			name:     "bad qiniu image",
			data:     data,
			wantPass: false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qnFs := filesystem.MustAsQiniu(tt.fs)
			// Test image check
			suggestion, reasons, err := qnFs.NewCensor().CheckImageData(tt.data)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantPass, suggestion == qiniu.SuggestionPass, "CheckImageData result mismatch"+strings.Join(reasons, ";"))
			}
		})
	}
}

func TestQiniuFilesystem_Exists(t *testing.T) {
	testData := []byte("test")
	// 使用时间戳和随机数组合生成唯一文件名
	randNum := rand.New(rand.NewSource(time.Now().UnixNano())).Int63()
	testFile := fmt.Sprintf("test_exists_%d_%d.txt", time.Now().UnixNano(), randNum)
	err := qnFsPrivate.Put(context.Background(), testFile, testData)
	if err != nil {
		t.Fatalf("Failed to upload test file: %v", err)
	}

	if !qnFsPrivate.Exists(testFile) {
		t.Error("Expected file to exist, but it doesn't")
	}

	err = qnFsPrivate.Delete(testFile)
	if err != nil {
		t.Fatalf("Failed to delete test file: %v", err)
	}

	if qnFsPrivate.Exists(testFile) {
		t.Error("Expected file to not exist, but it does")
	}
}
