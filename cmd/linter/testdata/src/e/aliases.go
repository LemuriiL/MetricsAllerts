package e

import (
	l "log"
	o "os"
)

func f() {
	l.Fatal("bad") // want "do not use log.Fatal outside main function of main package"
	o.Exit(1)      // want "do not use os.Exit outside main function of main package"
}
