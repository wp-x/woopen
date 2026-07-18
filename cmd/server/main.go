package main

import (
	"log"

	"woopen/internal/config"
	"woopen/internal/middleware"
	"woopen/internal/repository"
	"woopen/internal/wopan"
)

func main() {
	// 加载配置
	cfg := config.DefaultConfig()
	logStartup(cfg)

	// 设置JWT密钥
	middleware.JWTSecret = cfg.JWTSecret

	// 初始化数据库
	db, err := repository.NewDatabase(cfg.DatabasePath())
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer db.Close()

	repos := initRepositories(db)
	wopanClient := initWopanClient(repos.settings)
	h := initHandler(repos, wopanClient, cfg.AdminPassword)
	monitorService := initMonitorService(repos, wopanClient, h)
	maybeStartMonitor(repos.settings, monitorService)

	r := setupRouter(routeOptions{
		handler: h,
	})
	davHandler := newWebDAVHandler(wopanClient, repos.settings, cfg.AdminPassword)
	startHTTPServer(r, davHandler, cfg.Port)
}

// initWopanClient 初始化云盘客户端
func initWopanClient(settingsRepo *repository.SettingsRepository) *wopan.Client {
	settings, err := settingsRepo.Get()
	if err != nil {
		log.Printf("获取设置失败: %v", err)
		return wopan.NewClient("")
	}

	if settings.RefreshToken == "" && settings.AccessToken == "" {
		log.Println("未配置 refresh_token 或 access_token，请在管理后台配置")
		return wopan.NewClient("")
	}

	client := wopan.NewClient(settings.RefreshToken)
	client.SetRootFolderID(settings.RootFolderID)

	// AccessToken 可选；提供后可减少首次刷新
	if settings.AccessToken != "" {
		client.SetAccessToken(settings.AccessToken)
	}

	// 设置Token更新回调
	client.SetTokenUpdateCallback(func(refreshToken, accessToken string) {
		if err := settingsRepo.UpdateToken(refreshToken, accessToken); err != nil {
			log.Printf("保存Token失败: %v", err)
		} else {
			log.Println("Token已自动更新并保存")
		}
	})

	// 初始化客户端（个人云模式）
	if err := client.Init(); err != nil {
		log.Printf("初始化云盘客户端失败: %v (请检查Token是否有效)", err)
		return client
	}

	log.Println("云盘客户端初始化成功")
	return client
}
