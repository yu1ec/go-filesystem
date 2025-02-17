package config

// 文件系统驱动
type FilesystemDriver struct {
	Name   string `yaml:"name"`
	Config any    `yaml:"config"`
}

// 本地文件系统
type LocalDriverConfig struct {
	Root    string `yaml:"root,omitempty"`     // 文件存储根目录 设置后，文件会被限制到此目录下
	BaseUrl string `yaml:"base_url,omitempty"` // 基础URL, 用于生成完整URL
}

// 七牛云文件系统
type QiniuDriverConfig struct {
	AccessKey       string `yaml:"access_key"`
	AccessSecret    string `yaml:"access_secret"`
	Bucket          string `yaml:"bucket"`
	Domain          string `yaml:"domain"`
	TimestampEncKey string `yaml:"timestamp_enc_key,omitempty"`
	Private         bool   `yaml:"private,omitempty"`
}

// Webdav文件系统
type WebdavDriverConfig struct {
	Uri      string `yaml:"uri"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}
