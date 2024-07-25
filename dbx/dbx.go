package dbx

import (
	"database/sql"
	"log"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type Task struct {
	Id   int       `json:"id"`
	Text string    `json:"text"`
	Tags []string  `json:"tags"`
	Due  time.Time `json:"due"`
}

type Service struct {
	sync.Mutex

	db *sql.DB
}

func New() *Service {
	// os.Remove("./todo.db")
	db, err := sql.Open("sqlite3", "./todo.db")
	checkErr(err)

	createTagsTable := `create table if not exists tags(id integer primary key autoincrement, tag text not null unique);`
	_, err = db.Exec(createTagsTable)
	if err != nil {
		log.Fatal(err)
	}

	createTasksTable := `create table if not exists todo(Id integer not null primary key autoincrement, Text text not null, Due datetime);`
	_, err = db.Exec(createTasksTable)
	if err != nil {
		log.Fatal(err)
	}

	createTaskTagsTable := `create table if not exists taskTags(todoId integer, tagId integer, foreign key(todoId) references todo(Id), foreign key(tagId) references tags(id), primary key(todoId, tagId));`
	_, err = db.Exec(createTaskTagsTable)
	if err != nil {
		log.Fatal(err)
	}
	return &Service{
		db: db,
	}
}

func (svc *Service) scanTasks(rows *sql.Rows) ([]Task, error) {
	tasks := make([]Task, 0)

	for rows.Next() {
		var task Task
		err := rows.Scan(&task.Id, &task.Text, &task.Due)
		if err != nil {
			return []Task{}, err
		}
		task.Tags, err = svc.getTagsByTask(task.Id)
		if err != nil {
			return []Task{}, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (svc *Service) getTagsByTask(id int) ([]string, error) {
	rows, err := svc.db.Query("select tagId from taskTags where todoId = ?;", id)
	if err != nil {
		return nil, err
	}
	tagIds := make([]int, 0)
	defer rows.Close()
	for rows.Next() {
		var id int
		err := rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		tagIds = append(tagIds, id)
	}
	tags := make([]string, 0)
	for _, tagId := range tagIds {
		var tag string
		err := svc.db.QueryRow("select tag from tags where id = ?;", tagId).Scan(&tag)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func (svc *Service) GetTask(id int) (Task, error) {
	svc.Lock()
	defer svc.Unlock()
	var task Task

	err := svc.db.QueryRow("select * from todo where id = ?;", id).Scan(&task.Id, &task.Text, &task.Due)
	if err != nil {
		return Task{}, err
	}

	task.Tags, err = svc.getTagsByTask(id)
	checkErr(err)

	log.Printf("task: %v", task)
	return task, nil
}

func (svc *Service) GetAllTasks() []Task {
	svc.Lock()
	defer svc.Unlock()

	stmt, err := svc.db.Prepare("select * from todo;")
	checkErr(err)
	defer stmt.Close()
	rows, err := stmt.Query()
	checkErr(err)
	defer rows.Close()
	tasks, err := svc.scanTasks(rows)
	checkErr(err)
	return tasks
}

func (svc *Service) getOrInsertTag(tag string) (int, error) {
	var tagId int64
	err := svc.db.QueryRow(`select id from tags where tag = ?`, tag).Scan(&tagId)
	if err == sql.ErrNoRows {
		stmt, err := svc.db.Prepare(`insert into tags(tag) values(?)`)
		checkErr(err)
		defer stmt.Close()
		res, err := stmt.Exec(tag)
		checkErr(err)
		tagId, err = res.LastInsertId()
		checkErr(err)
	}
	return int(tagId), nil
}

func (svc *Service) CreateTask(text string, tags []string, due time.Time) int {
	svc.Lock()
	defer svc.Unlock()
	stmt, err := svc.db.Prepare(`
		INSERT INTO todo(Text, Due) VALUES(?, datetime(?));
		`)
	checkErr(err)
	defer stmt.Close()

	res, err := stmt.Exec(text, due)
	if err != nil {
		checkErr(err)
	}

	todoId, err := res.LastInsertId()
	checkErr(err)
	tagIds := make([]int, len(tags))
	for i, tag := range tags {
		id, err := svc.getOrInsertTag(tag)
		checkErr(err)
		tagIds[i] = id
	}

	taskTagsStmt, err := svc.db.Prepare(`insert into taskTags(todoId, tagId) values(?, ?)`)
	checkErr(err)
	for _, tagId := range tagIds {
		_, err := taskTagsStmt.Exec(todoId, tagId)
		checkErr(err)
	}

	log.Printf("Id = %d", todoId)
	return int(todoId)
}

func (svc *Service) DeleteTask(id int) error {
	svc.Lock()
	defer svc.Unlock()
	tx, err := svc.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare("delete from todo where id = (?);")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(id)
	if err != nil {
		return err
	}
	taskTagStmt, err := tx.Prepare("delete from taskTags where todoId = (?);")
	if err != nil {
		return err
	}
	defer taskTagStmt.Close()
	_, err = taskTagStmt.Exec(id)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (svc *Service) DeleteAllTasks() error {
	svc.Lock()
	defer svc.Unlock()
	tx, err := svc.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`delete from todo; `)
	if err != nil {
		return err
	}
	_, err = tx.Exec("delete from tags;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("delete from taskTags;")
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (svc *Service) GetTasksByDueDate(year int, month time.Month, day int) []Task {
	svc.Lock()
	defer svc.Unlock()
	stmt, err := svc.db.Prepare("select * from todo where date(Due) = ?")
	checkErr(err)
	defer stmt.Close()
	dueDate := time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	rows, err := stmt.Query(dueDate)
	checkErr(err)
	defer rows.Close()
	tasks, err := svc.scanTasks(rows)
	checkErr(err)
	return tasks
}

func (svc *Service) GetTasksByTag(tag string) []Task {
	svc.Lock()
	defer svc.Unlock()
	stmt, err := svc.db.Prepare("select id from tags where tag = (?);")
	checkErr(err)
	defer stmt.Close()
	var tagId int
	err = stmt.QueryRow(tag).Scan(&tagId)
	checkErr(err)
	rows, err := svc.db.Query(`select todoId from taskTags where tagId = (?);`, tagId)
	checkErr(err)
	defer rows.Close()
	var todoIds []int = []int{}
	for rows.Next() {
		var todoId int
		rows.Scan(&todoId)
		todoIds = append(todoIds, todoId)
	}
	var tasks []Task
	for _, todoId := range todoIds {
		var task Task
		err := svc.db.QueryRow("select * from todo where id = ?;", todoId).Scan(&task.Id, &task.Text, &task.Due)
		checkErr(err)
		tags, err := svc.getTagsByTask(todoId)
		task.Tags = tags
		checkErr(err)
		tasks = append(tasks, task)
	}
	return tasks
}
