package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

func main() {
	fmt.Println("mass rename")
	files, _ := filepath.Glob("./FINAL/14-moroni/*")

	re := regexp.MustCompile("\\d{3}-Moro.")
	for _, file := range files {
		newName := re.ReplaceAllString(file, "")
		os.Rename(file, newName)
		fmt.Println(file, newName)
	}
}
