package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func formatCheckMark(done bool) string {
	if done {
		return "[X]"
	} else {
		return "[ ]"
	}
}

func unFormatCheckMark(checkMark string) bool {
	if checkMark == "[X]" || checkMark == "[x]" {
		return true
	}
	if checkMark == "[ ]" || checkMark == "[]" {
		return false
	}
	log.Fatalf("syntax error, no checkbox found\n")
	return false
}

func getHelp() string {
	return "Godo is a simple TODO-tool\n" +
		"Usage:\n" +
		"    new n <contents>  add new TODO row to database\n" +
		"    edit e <id>       edit existing TODO\n" +
		"    list l            list all existing TODOs\n" +
		"    done x <id>       mark TODO as done\n" +
		"    delete d <id>     delete existing TODO\n" +
		"    view v <id>       view single TODO\n" +
		"    --tags -t         filter by tags\n" +
		"    --edit -e         use as argument with new command\n"
}

type CommandMap map[string][]string

func (commands CommandMap) getKeyArgs(commandName string, shortCommandName string) ([]string, bool) {
	args, ok := commands[commandName]

	if ok {
		return args, ok
	}

	args, ok = commands[shortCommandName]
	return args, ok
}

func isValidOption(command string) bool {
	switch command {
	case
		"--created",
		"--tag",
		"-c",
		"-h",
		"-t":
		return true
	}
	return false
}

func isValidCommand(command string) bool {
	switch command {
	case
		"delete",
		"done",
		"edit",
		"list",
		"new",
		"view",
		"d",
		"e",
		"l",
		"n",
		"v",
		"x":
		return true
	}
	return false
}

// TODO: maybe convert this to using Command struct type? Would it make this simpler?
// also maybe use an interface for next etc in the list?
func filterArguments(args []string) CommandMap {
	var validCommandIndeces []int
	var commandArgsMap = make(CommandMap)

	if isValidCommand(args[1]) {
		validCommandIndeces = append(validCommandIndeces, 1)
	} else {
		return commandArgsMap
	}

	for i, arg := range args {
		if isValidOption(arg) {
			validCommandIndeces = append(validCommandIndeces, i)
		}
	}

	if len(validCommandIndeces) == 1 {
		command := args[validCommandIndeces[0]]
		restOfArgs := args[validCommandIndeces[0]+1:]
		commandArgsMap[command] = restOfArgs
		return commandArgsMap
	}

	for i := range validCommandIndeces {
		var restOfArgs []string
		command := args[validCommandIndeces[i]]

		if i < len(validCommandIndeces)-1 {
			restOfArgs = args[validCommandIndeces[i]+1 : validCommandIndeces[i+1]]
		} else {
			restOfArgs = args[validCommandIndeces[i]+1:]
		}

		commandArgsMap[command] = restOfArgs
	}
	return commandArgsMap
}

func formatTags(tags string) string {
	if len(tags) == 0 {
		return "-"
	}
	return tags
}

func findMaxIdLength(todos []Todo) int {
	maxId := 0
	for _, r := range todos {
		if r.Id > maxId {
			maxId = r.Id
		}
	}
	return len(strconv.Itoa(maxId))
}

func slicesEqual(s1, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}

	for i, v := range s1 {
		if v != s2[i] {
			return false
		}
	}
	return true
}

