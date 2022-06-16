package runtask

import (
	"github.com/pkg/sftp"
	"go.uber.org/zap"
	"pear-admin-go/app/core/log"
	"pear-admin-go/app/core/scli"
	"pear-admin-go/app/dao"

	"pear-admin-go/app/global/e"
	"pear-admin-go/app/model"
	"pear-admin-go/app/util/pool"
)

type RunTask struct {
	task         model.Task
	sourceClient *sftp.Client // 源服务器连接
	dstClient    *sftp.Client // 目标服务器连接
	fp           *pool.Pool   // chan pool
	counter      uint64       // 文件传输计数器
}

func NewRunTask(task model.Task) *RunTask {
	return &RunTask{
		task:    task,
		fp:      pool.NewPool(e.MaxPool),
		counter: 0,
	}
}

func (this *RunTask) SetSourceClient() *RunTask {
	if this.task.SourceType == e.Local {
		return this
	}
	c, err := this.getClient(this.task.SourceServer)
	if err != nil {
		log.Instance().Error("SetSourceClient.getClient", zap.Error(err))
		return this
	}
	this.sourceClient = c
	return this
}

func (this *RunTask) SetDstClient() *RunTask {
	if this.task.DstType == e.Local {
		return this
	}
	c, err := this.getClient(this.task.DstServer)
	if err != nil {
		log.Instance().Error("SetSourceClient.getClient", zap.Error(err))
		return this
	}
	this.dstClient = c
	return this
}

func (this *RunTask) getClient(sid int) (*sftp.Client, error) {
	server, err := dao.NewTaskServerDaoImpl().FindOne(sid)
	if err != nil {
		return nil, err
	}
	c, err := scli.Instance(*server)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (this *RunTask) Run() {
	if this.task.SourceType == e.Remote && this.task.DstType == e.Remote {
		this.RunR2R()
	} else if this.task.SourceType == e.Remote && this.task.DstType == e.Local {
		this.RunR2L()
	} else if this.task.SourceType == e.Local && this.task.DstType == e.Remote {
		this.RunL2R()
	}
	err := dao.NewTaskDaoImpl().Update(this.task, map[string]interface{}{"task_file_num": this.counter})
	if err != nil {
		log.Instance().Error("Run.Update", zap.Error(err))
	}
}
