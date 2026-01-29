// Package dependency 测试 GetDownloader 与 HTTP 下载器
package dependency

import (
	"context"
	"testing"
)

func TestGetDownloader(t *testing.T) {
	tests := []struct {
		source   string
		wantErr  bool
		wantType string
	}{
		{"git", false, "git"},
		{"http", false, "http"},
		{"https", false, "http"},
		{"unknown", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			dl, err := GetDownloader(tt.source)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDownloader(%q) err = %v, wantErr %v", tt.source, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			switch tt.wantType {
			case "git":
				if _, ok := dl.(*GitDownloader); !ok {
					t.Errorf("GetDownloader(%q) = %T, want *GitDownloader", tt.source, dl)
				}
			case "http":
				if _, ok := dl.(*HttpDownloader); !ok {
					t.Errorf("GetDownloader(%q) = %T, want *HttpDownloader", tt.source, dl)
				}
			}
		})
	}
}

func TestHttpDownloader_Download_MissingURL(t *testing.T) {
	dl := NewHttpDownloader()
	ctx := context.Background()
	err := dl.Download(ctx, "pkg", "1.0", t.TempDir(), nil)
	if err == nil {
		t.Error("expected error when options has no url")
	}
	opt := map[string]interface{}{"other": "value"}
	err = dl.Download(ctx, "pkg", "1.0", t.TempDir(), opt)
	if err == nil {
		t.Error("expected error when options has no url")
	}
}
