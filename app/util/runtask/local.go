package runtask

import (
	"fmt"
	"github.com/cilidm/toolbox/OS"
	"go.uber.org/zap"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"pear-admin-go/app/core/log"
	"pear-admin-go/app/dao"
	"pear-admin-go/app/model"
	"pear-admin-go/app/util/check"
	"strings"
	"sync/atomic"
	"time"
)

// RunL2R 本地->远端
func (this *RunTask) RunL2R() {
	_ = filepath.Walk(this.task.SourcePath, func(v string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Instance().Error("RunL2R.Walk.err", zap.Error(err))
			return nil
		}
		if info == nil {
			log.Instance().Error("RunL2R.Walk.info Is Nil")
			return nil
		}
		stat, err := os.Stat(v)
		if err != nil {
			log.Instance().Error("RunL2R.os.Stat", zap.Error(err))
			return nil
		}
		if stat.IsDir() {
			if v == this.task.SourcePath {
				return nil
			}
			dname := string([]rune(strings.ReplaceAll(v, this.task.SourcePath, ""))[1:])
			err = this.MkRemotedir(dname)
			if err != nil {
				log.Instance().Error("RunL2R.rm.Mkdir", zap.Error(err))
				return nil
			}
		} else {
			this.fp.Add(1)
			atomic.AddUint64(&this.counter, 1)
			go func(v string, size int64) {
				defer this.fp.Done()
				err = this.LocalSend(v, size)
				if err != nil {
					log.Instance().Error("WalkPath.LocalToRemote", zap.Error(err))
				}
			}(v, stat.Size())
		}
		return nil
	})
}

func (this *RunTask) LocalSend(fname string, fsize int64) error {
	if OS.IsWindows() {
		this.task.SourcePath = strings.ReplaceAll(this.task.SourcePath, "\\", "/")
		this.task.DstPath = strings.ReplaceAll(this.task.DstPath, "\\", "/")
		fname = strings.ReplaceAll(fname, "\\", "/")
		fname = strings.ReplaceAll(fname, this.task.SourcePath, "")
	}
	rf := path.Join(this.task.DstPath, fname) // 文件在服务器的路径及名称
	has, err := this.dstClient.Stat(rf)
	if err == nil && (has.Size() == fsize) {
		log.Instance().Debug(fmt.Sprintf("文件%s已存在", rf))
		return nil
	}
	err = this.dstClient.MkdirAll(this.task.DstPath)
	if err != nil {
		return err
	}
	err = this.dstClient.Chmod(this.task.DstPath, os.ModePerm)
	if err != nil {
		return err
	}
	srcFile, err := os.Open(path.Join(this.task.SourcePath, fname))
	if err != nil {
		log.Instance().Error("源文件无法读取", zap.Error(err))
		return err
	}
	defer srcFile.Close()
	dstFile, err := this.dstClient.Create(rf) // 如果文件存在，create会清空原文件 openfile会追加
	if err != nil {
		log.Instance().Error("this.dstClient.Create", zap.Error(err))
		return err
	}
	defer dstFile.Close()

	buf := make([]byte, 10000)
	for {
		n, _ := srcFile.Read(buf)
		if n == 0 {
			break
		}
		dstFile.Write(buf[:n]) // 读多少 写多少
	}
	err = dao.NewTaskLogDaoImpl().Insert(model.TaskLog{
		TaskId:     this.task.Id,
		ServerId:   this.task.DstServer,
		SourcePath: path.Join(this.task.SourcePath, fname),
		DstPath:    rf,
		Size:       fsize,
		CreateTime: time.Now(),
	})
	if err != nil {
		return err
	}
	log.Instance().Info(fmt.Sprintf("【%s】传输完毕", fname))
	return nil
}

func (this *RunTask) MkRemotedir(p string) error {
	p = check.CheckWinPath(p)
	dst := path.Join(this.task.DstPath, p)
	err := this.dstClient.MkdirAll(dst)
	if err != nil {
		log.Instance().Error("MkRemotedir", zap.Error(err))
		return err
	}
	return nil
}
