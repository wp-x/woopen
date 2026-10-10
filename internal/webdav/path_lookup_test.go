package webdav

import (
	"context"
	"testing"
	"woopen/internal/model"
	"woopen/internal/wopan"
)

func TestResolveRootUsesAccountRootID(t *testing.T) {
	for _, rootID := range []string{"", "0", "account-sync-root"} {
		t.Run(rootID, func(t *testing.T) {
			client := wopan.NewClient("")
			client.SetRootFolderID(rootID)
			fs := NewWopanFS(client)
			want := rootID
			if want == "" {
				want = "0"
			}
			for _, name := range []string{"/", "."} {
				resolved, err := fs.ResolvePath(context.Background(), name)
				if err != nil || resolved.ID != want || resolved.ParentID != want {
					t.Fatalf("ResolvePath(%q) = %+v, %v; want root %q", name, resolved, err, want)
				}
				id, _, _, err := fs.lookupPath(context.Background(), name)
				if err != nil || id != want {
					t.Fatalf("lookupPath(%q) = %q, %v", name, id, err)
				}
			}
		})
	}
}

func TestResolveSyncDirectoryUsesReturnedNameAndIDs(t *testing.T) {
	for _, name := range []string{"我的同步盘", "同步盘", "重命名的目录"} {
		t.Run(name, func(t *testing.T) {
			client := wopan.NewClient("")
			client.SetRootFolderID("account-root")
			fs := NewWopanFS(client)
			fs.setDirCacheEntry("account-root", []*model.FileInfo{
				{ID: "sync-id", Name: name, IsDir: true},
			})
			fs.setDirCacheEntry("sync-id", []*model.FileInfo{
				{ID: "file-operation-id", FID: "download-fid", Name: "hello.txt"},
			})
			resolved, err := fs.ResolvePath(context.Background(), "/"+name+"/hello.txt")
			if err != nil {
				t.Fatal(err)
			}
			if resolved.ID != "file-operation-id" || resolved.ParentID != "sync-id" {
				t.Fatalf("incorrect cloud IDs: %+v", resolved)
			}
			if DownloadIDFor(resolved.Info) != "download-fid" {
				t.Fatal("download must use fid")
			}
		})
	}
}
