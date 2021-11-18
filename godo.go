package main

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Todo struct {
	gorm.Model
	Id        int `gorm:"primaryKey"`
	CreatedAt string
	Content   string
	Done      bool `gorm:"default:false"`
}

type Row struct {
	Id        int
	CreatedAt string
	Done      string
	Content   string
}

func formatCheckMark(done bool) string {
	if done {
		return "[X]"
	} else {
		return "[ ]"
	}
}

func createDirIfDoesNotExist(directoryName string) {
	_, err := os.Stat(directoryName)
	if os.IsNotExist(err) {
		os.Mkdir(directoryName, 0755)
	}
}

func getHelp() string {
	return "Godo is a simple TODO-tool\n" +
		"Usage:\n" +
		"    --new -n <contents>  add new TODO row to database\n" +
		"    --list -l            list all existing TODOs\n" +
		"    --done -x <id>       mark TODO as done\n" +
		"    --delete -d <id>     delete existing TODO\n\n"
}

func isValidCommand(command string) bool {
	switch command {
	case
		"--new",
		"-n",
		"--list",
		"-l",
		"--done",
		"-x",
		"--delete",
		"-d":
		return true
	}
	return false
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf(getHelp())
		return
	}

	command := os.Args[1]

	if !isValidCommand(command) {
		fmt.Printf("%s is not a valid argument\n", command)
		fmt.Printf(getHelp())
		return
	}

	args := os.Args[2:]

	currentUser, err := user.Current()

	if err != nil {
		log.Fatal(err)
	}

	homeDir := currentUser.HomeDir

	directoryName := homeDir + "/.TODO"
	createDirIfDoesNotExist(directoryName)

	databasePath := directoryName + "/todos.db"

	db, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})

	if err != nil {
		log.Fatal(err)
	}

	db.AutoMigrate(&Todo{})

	var todos []Todo

	if command == "--new" || command == "-n" {
		if len(args) < 1 {
			fmt.Println("Gimme content for the TODO")
			return
		}
		new_todo := strings.Join(args, " ")
		db.Create(&Todo{CreatedAt: time.Now().Local().Format(time.Stamp), Content: new_todo})
		return
	}

	if command == "--list" || command == "-l" {
		db.Find(&todos)

		var doneTable []*Row
		var unDoneTable []*Row

		for _, t := range todos {
			if t.Done {
				doneTable = append(doneTable, &Row{t.Id, t.CreatedAt, formatCheckMark(t.Done), t.Content})
			} else {
				unDoneTable = append(unDoneTable, &Row{t.Id, t.CreatedAt, formatCheckMark(t.Done), t.Content})
			}
		}

		for _, t := range unDoneTable {
			fmt.Printf("%d  %s  %s  %s\n", t.Id, t.CreatedAt, t.Done, t.Content)
		}
		fmt.Println("")
		for _, t := range doneTable {
			fmt.Printf("%d  %s  %s  %s\n", t.Id, t.CreatedAt, t.Done, t.Content)
		}

		if err != nil {
			log.Fatal(err)
		}
		return
	}

	if command == "--done" || command == "-x" {
		if len(args) < 1 {
			fmt.Println("Gimme number of the TODO to mark done")
			return
		}
		id := args
		db.Model(&Todo{}).Where("Id = ?", id).Update("Done", true)
		return
	}

	if command == "--delete" || command == "-d" {
		if len(args) < 1 {
			fmt.Println("Gimme number of the TODO to remove")
			return
		}
		id := args
		db.Delete(&Todo{}, id)
		return
	}

	if command == "--help" {
		fmt.Printf(getHelp())
		return
	}
}
