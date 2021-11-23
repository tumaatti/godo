package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

type Todo struct {
	gorm.Model
	Id        int `gorm:"primaryKey"`
	CreatedAt string
	Content   string
	Done      bool `gorm:"default:false"`
	Tags      string
}

func (todo Todo) generateContents() string {
	return "DONE: " + formatCheckMark(todo.Done) + "; TAGS: " + todo.Tags + "\n" + todo.Content
}

func (todo Todo) printRowString(commands CommandMap, Cok bool, maxLen int) {
	firstLineOfContent := strings.Split(todo.Content, "\n")[0]

	if len(firstLineOfContent) > 80 {
		firstLineOfContent = firstLineOfContent[:80] + "..."
	}

	checkMark := formatCheckMark(todo.Done)
	tags := formatTags(todo.Tags)

	paddingLen := maxLen - len(strconv.Itoa(todo.Id))

	paddedSpaces := strings.Repeat(" ", paddingLen)

	if Cok {
		fmt.Printf("%d %s %s  %s  %s   |   %s\n",
			todo.Id,
			paddedSpaces,
			todo.CreatedAt,
			checkMark,
			firstLineOfContent,
			tags,
		)
	} else {
		fmt.Printf("%d %s %s  %s   |   %s\n",
			todo.Id,
			paddedSpaces,
			checkMark,
			firstLineOfContent,
			tags,
		)
	}
}

func (todo Todo) createNewTmpFile(filepath string) error {
	return ioutil.WriteFile(filepath, []byte(todo.generateContents()), 0755)
}
