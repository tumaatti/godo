package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strconv"
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

type File struct {
	filepath string
	content  []byte
}

func (file File) createDirIfDoesNotExist() {
	_, err := os.Stat(file.filepath)

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

func getHelp() string {
	return "Godo is a simple TODO-tool\n" +
		"Usage:\n" +
		"    --new -n <contents>  add new TODO row to database\n" +
		"    --edit -e <id>       edit existing TODO\n" +
		"    --list -l            list all existing TODOs\n" +
		"    --done -x <id>       mark TODO as done\n" +
		"    --delete -d <id>     delete existing TODO\n" +
		"    --view -v <id>       view single TODO\n" +
		"    --tags -t            add and sort by tags\n"
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

	dbFile := File{filepath: filepath}
	dbFile.createDirIfDoesNotExist()

	db, err := gorm.Open(sqlite.Open(dbFile.filepath), &gorm.Config{})

	if err != nil {
		log.Fatal(err)
	}

	var todo Todo

	db.AutoMigrate(&todo)

	args, ok := commands.getKeyArgs("--new", "-n")

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

		tagArgs, ok := commands.getKeyArgs("--tags", "-t")
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
			err := todo.createNewTmpFile(TmpFile.filepath)

			if err != nil {
				log.Fatalf("%+v\n", err)
			}

			TmpFile.editInNvim()
			TmpFile.content, err = ioutil.ReadFile(TmpFilePath)

			if err != nil {
				log.Fatalf("%+v\n", err)
			}

			content, tags, done = TmpFile.parseFile()
		}

		db.Create(&Todo{
			CreatedAt: time.Now().Local().Format(time.Stamp),
			Content:   content,
			Tags:      tags,
			Done:      done,
		})
		return
	}

	args, ok = commands.getKeyArgs("--list", "-l")

	// --list -l
	if ok {
		var dones []Todo
		var undones []Todo

		args, ok := commands.getKeyArgs("--tag", "-t")
		if ok {
			tags := args
			db.Where("Tags = ? AND Done = ?", tags, true).Find(&dones)
			db.Where("Tags = ? AND Done = ?", tags, false).Find(&undones)
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

	args, ok = commands.getKeyArgs("--done", "-x")

	// --done -x
	if ok {
		if len(args) < 1 {
			fmt.Println("Gimme number of the TODO to mark done")
			return
		}

		id := args

		db.Table("todos").Where("Id IN ?", id).Updates(map[string]interface{}{"Done": true})
		return
	}

	args, ok = commands.getKeyArgs("--edit", "-e")

	// --edit -e
	if ok {
		if len(args) < 1 {
			fmt.Println("Gimme number of the TODO to edit")
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

		db.Model(&todo).Where("Id = ?", id).Updates(map[string]interface{}{"Content": content, "Tags": tags, "Done": done})
		return
	}

	args, ok = commands.getKeyArgs("--delete", "-d")

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

	args, ok = commands.getKeyArgs("--help", "-h")

	// --help
	if ok {
		fmt.Printf(getHelp())
		return
	}

	args, ok = commands.getKeyArgs("--view", "-v")

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