func isExistingInDB(db *gorm.DB, inputIds []string) bool {
	var dbTodos []Todo

	existingIds := make(map[int]bool)
	db.Select("id").Find(&dbTodos)

	for _, v := range dbTodos {
		existingIds[v.Id] = true
	}

	var matchingIds []string
	for _, id := range inputIds {
		id_i, _ := strconv.Atoi(id)
		if _, ok := existingIds[id_i]; ok {
			matchingIds = append(matchingIds, strconv.Itoa(id_i))
		}
	}

	return slicesEqual(matchingIds, inputIds)

}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf(getHelp())
		return
	}

	// access commands `commands["--edit"]`
	commands := filterArguments(os.Args)

	if len(commands) == 0 {
		fmt.Printf("Did not receive any valid arguments\n")
		fmt.Printf(getHelp())
		return
	}

	currentUser, err := user.Current()

	tmpDir := os.TempDir()
	TmpFileName := "/godofile.txt"
	TmpFilePath := tmpDir + TmpFileName

	TmpFile := File{filepath: TmpFilePath}

	if err != nil {
		log.Fatal(err)
	}

	homeDir := currentUser.HomeDir
	directoryName := homeDir + "/.TODO"
	filepath := directoryName + "/todos.db"

	dbFile := File{directoryName: directoryName, filepath: filepath}
	dbFile.createDirIfDoesNotExist()

	db, err := gorm.Open(sqlite.Open(dbFile.filepath), &gorm.Config{})

	if err != nil {
		log.Fatal(err)
	}

	var todo Todo

	db.AutoMigrate(&todo)

	args, ok := commands.getKeyArgs("new", "n")

	// --new -n
	if ok {
		_, editok := commands.getKeyArgs("--edit", "-e")

		if len(args) < 1 && !editok {
			fmt.Println("Gimme content for the TODO")
			return
		}

		// default done to false, but possible to have it done during
		// if edited on creation
		done := false

		tagArgs, _ := commands.getKeyArgs("--tag", "-t")
		tags := strings.Join(tagArgs, ", ")

		content := strings.Join(args, " ")

		todo = Todo{
			Content: content,
			Tags:    tags,
			Done:    done,
		}

		if editok {
			err := todo.createNewTmpFile(TmpFile.filepath)

			if err != nil {
				log.Fatalf("%+v\n", err)
			}

			TmpFile.editInNvim()
			TmpFile.content, err = ioutil.ReadFile(TmpFilePath)

			if err != nil {
				log.Fatalf("%+v\n", err)
			}

			todo.Content, todo.Tags, todo.Done = TmpFile.parseFile()

			if len(content) == 0 {
				fmt.Println("Empty content, database not updated")
				return
			}
		}

		todo.CreatedAt = time.Now().Local().Format(time.Stamp)

		db.Create(&todo)
		return
	}

	args, ok = commands.getKeyArgs("list", "l")

	// --list -l
	if ok {
		var dones []Todo
		var undones []Todo

		args, ok := commands.getKeyArgs("--tag", "-t")
		tags := "%" + strings.Join(args, ", ") + "%" // format to "fuzzy" find format in database tags-string

		if ok {
			db.Where("Tags LIKE ? AND Done = ?", tags, true).Find(&dones)
			db.Where("Tags LIKE ? AND Done = ?", tags, false).Find(&undones)
		} else {
			db.Where("Done = ?", true).Find(&dones)
			db.Where("Done = ?", false).Find(&undones)
		}

		if len(dones) == 0 && len(undones) == 0 {
			fmt.Println("No todos found")
			return
		}

		_, Cok := commands.getKeyArgs("--created", "-c")

		todos := append(dones, undones...)

		maxLen := findMaxIdLength(todos)

		for _, t := range undones {
			t.printRowString(commands, Cok, maxLen)
		}

		if len(undones) != 0 {
			fmt.Println("")
		}

		for _, t := range dones {
			t.printRowString(commands, Cok, maxLen)
		}

		if err != nil {
			log.Fatal(err)
		}

		return
	}

	args, ok = commands.getKeyArgs("done", "x")

	// --done -x
	if ok {
		if len(args) < 1 {
			fmt.Println("Gimme ID of the TODO to mark done")
			return
		}

		if !isExistingInDB(db, args) {
			fmt.Println("ID does not exist")
			return
		}

		db.Table("todos").Where("Id IN ?", args).Updates(map[string]interface{}{"Done": true})
		return
	}

	args, ok = commands.getKeyArgs("edit", "e")

	// --edit -e
	if ok {
		if len(args) < 1 {
			fmt.Println("Gimme number of the TODO to edit")
			return
		}

		if !isExistingInDB(db, args) {
			fmt.Println("ID does not exist")
			return
		}

		id := args[0]
		db.First(&todo, id)

		err := todo.createNewTmpFile(TmpFile.filepath)

		if err != nil {
			log.Fatal(err)
		}

		TmpFile.editInNvim()
		TmpFile.content, err = ioutil.ReadFile(TmpFile.filepath)

		if err != nil {
			log.Fatal(err)
		}

		content, tags, done := TmpFile.parseFile()

		if len(content) == 0 {
			fmt.Println("Empty content, database not updated")
			return
		}

		db.Model(&todo).Where("Id = ?", id).Updates(map[string]interface{}{"Content": content, "Tags": tags, "Done": done})
		return
	}

	args, ok = commands.getKeyArgs("delete", "d")

	// --delete -d
	if ok {
		if len(args) < 1 {
			fmt.Println("Gimme number of the TODO to remove")
			return
		}

		if !isExistingInDB(db, args) {
			fmt.Println("ID does not exist")
			return
		}

		id := args
		// TODO: add ability to delete multiple todos at a time
		db.Delete(&todo, id)
		return
	}

	args, ok = commands.getKeyArgs("--help", "-h")

	// --help
	if ok {
		fmt.Printf(getHelp())
		return
	}

	args, ok = commands.getKeyArgs("view", "v")

	// --view
	if ok {
		if len(args) < 1 {
			fmt.Println("Gimme id to view")
		}

		if !isExistingInDB(db, args) {
			fmt.Println("ID does not exist")
			return
		}

		id := args[0]
		db.First(&todo, id)
		fmt.Printf("TAGS: %s\n%s\n", todo.Tags, todo.Content)
		return
	}
}
