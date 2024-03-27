package router

import (
	"net/http"
	"time"

	"github.com/ccfos/nightingale/v6/alert/sender"
	"github.com/ccfos/nightingale/v6/models"

	"github.com/gin-gonic/gin"
	"github.com/toolkits/pkg/ginx"
	"github.com/toolkits/pkg/str"
)

func (rt *Router) taskGets(c *gin.Context) {
	bgid := ginx.UrlParamInt64(c, "id")
	mine := ginx.QueryBool(c, "mine", false)
	days := ginx.QueryInt64(c, "days", 7)
	limit := ginx.QueryInt(c, "limit", 20)
	query := ginx.QueryStr(c, "query", "")
	user := c.MustGet("user").(*models.User)

	creator := ""
	if mine {
		creator = user.Username
	}

	beginTime := time.Now().Unix() - days*24*3600

	total, err := models.TaskRecordTotal(rt.Ctx, []int64{bgid}, beginTime, creator, query)
	ginx.Dangerous(err)

	list, err := models.TaskRecordGets(rt.Ctx, []int64{bgid}, beginTime, creator, query, limit, ginx.Offset(c, limit))
	ginx.Dangerous(err)

	ginx.NewRender(c).Data(gin.H{
		"total": total,
		"list":  list,
	}, nil)
}

func (rt *Router) taskGetsByGids(c *gin.Context) {
	gids := str.IdsInt64(ginx.QueryStr(c, "gids"), ",")
	if len(gids) == 0 {
		ginx.NewRender(c, http.StatusBadRequest).Message("arg(gids) is empty")
		return
	}

	for _, gid := range gids {
		rt.bgroCheck(c, gid)
	}

	mine := ginx.QueryBool(c, "mine", false)
	days := ginx.QueryInt64(c, "days", 7)
	limit := ginx.QueryInt(c, "limit", 20)
	query := ginx.QueryStr(c, "query", "")
	user := c.MustGet("user").(*models.User)

	creator := ""
	if mine {
		creator = user.Username
	}

	beginTime := time.Now().Unix() - days*24*3600

	total, err := models.TaskRecordTotal(rt.Ctx, gids, beginTime, creator, query)
	ginx.Dangerous(err)

	list, err := models.TaskRecordGets(rt.Ctx, gids, beginTime, creator, query, limit, ginx.Offset(c, limit))
	ginx.Dangerous(err)

	ginx.NewRender(c).Data(gin.H{
		"total": total,
		"list":  list,
	}, nil)
}

type taskForm struct {
	Title     string   `json:"title" binding:"required"`
	Account   string   `json:"account" binding:"required"`
	Batch     int      `json:"batch"`
	Tolerance int      `json:"tolerance"`
	Timeout   int      `json:"timeout"`
	Pause     string   `json:"pause"`
	Script    string   `json:"script" binding:"required"`
	Args      string   `json:"args"`
	Action    string   `json:"action" binding:"required"`
	Creator   string   `json:"creator"`
	Hosts     []string `json:"hosts" binding:"required"`
}

func (rt *Router) taskRecordAdd(c *gin.Context) {
	var f *models.TaskRecord
	ginx.BindJSON(c, &f)
	ginx.NewRender(c).Message(f.Add(rt.Ctx))
}

func (rt *Router) taskAdd(c *gin.Context) {
	var f models.TaskForm
	ginx.BindJSON(c, &f)

	bgid := ginx.UrlParamInt64(c, "id")
	user := c.MustGet("user").(*models.User)
	f.Creator = user.Username

	err := f.Verify()
	ginx.Dangerous(err)

	f.HandleFH(f.Hosts[0])

	// check permission
	rt.checkTargetPerm(c, f.Hosts)

	// call ibex
	taskId, err := sender.TaskAdd(f, user.Username, rt.Ctx.IsCenter)
	// taskId, err := TaskCreate(f, rt.NotifyConfigCache.GetIbex())
	ginx.Dangerous(err)

	if taskId <= 0 {
		ginx.Dangerous("created task.id is zero")
	}

	// write db
	record := models.TaskRecord{
		Id:      taskId,
		GroupId: bgid,
		// IbexAddress:  rt.NotifyConfigCache.GetIbex().Address,
		// IbexAuthUser: rt.NotifyConfigCache.GetIbex().BasicAuthUser,
		// IbexAuthPass: rt.NotifyConfigCache.GetIbex().BasicAuthPass,
		Title:     f.Title,
		Account:   f.Account,
		Batch:     f.Batch,
		Tolerance: f.Tolerance,
		Timeout:   f.Timeout,
		Pause:     f.Pause,
		Script:    f.Script,
		Args:      f.Args,
		CreateAt:  time.Now().Unix(),
		CreateBy:  f.Creator,
	}

	err = record.Add(rt.Ctx)
	ginx.NewRender(c).Data(taskId, err)
}

// func (rt *Router) taskProxy(c *gin.Context) {
// 	target, err := url.Parse(rt.NotifyConfigCache.GetIbex().Address)
// 	if err != nil {
// 		ginx.NewRender(c).Message("invalid ibex address: %s", rt.NotifyConfigCache.GetIbex().Address)
// 		return
// 	}

// 	director := func(req *http.Request) {
// 		req.URL.Scheme = target.Scheme
// 		req.URL.Host = target.Host

// 		// fe request e.g. /api/n9e/busi-group/:id/task/*url
// 		index := strings.Index(req.URL.Path, "/task/")
// 		if index == -1 {
// 			panic("url path invalid")
// 		}

// 		req.URL.Path = "/ibex/v1" + req.URL.Path[index:]

// 		if target.RawQuery == "" || req.URL.RawQuery == "" {
// 			req.URL.RawQuery = target.RawQuery + req.URL.RawQuery
// 		} else {
// 			req.URL.RawQuery = target.RawQuery + "&" + req.URL.RawQuery
// 		}

// 		if rt.NotifyConfigCache.GetIbex().BasicAuthUser != "" {
// 			req.SetBasicAuth(rt.NotifyConfigCache.GetIbex().BasicAuthUser, rt.NotifyConfigCache.GetIbex().BasicAuthPass)
// 		}
// 	}

// 	errFunc := func(w http.ResponseWriter, r *http.Request, err error) {
// 		ginx.NewRender(c, http.StatusBadGateway).Message(err)
// 	}

// 	proxy := &httputil.ReverseProxy{
// 		Director:     director,
// 		ErrorHandler: errFunc,
// 	}

// 	proxy.ServeHTTP(c.Writer, c.Request)
// }
