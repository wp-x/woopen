package repository

import (
	"path/filepath"
	"testing"

	"woopen/internal/model"
)

// 验证迁移后的列顺序与 Scan 对齐：直连分享写入->读回，设置默认值。
func TestDirectShareAndSettingsRoundtrip(t *testing.T) {
	db, err := NewDatabase(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("建库失败: %v", err)
	}
	defer db.Close()

	shareRepo := NewShareRepository(db.DB())
	s := &model.Share{
		ShareCode:  "img1",
		TargetType: "file",
		TargetID:   "fid-123",
		TargetPath: "/a.png",
		TargetName: "a.png",
		IsActive:   true,
		IsDirect:   true,
	}
	if err := shareRepo.Create(s); err != nil {
		t.Fatalf("创建分享失败: %v", err)
	}

	got, err := shareRepo.GetByCode("img1")
	if err != nil {
		t.Fatalf("读取分享失败: %v", err)
	}
	if !got.IsDirect || got.TargetID != "fid-123" || got.TargetName != "a.png" {
		t.Fatalf("字段错位: %+v", got)
	}

	got.IsDirect = false
	if err := shareRepo.Update(got); err != nil {
		t.Fatalf("更新失败: %v", err)
	}
	if again, _ := shareRepo.GetByCode("img1"); again.IsDirect {
		t.Fatal("更新 is_direct 未生效")
	}

	settingsRepo := NewSettingsRepository(db.DB())
	set, err := settingsRepo.Get()
	if err != nil {
		t.Fatalf("读取设置失败: %v", err)
	}
	if !set.DirectAllowEmptyReferer || set.DirectRateLimit != 60 || !set.DirectRejectPlaceholder {
		t.Fatalf("直连默认值错误: %+v", set)
	}

	set.DirectRefererWhitelist = "example.com"
	set.DirectRateLimit = 10
	if err := settingsRepo.Update(set); err != nil {
		t.Fatalf("保存设置失败: %v", err)
	}
	reread, _ := settingsRepo.Get()
	if reread.DirectRefererWhitelist != "example.com" || reread.DirectRateLimit != 10 {
		t.Fatalf("设置回读错误: %+v", reread)
	}
}

func TestMigrationConvertsDirectFolderToTreeShare(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "folder.db")
	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("建库失败: %v", err)
	}
	repo := NewShareRepository(db.DB())
	share := &model.Share{
		ShareCode:  "docs",
		TargetType: "folder",
		TargetID:   "dir-id",
		TargetPath: "/docs",
		TargetName: "docs",
		IsActive:   true,
		IsDirect:   true,
	}
	if err := repo.Create(share); err != nil {
		t.Fatalf("写入历史数据失败: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("关闭数据库失败: %v", err)
	}

	reopened, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("重新迁移失败: %v", err)
	}
	defer reopened.Close()
	got, err := NewShareRepository(reopened.DB()).GetByCode("docs")
	if err != nil {
		t.Fatalf("读取分享失败: %v", err)
	}
	if got.IsDirect {
		t.Fatal("文件夹应被迁移为目录树分享")
	}
}
