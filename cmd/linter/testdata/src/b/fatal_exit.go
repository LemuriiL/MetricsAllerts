package b

import (
	"log"
	"os"
)

func f() {
	log.Fatal("bad") // want "do not use log.Fatal outside main function of main package"
	os.Exit(1)       // want "do not use os.Exit outside main function of main package"
}
