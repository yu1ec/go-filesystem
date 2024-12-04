package webdav_test

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/yu1ec/go-filesystem/driver/webdav"
	xwebdav "golang.org/x/net/webdav"
)

// setupTestServer 设置测试服务器
func setupTestServer() (*httptest.Server, *webdav.WebdavFilesystem, func(), error) {
	// 创建一个临时目录作为 WebDAV 根目录
	tempDir, err := os.MkdirTemp("", "webdav-test")

	if err != nil {
		return nil, nil, nil, err
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	handler := &xwebdav.Handler{
		FileSystem: xwebdav.Dir(tempDir),
		LockSystem: xwebdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				fmt.Printf("WebDAV Error: %s\n", err)
			}
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "testuser" || password != "testpass" {
			w.Header().Set("WWW-Authenticate", `Basic realm="WebDAV"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	}))

	fs, err := webdav.NewStorage(server.URL, "testuser", "testpass")
	if err != nil {
		server.Close()
		cleanup()
		return nil, nil, nil, err
	}

	return server, fs, cleanup, nil
}

func TestWebdavFilesystem(t *testing.T) {
	server, fs, cleanup, err := setupTestServer()
	if err != nil {
		t.Fatalf("Failed to setup test server: %v", err)
	}
	defer server.Close()
	defer cleanup()

	t.Run("Put and Get", func(t *testing.T) {
		data := []byte("test data")
		err := fs.Put(context.Background(), "/test.txt", data)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		retrieved, err := fs.Get("/test.txt")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if !bytes.Equal(data, retrieved) {
			t.Errorf("Retrieved data doesn't match. Expected %s, got %s", data, retrieved)
		}
	})

	t.Run("PutWithoutContext", func(t *testing.T) {
		data := []byte("test data without context")
		err := fs.PutWithoutContext("/test_no_context.txt", data)
		if err != nil {
			t.Fatalf("PutWithoutContext failed: %v", err)
		}

		retrieved, err := fs.Get("/test_no_context.txt")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if !bytes.Equal(data, retrieved) {
			t.Errorf("Retrieved data doesn't match. Expected %s, got %s", data, retrieved)
		}
	})

	t.Run("GetUrl", func(t *testing.T) {
		url := fs.GetUrl("/test.txt")
		expected := server.URL + "/test.txt"
		if url != expected {
			t.Errorf("GetUrl returned incorrect URL. Expected %s, got %s", expected, url)
		}
	})

	t.Run("GetSignedUrl", func(t *testing.T) {
		url, err := fs.GetSignedUrl("/test.txt", 3600)
		if err != nil {
			t.Fatalf("GetSignedUrl failed: %v", err)
		}
		expected := "http://testuser:testpass@" + server.URL[7:] + "/test.txt"
		if url != expected {
			t.Errorf("GetSignedUrl returned incorrect URL. Expected %s, got %s", expected, url)
		}
	})

	t.Run("MustGetSignedUrl", func(t *testing.T) {
		url := fs.MustGetSignedUrl("/test.txt", 3600)
		expected := "http://testuser:testpass@" + server.URL[7:] + "/test.txt"
		if url != expected {
			t.Errorf("MustGetSignedUrl returned incorrect URL. Expected %s, got %s", expected, url)
		}
	})

	t.Run("GetImageWidthHeight", func(t *testing.T) {
		// 创建一个2x3像素的测试图像
		img := image.NewRGBA(image.Rect(0, 0, 2, 3))
		var buf bytes.Buffer
		png.Encode(&buf, img)
		imgData := buf.Bytes()

		err := fs.Put(context.Background(), "/test_image.png", imgData)
		if err != nil {
			t.Fatalf("Failed to put test image: %v", err)
		}

		width, height, err := fs.GetImageWidthHeight("/test_image.png")
		if err != nil {
			t.Fatalf("GetImageWidthHeight failed: %v", err)
		}

		if width != 2 || height != 3 {
			t.Errorf("Incorrect image dimensions. Expected 2x3, got %dx%d", width, height)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		// 先创建一个测试文件
		testData := []byte("test file for deletion")
		testPath := "/test_delete.txt"

		err := fs.Put(context.Background(), testPath, testData)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// 测试删除文件
		err = fs.Delete(testPath)
		if err != nil {
			t.Errorf("Delete failed: %v", err)
		}

		// 验证文件已被删除
		_, err = fs.Get(testPath)
		if err == nil {
			t.Error("Expected error when getting deleted file, got nil")
		}
	})
}
