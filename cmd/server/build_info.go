package main

import "fmt"

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

func printBuildInfo() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}
