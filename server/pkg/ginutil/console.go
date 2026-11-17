package ginutil

import (
	"runtime"

	"github.com/gin-gonic/gin"
)

// DisableColorOnWindows turns off Gin logger ANSI escapes on Windows consoles
// that often print them as garbled text (e.g. [97;42m) instead of colors.
func DisableColorOnWindows() {
	if runtime.GOOS == "windows" {
		gin.DisableConsoleColor()
	}
}
