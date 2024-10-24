package qiniu_test

import (
	"context"
	"net/http"
	"net/url"
	"os"
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
	// 需要添加新的bucket名称
	key := os.Getenv("QINIU_PRIVATE_TEST_PRIVATE_KEY")
	privateUrl := qnFsPrivate.GetPrivateUrl(key, 30, nil)
	// 测试是否可以访问
	resp, err := http.Get(privateUrl)
	if err != nil {
		t.Error(err)
	}
	defer resp.Body.Close()
	t.Log("private url:", privateUrl)

	// 加query签名地址
	query := url.Values{}
	query.Set("query", "test")
	queryUrl := qnFsPrivate.GetPrivateUrl(key, 30, &query)
	t.Log("query private url:", queryUrl)

	// 测试是否可以访问
	resp, err = http.Get(queryUrl)
	if err != nil {
		t.Error(err)
	}
	defer resp.Body.Close()
	t.Log("query private url:", queryUrl)
}
