package runtask

import (
	"crypto/md5"
	"fmt"
	"github.com/cilidm/toolbox/OS"
	"io"
	"path"
	"strings"
)

// 工具

func GetMd(reader io.Reader) (string, error) {
	md5hash := md5.New()
	if _, err := io.Copy(md5hash, reader); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", md5hash.Sum(nil)), nil
}

func pathJoin(p string) (np string) {
	if strings.HasSuffix(p, "/") == false {
		p = p + "/"
	}
	if OS.IsWindows() {
		np = strings.ReplaceAll(path.Join(p, "*"), "\\", "/")
	} else {
		np = path.Join(p, "*")
	}
	return
}
