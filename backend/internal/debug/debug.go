package debug

import (
	"fmt"
	"log"
	"os"
	"strings"
)

var enabled bool

func Init() {
	v := os.Getenv("DEBUG")
	if v == "true" || v == "1" || v == "yes" {
		enabled = true
	}
}

func IsEnabled() bool {
	return enabled
}

func Log(v ...interface{}) {
	if !enabled {
		return
	}
	args := []interface{}{"[DEBUG]"}
	args = append(args, v...)
	log.Println(args...)
}

func Logf(format string, v ...interface{}) {
	if !enabled {
		return
	}
	if !strings.HasSuffix(format, "\n") {
		format = format + "\n"
	}
	fmt.Printf("[DEBUG] "+format, v...)
}
