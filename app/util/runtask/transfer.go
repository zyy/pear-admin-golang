package runtask

import (
	"fmt"
	"github.com/cilidm/toolbox/file"
	"github.com/pkg/sftp"
	"go.uber.org/zap"
	"os"
	"path"
	"pear-admin-go/app/core/log"
)

func continuation(dstFile string, srcFile *sftp.File, has os.FileInfo) error {
	lf, err := os.OpenFile(dstFile, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}
	defer lf.Close()
	_, _ = srcFile.Seek(has.Size()+1, 0)
	if _, err = srcFile.WriteTo(lf); err != nil {
		return err
	}
	log.Instance().Info(fmt.Sprintf("【%s】传输完毕", srcFile.Name()))
	return nil
}

func create(dstFile string, srcFile *sftp.File) error {
	dir, _ := path.Split(dstFile)
	err := file.IsNotExistMkDir(dir)
	if err != nil {
		log.Instance().Error("RemoteToLocal.IsNotExistMkDir", zap.Error(err))
		return err
	}

	lf, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer lf.Close()

	if _, err = srcFile.WriteTo(lf); err != nil {
		return err
	}
	log.Instance().Info(fmt.Sprintf("【%s】传输完毕", srcFile.Name()))
	return nil
}
