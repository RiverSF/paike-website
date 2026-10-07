package path

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	wdOnce sync.Once
	wdVal  string
)

// Root 返回应用根目录，可通过 APP_ROOT 覆盖（二进制不在项目根目录运行时使用）。
func Root() string {
	if v := os.Getenv("APP_ROOT"); v != "" {
		return v
	}
	wdOnce.Do(func() {
		wd, err := os.Getwd()
		if err != nil {
			wdVal = "."
			return
		}
		wdVal = wd
	})
	return wdVal
}

// Join 拼接应用根目录下的路径。
func Join(elem ...string) string {
	if len(elem) == 0 {
		return Root()
	}
	parts := make([]string, 0, len(elem)+1)
	parts = append(parts, Root())
	parts = append(parts, elem...)
	return filepath.Join(parts...)
}
