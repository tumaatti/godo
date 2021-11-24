# GODO -- TODO in Golang lol

Simple TODO cli application. Creates TODOs in sqlite3 database to
`~/.TODO/todos.db`.

## Usage

Install `go install ./cmd/...`

```bash
godo ...
    --new -n <contents>  add new TODO row to database
    --edit -e <id>       edit existing TODO in neovim
    --list -l            list all existing TODOs
    --done -x <id...>       mark TODO as done
    --delete -d <id>     delete existing TODO
```
