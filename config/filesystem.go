package config

// 文件系统驱动
type FilesystemDriver struct {
	Name   string `yaml:"name"`
	Config any    `yaml:"config"`
}

// 本地文件系统
type LocalDriverConfig struct {
	Root string `yaml:"root,omitempty"`
}

// 七牛云文件系统
type QiniuDriverConfig struct {
	AccessKey       string `yaml:"access_key"`
	AccessSecret    string `yaml:"access_secret"`
	Bucket          string `yaml:"bucket"`
	Domain          string `yaml:"domain"`
	TimestampEncKey string `yaml:"timestamp_enc_key,omitempty"`
}

// Webdav文件系统
type WebdavDriverConfig struct {
	Uri      string `yaml:"uri"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}
