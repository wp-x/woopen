package handler

import "testing"

func TestDirectShareEnabled(t *testing.T) {
	cases := []struct {
		name       string
		targetType string
		requested  bool
		want       bool
	}{
		{name: "文件可开启直连", targetType: "file", requested: true, want: true},
		{name: "文件未请求直连", targetType: "file", requested: false, want: false},
		{name: "文件夹强制目录树", targetType: "folder", requested: true, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := directShareEnabled(tc.targetType, tc.requested); got != tc.want {
				t.Fatalf("directShareEnabled(%q, %v) = %v, want %v", tc.targetType, tc.requested, got, tc.want)
			}
		})
	}
}
