package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// fid 是 base64 形态、可能含 '/'：路径段会破坏路由，必须支持 query 传递。
func TestShareFileID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name  string
		url   string
		param string
		want  string
	}{
		{"query 优先且保留特殊字符", "/s/x/download?fid=MYZTj%2FabC%2BdeF", "", "MYZTj/abC+deF"},
		{"兼容旧路径参数", "/s/x/download/plain-id", "plain-id", "plain-id"},
		{"都为空", "/s/x/download", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", tc.url, nil)
			if tc.param != "" {
				c.Params = gin.Params{{Key: "fileId", Value: tc.param}}
			}
			if got := shareFileID(c); got != tc.want {
				t.Fatalf("shareFileID() = %q, want %q", got, tc.want)
			}
		})
	}
}
