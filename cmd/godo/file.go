package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

type File struct {
	directoryName string
	filepath      string
	content       []byte
}

func (file File) createDirIfDoesNotExist() {
	_, err := os.Stat(file.directoryName)

	if os.IsNotExist(err) {
		os.Mkdir(file.filepath, 0755)
	}
}

func (file File) editInNvim() {
	cmd := exec.Command("nvim", file.filepath)
	// cmd needs the stdin etc to function correctly
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if err != nil {
		log.Fatal(err)
	}
}

func (file File) parseFile() (string, string, bool) {
	var tags string

	splitByLines := strings.Split(string(file.content), "\n")
	contentList := splitByLines[1:]
	content := strings.Join(contentList, "\n")

	tagsRow := splitByLines[0]
	splitTagsRow := strings.Split(tagsRow, ";")

	checkBox := strings.TrimSpace(strings.Split(splitTagsRow[0], ":")[1])
	done := unFormatCheckMark(checkBox)

	tagsList := strings.Split(splitTagsRow[1], ":")

	if len(tagsList) < 2 {
		tags = ""
	} else {
		tags = strings.TrimSpace(tagsList[1])
	}

	return content, tags, done
}
