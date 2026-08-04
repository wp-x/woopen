package handler

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"woopen/internal/model"
	"woopen/internal/repository"

	"github.com/gin-gonic/gin"
)

// 端到端验证 /f/:code 的路由与防盗链/占位图路径（在触达上游 GetDownloadURL 之前的分支）。
func TestDirectLinkRouteHotlink(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := repository.NewDatabase(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("建库失败: %v", err)
	}
	defer db.Close()

	settingsRepo := repository.NewSettingsRepository(db.DB())
	set, _ := settingsRepo.Get()
	set.DirectRefererWhitelist = "example.com"
	settingsRepo.Update(set)

	shareRepo := repository.NewShareRepository(db.DB())
	shareRepo.Create(&model.Share{ShareCode: "img", TargetType: "file", TargetID: "fid", TargetPath: "/a.png", TargetName: "a.png", IsActive: true, IsDirect: true})
	shareRepo.Create(&model.Share{ShareCode: "sec", TargetType: "file", TargetID: "fid2", TargetPath: "/b", TargetName: "b", IsActive: true, IsDirect: false})
	shareRepo.Create(&model.Share{ShareCode: "dir", TargetType: "folder", TargetID: "dir-id", TargetPath: "/docs", TargetName: "docs", IsActive: true, IsDirect: true})

	h := NewHandler(HandlerOptions{
		SettingsRepo:  settingsRepo,
		ShareRepo:     shareRepo,
		AccessLogRepo: repository.NewAccessLogRepository(db.DB()),
		ListCache:     NewListTTLCache(DefaultListCacheTTL),
		HTTPClient:    http.DefaultClient,
	})
	r := gin.New()
	r.Use(gin.Recovery()) // 与生产 gin.Default() 一致：上游不可用时 panic→500，不崩测试
	r.GET("/f/:code", h.DirectLink)

	do := func(code, referer, dest string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/f/"+code, nil)
		if referer != "" {
			req.Header.Set("Referer", referer)
		}
		if dest != "" {
			req.Header.Set("Sec-Fetch-Dest", dest)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	// 非直连分享走 /f/ → 404 占位图
	if w := do("sec", "", ""); w.Code != http.StatusNotFound {
		t.Fatalf("非直连应 404，得 %d", w.Code)
	}
	// 历史误标为直连的文件夹应回到目录树页，不请求文件下载直链。
	if w := do("dir", "", "document"); w.Code != http.StatusFound || w.Header().Get("Location") != "/s/dir" {
		t.Fatalf("文件夹应跳转目录树，得 %d %q", w.Code, w.Header().Get("Location"))
	}
	// 陌生 Referer 被白名单拦截 → 403 占位图
	if w := do("img", "https://evil.com", "image"); w.Code != http.StatusForbidden ||
		w.Header().Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("盗链应 403+svg，得 %d %s", w.Code, w.Header().Get("Content-Type"))
	}
	// 主动打开（Sec-Fetch-Dest: document）绕过白名单，进入取直链阶段（无上游→502）
	if w := do("img", "https://evil.com", "document"); w.Code == http.StatusForbidden {
		t.Fatal("document 应绕过防盗链")
	}
}
