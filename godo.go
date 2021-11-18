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
		"    --edit -e <id>       edit existing TODO\n" +
		"    --list -l            list all existing TODOs\n" +
		"    --done -x <id>       mark TODO as done\n" +
		"    --delete -d <id>     delete existing TODO\n\n"
}

func isValidCommand(command string) bool {
	switch command {
	case
		"--delete",
		"--done",
		"--edit",
		"--list",
		"--new",
		"-d",
		"-e",
		"-h",
		"-l",
		"-n",
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

type CommandMap map[string][]string

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
	var todo Todo

	args, ok := checkIfKeyExists(commands, "--new", "-n")

	// --new -n
	if ok {
		if len(args) < 1 {
			fmt.Println("Gimme content for the TODO")

			return
		}

		new_todo := strings.Join(args, " ")
		db.Create(&Todo{CreatedAt: time.Now().Local().Format(time.Stamp), Content: new_todo})

		return
	}

	args, ok = checkIfKeyExists(commands, "--list", "-l")

	// --list -l
	if ok {
		db.Find(&todos)

		var doneTable []*Row
		var unDoneTable []*Row

		for _, t := range todos {
			firstLineOfContent := strings.Split(t.Content, "\n")[0]
			checkMark := formatCheckMark(t.Done)

			if t.Done {
				doneTable = append(doneTable, &Row{t.Id, t.CreatedAt, checkMark, firstLineOfContent})
			} else {
				unDoneTable = append(unDoneTable, &Row{t.Id, t.CreatedAt, checkMark, firstLineOfContent})
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

	args, ok = checkIfKeyExists(commands, "--done", "-x")

	// --done -x
	if ok {
		if len(args) < 1 {
			fmt.Println("Gimme number of the TODO to mark done")

			return
		}

		id := args
		db.Model(&Todo{}).Where("Id = ?", id).Update("Done", true)

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

		tmpDir := os.TempDir()
		filename := tmpDir + "/godofile.txt"

		err := ioutil.WriteFile(filename, []byte(todo.Content), 0755)

		if err != nil {
			log.Fatal(err)
		}

		editInNvim(filename)

		content, err := ioutil.ReadFile(filename)

		if err != nil {
			log.Fatal(err)
		}

		db.Model(&Todo{}).Where("Id = ?", id).Update("Content", content)

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
		db.Delete(&Todo{}, id)

		return
	}

	args, ok = checkIfKeyExists(commands, "--help", "-h")

	// --help
	if ok {
		fmt.Printf(getHelp())

		return
	}
}
