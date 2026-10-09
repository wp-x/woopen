package webdav

import (
	"testing"
	"woopen/internal/model"
)

// TestDownloadIDFor 验证下载标识优先使用 fid（联通下载接口拒绝十六进制 id），
// fid 缺失时回退 id，保证同步盘等特殊空间的文件也能正确取链。
func TestDownloadIDFor(t *testing.T) {
	tests := []struct {
		name string
		info *model.FileInfo
		want string
	}{
		{
			name: "nil info",
			info: nil,
			want: "",
		},
		{
			name: "fid preferred",
			info: &model.FileInfo{ID: "84a641abd0ae4b25b92bc9087b61b2b5", FID: "QWwTG_pHKQtUapdAmTm1rBXsb"},
			want: "QWwTG_pHKQtUapdAmTm1rBXsb",
		},
		{
			name: "fallback to id when fid empty",
			info: &model.FileInfo{ID: "84a641abd0ae4b25b92bc9087b61b2b5"},
			want: "84a641abd0ae4b25b92bc9087b61b2b5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DownloadIDFor(tt.info); got != tt.want {
				t.Errorf("DownloadIDFor() = %q, want %q", got, tt.want)
			}
		})
	}
}
