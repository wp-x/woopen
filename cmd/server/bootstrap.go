package main

import (
	"log"
	"net/http"
	"strings"

	"woopen/internal/config"
	"woopen/internal/handler"
	"woopen/internal/middleware"
	"woopen/internal/repository"
	"woopen/internal/service"
	"woopen/internal/webdav"
	"woopen/internal/wopan"

	"github.com/gin-gonic/gin"
	xwebdav "golang.org/x/net/webdav"
)

type repoBundle struct {
	settings     *repository.SettingsRepository
	share        *repository.ShareRepository
	accessLog    *repository.AccessLogRepository
	monitor      *repository.MonitorRepository
	notification *repository.NotificationLogRepository
}

func logStartup(cfg *config.Config) {
	log.Printf("WoOpen 正在启动...")
	log.Printf("数据目录: %s", cfg.DataDir)
	log.Printf("服务端口: %s", cfg.Port)
}

func initRepositories(db *repository.Database) repoBundle {
	return repoBundle{
		settings:     repository.NewSettingsRepository(db.DB()),
		share:        repository.NewShareRepository(db.DB()),
		accessLog:    repository.NewAccessLogRepository(db.DB()),
		monitor:      repository.NewMonitorRepository(db.DB()),
		notification: repository.NewNotificationLogRepository(db.DB()),
	}
}

func initHandler(repos repoBundle, wopanClient *wopan.Client, adminPassword string) *handler.Handler {
	return handler.NewHandler(
		repos.settings,
		repos.share,
		repos.accessLog,
		repos.monitor,
		repos.notification,
		wopanClient,
		adminPassword,
	)
}

func initMonitorService(repos repoBundle, wopanClient *wopan.Client, h *handler.Handler) *service.MonitorService {
	notifier := service.NewNotifier(repos.notification)
	monitorService := service.NewMonitorService(repos.settings, repos.monitor, wopanClient, notifier)
	h.SetMonitorService(monitorService)
	return monitorService
}

func maybeStartMonitor(settingsRepo *repository.SettingsRepository, monitorService *service.MonitorService) {
	settings, _ := settingsRepo.Get()
	if settings != nil && settings.MonitorEnabled {
		monitorService.SetInterval(settings.MonitorInterval)
		monitorService.Start()
	}
}

func setupRouter(opts routeOptions) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(middleware.CORSMiddleware())
	registerRoutes(r, opts)
	return r
}

func newWebDAVHandler(wopanClient *wopan.Client, adminPassword string) http.Handler {
	davFS := webdav.NewWopanFS(wopanClient)
	dav := &xwebdav.Handler{
		Prefix:     "/dav",
		FileSystem: davFS,
		LockSystem: xwebdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				log.Printf("[WebDAV] %s %s → error: %v", r.Method, r.URL.Path, err)
			}
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[WebDAV] %s %s", r.Method, r.URL.RequestURI())

		// 所有响应都携带 DAV 能力头，让客户端识别为 WebDAV 服务
		w.Header().Set("DAV", "1, 2")
		w.Header().Set("MS-Author-Via", "DAV")

		// OPTIONS 免认证
		if r.Method == http.MethodOptions {
			dav.ServeHTTP(w, r)
			return
		}

		if !authenticateWebDAV(w, r, adminPassword) {
			return
		}

		// GET/HEAD 目录：x/net/webdav 对目录返回 405，但客户端期望 200
		if r.Method == http.MethodHead || r.Method == http.MethodGet {
			reqPath := strings.TrimPrefix(r.URL.Path, "/dav")
			if reqPath == "" {
				reqPath = "/"
			}
			if fi, err := davFS.Stat(r.Context(), reqPath); err == nil && fi.IsDir() {
				w.Header().Set("Content-Type", "httpd/unix-directory")
				w.Header().Set("Content-Length", "0")
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		dav.ServeHTTP(w, r)
	})
}

func authenticateWebDAV(w http.ResponseWriter, r *http.Request, adminPassword string) bool {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		davChallenge(w)
		return false
	}
	if strings.HasPrefix(auth, "Bearer ") {
		if !middleware.ValidateToken(strings.TrimPrefix(auth, "Bearer ")) {
			davChallenge(w)
			return false
		}
		return true
	}
	user, pass, ok := r.BasicAuth()
	if !ok || user != "admin" || pass != adminPassword {
		log.Printf("[WebDAV] auth failed: user=%q ok=%v", user, ok)
		davChallenge(w)
		return false
	}
	return true
}

func davChallenge(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="WoOpen WebDAV"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

func startHTTPServer(ginEngine *gin.Engine, davHandler http.Handler, port string) {
	mux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/dav" || strings.HasPrefix(r.URL.Path, "/dav/") {
			davHandler.ServeHTTP(w, r)
			return
		}
		ginEngine.ServeHTTP(w, r)
	})
	log.Printf("WoOpen 启动成功! 访问: http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
}
