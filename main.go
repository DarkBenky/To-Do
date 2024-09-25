package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Todo struct {
	ID        int
	Task      string
	Priority  int
	DueDate   time.Time
	Completed bool
}

type TodoGroup struct {
	Date  time.Time
	Todos []Todo
}

var db *sql.DB
var tmpl *template.Template

func init() {
	var err error
	db, err = sql.Open("sqlite3", "./todos.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS todos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task TEXT,
		priority INTEGER,
		due_date DATE,
		completed BOOLEAN DEFAULT 0
	)`)
	if err != nil {
		log.Fatal(err)
	}

	tmpl = template.Must(template.ParseFiles("index.html"))
}

func main() {
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/add", handleAdd)
	http.HandleFunc("/complete", handleComplete)
	http.HandleFunc("/uncompleted", handleUncompleted)
	fmt.Println("Server is running on http://localhost:5050")
	log.Fatal(http.ListenAndServe(":5050", nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	todoGroups, err := getTodoGroups()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, todoGroups)
}

func handleAdd(w http.ResponseWriter, r *http.Request) {
	task := r.FormValue("task")
	priority, _ := strconv.Atoi(r.FormValue("priority"))
	dueDate, _ := time.Parse("2006-01-02", r.FormValue("due_date"))

	_, err := db.Exec("INSERT INTO todos (task, priority, due_date) VALUES (?, ?, ?)", task, priority, dueDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	todoGroups, err := getTodoGroups()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "todo-list", todoGroups); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleComplete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.FormValue("id"))
	_, err := db.Exec("UPDATE todos SET completed = 1 WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	todoGroups, err := getTodoGroups()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "todo-list", todoGroups); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleUncompleted(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.FormValue("id"))
	_, err := db.Exec("UPDATE todos SET completed = 0 WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	todoGroups, err := getTodoGroups()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "todo-list", todoGroups); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getTodoGroups() ([]TodoGroup, error) {
	todos, err := getTodos()
	if err != nil {
		return nil, err
	}

	groups := make(map[time.Time][]Todo)
	for _, todo := range todos {
		date := todo.DueDate.Truncate(24 * time.Hour)
		groups[date] = append(groups[date], todo)
	}

	var todoGroups []TodoGroup
	for date, todos := range groups {
		todoGroups = append(todoGroups, TodoGroup{Date: date, Todos: todos})
	}

	sort.Slice(todoGroups, func(i, j int) bool {
		return todoGroups[i].Date.Before(todoGroups[j].Date)
	})

	return todoGroups, nil
}

func getTodos() ([]Todo, error) {
	rows, err := db.Query("SELECT id, task, priority, due_date, completed FROM todos ORDER BY due_date DES, priority ASC, completed ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var t Todo
		err := rows.Scan(&t.ID, &t.Task, &t.Priority, &t.DueDate, &t.Completed)
		if err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}
	return todos, nil
}