package gin

import (
	"chatgpt-adapter/core/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iocgo/sdk"
	"github.com/iocgo/sdk/env"
	"github.com/iocgo/sdk/router"
	"net/http"
	"net/http/httputil"
	"strings"
)

var (
	debug bool
)

// @Inject(lazy="false", name="ginInitializer")
func Initialized(env *env.Environment) sdk.Initializer {
	debug = env.GetBool("server.debug")
	return sdk.InitializedWrapper(0, func(container *sdk.Container) (err error) {
		sdk.ProvideTransient(container, sdk.NameOf[*gin.Engine](), func() (engine *gin.Engine, err error) {
			if !debug {
				gin.SetMode(gin.ReleaseMode)
			}

			engine = gin.Default()
			{
				engine.Use(gin.Recovery())
				engine.Use(cros)
				engine.Use(token)
			}
			engine.Static("/file/", "tmp")
			beans := sdk.ListInvokeAs[router.Router](container)
			for _, route := range beans {
				route.Routers(engine)
			}

			return
		})
		return
	})
}

func token(gtx *gin.Context) {
	str := gtx.Request.Header.Get("X-Api-Key")
	if str == "" {
		str = strings.TrimPrefix(gtx.Request.Header.Get("Authorization"), "Bearer ")
	}
	gtx.Set("token", str)
}

func cros(gtx *gin.Context) {
	method := gtx.Request.Method
	gtx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	gtx.Header("Access-Control-Allow-Origin", "*") // 设置允许访问所有域
	gtx.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE,UPDATE")
	gtx.Header("Access-Control-Allow-Headers", "*")
	gtx.Header("Access-Control-Expose-Headers", "*")
	gtx.Header("Access-Control-Max-Age", "172800")
	gtx.Header("Access-Control-Allow-Credentials", "false")
	//gtx.Set("content-type", "application/json")

	if method == "OPTIONS" {
		gtx.Status(http.StatusOK)
		return
	}

	if gtx.Request.RequestURI == "/" ||
		gtx.Request.RequestURI == "/favicon.ico" ||
		strings.Contains(gtx.Request.URL.Path, "/v1/models") ||
		strings.HasPrefix(gtx.Request.URL.Path, "/file/") {
		// 处理请求
		gtx.Next()
		return
	}

	uid := uuid.NewString()
	// 请求打印
	data, _ := httputil.DumpRequest(gtx.Request, debug)
	logger.Infof("------ START REQUEST %s ---------", uid)
	println(string(data))

	// 处理请求
	gtx.Next()

	// 结束处理
	logger.Infof("------ END REQUEST %s ---------", uid)
}
