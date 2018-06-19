package main

import "log"

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	bootstrap.Execute()
}
