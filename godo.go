package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
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
	Tags      string
}

func (todo Todo) generateContents() string {
	return "DONE: " + formatCheckMark(todo.Done) + "; TAGS: " + todo.Tags + "\n" + todo.Content
}

func (todo Todo) printRowString(commands CommandMap) {
	firstLineOfContent := strings.Split(todo.Content, "\n")[0]

	if len(firstLineOfContent) > 80 {
		firstLineOfContent = firstLineOfContent[:80] + "..."
	}

	checkMark := formatCheckMark(todo.Done)
	tags := formatTags(todo.Tags)

	_, ok := checkIfKeyExists(commands, "--created", "-c")

	if ok {
		fmt.Printf("%d  %s  %s  %s   |   %s\n", todo.Id, todo.CreatedAt, checkMark, firstLineOfContent, tags)
	} else {
		fmt.Printf("%d  %s  %s   |   %s\n", todo.Id, checkMark, firstLineOfContent, tags)
	}
}

type CommandMap map[string][]string

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
		"    --edit -e <id>       edit existing TODO\n" +
		"    --list -l            list all existing TODOs\n" +
		"    --done -x <id>       mark TODO as done\n" +
		"    --delete -d <id>     delete existing TODO\n\n"
}

func isValidCommand(command string) bool {
	switch command {
	case
		"--created",
		"--delete",
		"--done",
		"--edit",
		"--list",
		"--new",
		"--tag",
		"--view",
		"-c",
		"-d",
		"-e",
		"-h",
		"-l",
		"-n",
		"-t",
		"-v",
		"-x":
		return true
	}
	return false
}

func editInNvim(filename string) {
	cmd := exec.Command("nvim", filename)
	// cmd needs the stdin etc to function correctly
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if err != nil {
		log.Fatal(err)
	}
}

// TODO: maybe convert this to using Command struct type? Would it make this simpler?
// also maybe use an interface for next etc in the list?
func filterArguments(args []string) CommandMap {
	var validCommandIndeces []int

	for i, arg := range args {
		if isValidCommand(arg) {
			validCommandIndeces = append(validCommandIndeces, i)
		}
	}

	var commandArgsMap = make(CommandMap)

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

func checkIfKeyExists(commands CommandMap, commandName string, shortCommandName string) ([]string, bool) {
	args, ok := commands[commandName]

	if ok {
		return args, ok
	}

	args, ok = commands[shortCommandName]
	return args, ok
}

func formatTags(tags string) string {
	if len(tags) == 0 {
		return "-"
	}
	return tags
}

func createNewTmpFile(todo Todo, filepath string) error {
	fmt.Println(filepath)
	return ioutil.WriteFile(filepath, []byte(todo.generateContents()), 0755)
}

func parseFile(fileContent []byte) (string, string, bool) {
	var tags string

	splitByLines := strings.Split(string(fileContent), "\n")
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

func main() {
	if len(os.Args) < 2 {
		fmt.Printf(getHelp())
		return
	}

	// access commands `commands["--edit"]`
	commands := filterArguments(os.Args)

	if len(commands) == 0 {
		fmt.Printf("Did not reveive any valid arguments\n")
		fmt.Printf(getHelp())
		return
	}

	args := os.Args[2:]

	currentUser, err := user.Current()

	tmpDir := os.TempDir()
	TmpFileName := "/godofile.txt"
	TmpFilePath := tmpDir + TmpFileName

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

	var todo Todo

	db.AutoMigrate(&todo)

	args, ok := checkIfKeyExists(commands, "--new", "-n")

	// --new -n
	if ok {
		_, editok := checkIfKeyExists(commands, "--edit", "-e")

		if len(args) < 1 && !editok {
			fmt.Println("Gimme content for the TODO")
			return
		}

		// default done to false, but possible to have it done during
		// if edited on creation
		done := false

		tagArgs, ok := checkIfKeyExists(commands, "--tags", "-t")
		var tags string

		if !ok || len(tagArgs) == 0 {
			tags = ""
		} else {
			tags = strings.Join(tagArgs, ", ")
		}

		content := strings.Join(args, " ")

		todo = Todo{
			Content: content,
			Tags:    tags,
			Done:    false,
		}

		if editok {
			err := createNewTmpFile(todo, TmpFilePath)

			if err != nil {
				log.Fatalf("%+v\n", err)
			}

			editInNvim(TmpFilePath)
			fileContent, err := ioutil.ReadFile(TmpFilePath)

			if err != nil {
				log.Fatalf("%+v\n", err)
			}

			content, tags, done = parseFile(fileContent)
		}

		db.Create(&Todo{
			CreatedAt: time.Now().Local().Format(time.Stamp),
			Content:   content,
			Tags:      tags,
			Done:      done,
		})
		return
	}

	args, ok = checkIfKeyExists(commands, "--list", "-l")

	// --list -l
	if ok {
		var dones []Todo
		var undones []Todo

		args, ok := checkIfKeyExists(commands, "--tag", "-t")
		if ok {
			tags := args
			db.Where("Tags = ? AND Done = ?", tags, true).Find(&dones)
			db.Where("Tags = ? AND Done = ?", tags, false).Find(&undones)
		} else {
			db.Where("Done = ?", true).Find(&dones)
			db.Where("Done = ?", false).Find(&undones)
		}

		if len(dones) == 0 && len(undones) == 0 {
			return
		}

		for _, t := range undones {
			t.printRowString(commands)
		}

		if len(undones) != 0 {
			fmt.Println("")
		}

		for _, t := range dones {
			t.printRowString(commands)
		}

		if err != nil {
			log.Fatal(err)
		}

		return
	}

	args, ok = checkIfKeyExists(commands, "--done", "-x")

	// --done -x
	if ok {
		if len(args) < 1 {
			fmt.Println("Gimme number of the TODO to mark done")
			return
		}

		id := args
		db.Model(todo).Where("Id = ?", id).Update("Done", true)
		return
	}

	args, ok = checkIfKeyExists(commands, "--edit", "-e")

	// --edit -e
	if ok {
		if len(args) < 1 {
			fmt.Println("Gimme number of the TODO to edit")
			return
		}

		id := args[0]
		db.First(&todo, id)

		err := createNewTmpFile(todo, TmpFilePath)

		if err != nil {
			log.Fatal(err)
		}

		editInNvim(TmpFilePath)
		fileContent, err := ioutil.ReadFile(TmpFilePath)

		if err != nil {
			log.Fatal(err)
		}

		content, tags, done := parseFile(fileContent)

		db.Model(&todo).Where("Id = ?", id).Updates(map[string]interface{}{"Content": content, "Tags": tags, "Done": done})
		return
	}

	args, ok = checkIfKeyExists(commands, "--delete", "-d")

	// --delete -d
	if ok {
		if len(args) < 1 {
			fmt.Println("Gimme number of the TODO to remove")
			return
		}

		id := args
		db.Delete(&todo, id)
		return
	}

	args, ok = checkIfKeyExists(commands, "--help", "-h")

	// --help
	if ok {
		fmt.Printf(getHelp())
		return
	}

	args, ok = checkIfKeyExists(commands, "--view", "-v")

	// --view
	if ok {
		if len(args) < 1 {
			fmt.Println("Gimme id to view")
		}

		id := args[0]
		db.First(&todo, id)
		fmt.Printf("TAGS: %s\n%s\n", todo.Tags, todo.Content)
		return
	}
}
