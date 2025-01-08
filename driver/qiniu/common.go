package qiniu

import (
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/qiniu/go-sdk/v7/storage"
)

type SaveAs struct {
	SaveBucket      string // 文件存储bucket
	SaveKey         string // 文件存储key
	DeleteAfterDays *int   // 文件存储过期时间 单位/天 最小1天，不设置代表不会自动删除
}

// 将SaveAs转换为fop字符串
func (s *SaveAs) ToString() (string, error) {
	if s.SaveBucket == "" {
		return "", errors.New("SaveBucket is required")
	}

	outStr := "saveas/"

	if s.SaveKey != "" {
		outStr += storage.EncodedEntry(s.SaveBucket, s.SaveKey)
	} else {
		outStr += storage.EncodedEntryWithoutKey(s.SaveBucket)
	}

	if s.DeleteAfterDays != nil && *s.DeleteAfterDays > 0 {
		outStr += fmt.Sprintf("/deleteAfterDays/%d", *s.DeleteAfterDays)
	}

	return outStr, nil
}

// 压缩文件参数
type MkZipArgs struct {
	Encoding     string            // 编码方式 默认: utf-8
	IndexFileKey string            // 打包索引文件的key
	URLsMap      map[string]string // 需要压缩的文件路径列表 格式: {url: 文件地址(必须公网可访问)，例如: http://example.com/file.txt, alias: 别名}
}

// 获取压缩模式 2: 用于少量文件压缩 4: 用于大量文件压缩
func (m *MkZipArgs) GetMode() int {
	urlsStr := m.GetUrlsStr()
	if len(urlsStr) <= 2000 {
		return 2
	}
	return 4
}

// 获取打包索引文件的key
func (m *MkZipArgs) GetIndexFileKey() string {
	if m == nil || m.IndexFileKey == "" {
		return fmt.Sprintf("mkzip-%d.txt", time.Now().UnixNano())
	}
	return m.IndexFileKey
}

// 获取urlsStr
func (m *MkZipArgs) GetUrlsStr() string {
	if m == nil || m.URLsMap == nil {
		return ""
	}

	urlsStr := ""
	for url, alias := range m.URLsMap {
		urlsStr += "/url/" + base64.URLEncoding.EncodeToString([]byte(url))
		if alias != "" {
			urlsStr += "/alias/" + base64.URLEncoding.EncodeToString([]byte(alias))
		}
	}
	return urlsStr
}

// 将MkZipArgs转换为fop字符串
func (m *MkZipArgs) ToString() (string, error) {
	mode := m.GetMode()

	outStr := fmt.Sprintf("mkzip/%d", mode)

	if m.Encoding != "" {
		outStr += "/encoding/" + base64.URLEncoding.EncodeToString([]byte(m.Encoding))
	}

	if mode == 2 {
		urlsStr := m.GetUrlsStr()
		outStr += urlsStr
	}

	return outStr, nil
}
