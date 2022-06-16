package runtask

import (
	"fmt"
	"go.uber.org/zap"
	"os"
	"path"
	"pear-admin-go/app/core/log"
	"pear-admin-go/app/dao"
	"pear-admin-go/app/global/e"
	"pear-admin-go/app/model"
	"pear-admin-go/app/util/check"
	"strings"
	"sync/atomic"
	"time"
)

// 远端->远端  远端->本地

// RunR2R 远端->远端
func (this *RunTask) RunR2R() {
	this.WalkRemotePath(this.task.SourcePath, e.ToRemote)
}

// RunR2L 远端到本地
func (this *RunTask) RunR2L() {
	this.WalkRemotePath(this.task.SourcePath, e.ToLocal)
}

func (this *RunTask) WalkRemotePath(dirPath string, runType int) {
	globPath := pathJoin(dirPath)
	files, err := this.sourceClient.Glob(globPath)
	if err != nil {
		log.Instance().Error("WalkRemotePath.this.sourceClient.Glob", zap.Error(err))
		return
	}
	for _, v := range files {
		stat, err := this.sourceClient.Stat(v)
		if err != nil {
			log.Instance().Error("WalkRemotePath.this.sourceClient.Stat", zap.Error(err))
			continue
		}
		if stat.IsDir() {
			if runType == e.ToRemote {
				dname := string([]rune(strings.ReplaceAll(v, this.task.SourcePath, ""))[1:])
				err = this.MkRemotedir(dname)
				if err != nil {
					log.Instance().Error("WalkRemotePath.MkRemotedir", zap.Error(err))
					return
				}
			}
			this.WalkRemotePath(v, runType)
		} else {
			this.fp.Add(1)
			atomic.AddUint64(&this.counter, 1)
			go func(v string, size int64) {
				defer this.fp.Done()
				if runType == e.ToLocal {
					err = this.RemoteSendLocal(v, size)
				} else if runType == e.ToRemote {
					err = this.RemoteSendRemote(v, size)
				}
				if err != nil {
					log.Instance().Error("WalkRemotePath.RemoteToLocal", zap.Error(err))
				}
			}(v, stat.Size())
		}
	}
}

func (this *RunTask) RemoteSendRemote(fname string, fsize int64) error {
	newName := strings.ReplaceAll(fname, this.task.SourcePath, "")
	rf := path.Join(this.task.DstPath, newName) // 文件在目标服务器的路径及名称

	srcFile, err := this.sourceClient.Open(fname)
	if err != nil {
		log.Instance().Error("RemoteToLocal.sourceClient.Open", zap.Error(err))
		return err
	}
	defer srcFile.Close()
	md, err := GetMd(srcFile)
	if err != nil {
		log.Instance().Error("RemoteToLocal.sourceClient.GetMd", zap.Error(err))
		return err
	}
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
	dstFile, err := this.dstClient.Create(rf) // 如果文件存在，create会清空原文件 openfile会追加
	if err != nil {
		log.Instance().Error("RemoteSendRemote.this.dstClient.Create", zap.Error(err))
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
	log.Instance().Info(fmt.Sprintf("【%s】传输完毕", fname))
	err = dao.NewTaskLogDaoImpl().Insert(model.TaskLog{
		TaskId:     this.task.Id,
		ServerId:   this.task.SourceServer,
		SourcePath: fname,
		MD:         md,
		DstPath:    rf,
		Size:       fsize,
		CreateTime: time.Now(),
	})
	if err != nil {
		log.Instance().Error("RemoteSendRemote.dao.NewTaskLogDaoImpl.Insert", zap.Error(err))
		return err
	}
	return nil
}

// RemoteSendLocal 远端->本地 使用 sourceClient
func (this *RunTask) RemoteSendLocal(fname string, fsize int64) error { // 本地文件夹
	dstFile := path.Join(this.task.DstPath, strings.ReplaceAll(fname, this.task.SourcePath, "")) // 需要保存的本地文件地址

	srcFile, err := this.sourceClient.Open(fname)
	if err != nil {
		log.Instance().Error("RemoteToLocal.sourceClient.Open", zap.Error(err))
		return err
	}
	defer srcFile.Close()
	md, err := GetMd(srcFile)
	if err != nil {
		log.Instance().Error("RemoteToLocal.GetMd", zap.Error(err))
	}
	has, err := check.CheckFile(dstFile) // 是否已存在
	if err != nil {
		log.Instance().Error("RemoteToLocal.CheckFile", zap.Error(err))
		return err
	}
	if has != nil {
		if has.Size() == fsize {
			log.Instance().Debug(fmt.Sprintf("文件%s已存在", dstFile))
			return nil
		} else { // 续传
			err = continuation(dstFile, srcFile, has)
			if err != nil {
				return err
			}
		}
	} else {
		err = create(dstFile, srcFile)
		if err != nil {
			return err
		}
	}

	err = dao.NewTaskLogDaoImpl().Insert(model.TaskLog{
		TaskId:     this.task.Id,
		ServerId:   this.task.SourceServer,
		SourcePath: fname,
		DstPath:    dstFile,
		Size:       fsize,
		MD:         md,
		CreateTime: time.Now(),
	})
	if err != nil {
		log.Instance().Error("RemoteSendRemote.dao.NewTaskLogDaoImpl.Insert", zap.Error(err))
		return err
	}
	return nil
}
