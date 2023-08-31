package handler

import (
	"FileSystem/common"
	"FileSystem/util"
	"github.com/gin-gonic/gin"
	"net/http"
)

/*
// HTTPInterceptor : http请求拦截器
func HTTPInterceptor(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			r.ParseForm()
			username := r.Form.Get("username")
			token := r.Form.Get("token")

			//验证登录token是否有效
			if len(username) < 3 || !IsTokenValid(token) {
				// w.WriteHeader(http.StatusForbidden)
				// token校验失败则跳转到登录页面
				http.Redirect(w, r, "/static/view/signin.html", http.StatusFound)
				return
			}

			h(w, r)
		})
}
*/
// HTTPInterceptor : http请求拦截器
func HTTPInterceptor() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Request.FormValue("username")
		token := c.Request.FormValue("token")

		//验证登录token是否有效
		if len(username) < 3 || !IsTokenValid(token) {
			// w.WriteHeader(http.StatusForbidden)
			// token校验失败则跳转到登录页面
			c.Abort()
			resp := util.NewRespMsg(
				int(common.StatusInvalidToken),
				"token无效",
				nil,
			)
			c.JSON(http.StatusOK, resp)
			return
		}
		c.Next()
	}
}

