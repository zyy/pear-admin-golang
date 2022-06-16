package router

import (
	"github.com/gin-gonic/gin"
)

func TaskRouter(r *gin.Engine) {
	tr := r.Group("system")
	tr.GET("server/list", handler.ServerList)
	tr.GET("server/json", handler.ServerJson)
	tr.GET("server/add", handler.ServerAdd)
	tr.POST("server/add", handler.ServerAdd)
	tr.GET("server/edit", handler.ServerEdit)
	tr.POST("server/edit", handler.ServerEdit)
	tr.POST("server/del", handler.ServerDel)

	tr.GET("task/list", handler.TaskList)
	tr.GET("task/json", handler.TaskJson)
	tr.GET("task/add", handler.TaskAdd)
	tr.POST("task/add", handler.TaskAdd)
	tr.GET("task/edit", handler.TaskEdit)
	tr.POST("task/edit", handler.TaskEdit)
	tr.POST("task/del", handler.TaskDel)
}
