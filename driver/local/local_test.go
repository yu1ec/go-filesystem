package local_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/yu1ec/go-filesystem/driver/local"
)

func TestLocalFilesystem(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "local_test")
	if err != nil {
		t.Fatalf("无法创建临时目录：%v", err)
	}
	defer os.RemoveAll(tempDir)

	fs := local.NewStorage(tempDir)

	t.Run("Put和Get", func(t *testing.T) {
		path := filepath.Join(tempDir, "test.txt")
		data := []byte("测试数据")

		err := fs.Put(context.Background(), path, data)
		if err != nil {
			t.Fatalf("Put失败：%v", err)
		}

		retrieved, err := fs.Get(path)
		if err != nil {
			t.Fatalf("Get失败：%v", err)
		}

		if string(retrieved) != string(data) {
			t.Errorf("获取的数据不匹配。期望：%s，实际：%s", string(data), string(retrieved))
		}
	})

	t.Run("PutWithoutContext", func(t *testing.T) {
		path := filepath.Join(tempDir, "test_no_context.txt")
		data := []byte("无上下文测试数据")

		err := fs.PutWithoutContext(path, data)
		if err != nil {
			t.Fatalf("PutWithoutContext失败：%v", err)
		}

		retrieved, err := fs.Get(path)
		if err != nil {
			t.Fatalf("Get失败：%v", err)
		}

		if string(retrieved) != string(data) {
			t.Errorf("获取的数据不匹配。期望：%s，实际：%s", string(data), string(retrieved))
		}
	})

	t.Run("GetUrl", func(t *testing.T) {
		// 创建测试文件
		relativeFile := "test_url.txt"
		absoluteFile := filepath.Join(tempDir, "absolute_test.txt")

		for _, file := range []string{relativeFile, absoluteFile} {
			err := os.WriteFile(filepath.Join(tempDir, filepath.Base(file)), []byte("test content"), 0644)
			if err != nil {
				t.Fatalf("无法创建测试文件 %s: %v", file, err)
			}
		}

		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "相对路径",
				input:    relativeFile,
				expected: filepath.Join(tempDir, relativeFile),
			},
			{
				name:     "绝对路径",
				input:    absoluteFile,
				expected: absoluteFile,
			},
			{
				name:     "空Root的绝对路径",
				input:    absoluteFile,
				expected: absoluteFile,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var fs *local.LocalFilesystem
				if tt.name == "空Root的绝对路径" {
					fs = local.NewStorage("")
				} else {
					fs = local.NewStorage(tempDir)
				}

				url := fs.GetUrl(tt.input)

				expected, err := filepath.Abs(tt.expected)
				if err != nil {
					t.Fatalf("无法获取绝对路径: %v", err)
				}

				if url != expected {
					t.Errorf("GetUrl返回的URL不正确。\n期望：%s\n实际：%s", expected, url)
				}

				// 验证返回的路径是否指向正确的文件
				if _, err := os.Stat(url); os.IsNotExist(err) {
					t.Errorf("GetUrl返回的路径不存在：%s", url)
				}
			})
		}
	})

	t.Run("GetSignedUrl和MustGetSignedUrl", func(t *testing.T) {
		path := "test_signed.txt"
		expires := int64(3600)

		signedUrl, err := fs.GetSignedUrl(path, expires)
		if err != nil {
			t.Fatalf("GetSignedUrl失败：%v", err)
		}

		mustSignedUrl := fs.MustGetSignedUrl(path, expires)

		if signedUrl != mustSignedUrl {
			t.Errorf("GetSignedUrl和MustGetSignedUrl返回的URL不一致")
		}
	})

	t.Run("GetImageWidthHeight", func(t *testing.T) {
		// 创建一个2x3像素的图像
		img := image.NewRGBA(image.Rect(0, 0, 2, 3))
		img.Set(0, 0, color.RGBA{0, 0, 0, 255})
		img.Set(1, 0, color.RGBA{255, 255, 255, 255})
		img.Set(0, 1, color.RGBA{255, 0, 0, 255})
		img.Set(1, 1, color.RGBA{0, 255, 0, 255})
		img.Set(0, 2, color.RGBA{0, 0, 255, 255})
		img.Set(1, 2, color.RGBA{255, 255, 0, 255})

		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			t.Fatalf("无法编码PNG图像：%v", err)
		}

		imgData := buf.Bytes()
		imgPath := filepath.Join(tempDir, "test_image.png")

		err := fs.Put(context.Background(), imgPath, imgData)
		if err != nil {
			t.Fatalf("无法保存测试图像：%v", err)
		}

		width, height, err := fs.GetImageWidthHeight(imgPath)
		if err != nil {
			t.Fatalf("GetImageWidthHeight失败：%v", err)
		}

		if width != 2 || height != 3 {
			t.Errorf("图像尺寸不正确。期望：2x3，实际：%dx%d", width, height)
		}
	})
}
