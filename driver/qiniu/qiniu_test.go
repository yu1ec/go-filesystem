package qiniu_test

import (
	"context"
	"testing"

	"github.com/yu1ec/go-filesystem"
	"github.com/yu1ec/go-filesystem/driver/qiniu"
)

var qnFs *qiniu.QiniuFilesystem

func init() {
	accessKey := "VD7rjRLQpyBmrGT1RTx31HKU9_CMCYnZXpSBHHUq"
	accessSecret := "iJXmP76370SjtLv2fEDpvDuhT-Lspuk_1qpnGIOR"
	bucket := qiniu.Bucket{
		Name:            "ifonts-download",
		Domain:          "https://download.ifonts.com",
		TimestampEncKey: "0388ccbbf0abecaacda936890db846abfc8bbc8b",
	}

	qnFs = qiniu.NewStorage(accessKey, accessSecret, bucket)
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
	remoteUrl := "https://download.ifonts.com/test/2024/08/27/819bae42245f4c112308e8f277ccda00.txt"
	data, err := qnFs.Get(remoteUrl)
	if err != nil {
		t.Error(err)
	}
	t.Log("get data:", string(data))
}

func TestQiniuFilesystem_GetSignedUrl(t *testing.T) {
	remoteKey := "test/2024/08/27/819bae42245f4c112308e8f277ccda00.txt"
	signedUrl, err := qnFs.GetSignedUrl(remoteKey, 30)
	if err != nil {
		t.Error(err)
	}
	t.Log("signed url:", signedUrl)
}

func TestQiniuFilesystem_GetImageWidthSize(t *testing.T) {
	remoteKey := "aigc/2024/08/27/9a66bd4771e888cb6a36ea03e5f2c976.png"
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
