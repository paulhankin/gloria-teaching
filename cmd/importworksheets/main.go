package main

import (
	"flag"
	"log"

	"learningmaterial/internal/worksheetrepo"
)

func main() {
	usersRoot := flag.String("users", "", "directory containing per-user worksheet repositories")
	flag.Parse()
	if _, err := worksheetrepo.Prepare(*usersRoot); err != nil {
		log.Fatal(err)
	}
}
