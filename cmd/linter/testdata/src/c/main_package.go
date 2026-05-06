package main

import (
	"log"
	"os"
)

func helper() {
	log.Fatal("bad") // want "do not use log.Fatal outside main function of main package"
	os.Exit(1)       // want "do not use os.Exit outside main function of main package"
}

func main() {
}
