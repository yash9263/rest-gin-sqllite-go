package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"rest/taskstore"

	"github.com/gin-gonic/gin"
)

type taskServer struct {
	store *taskstore.TaskStore
}

func NewTaskServer() *taskServer {
	store := taskstore.New()
	return &taskServer{store: store}
}

func renderJson(w http.ResponseWriter, v interface{}) {
	js, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (ts *taskServer) getTaskHandler(c *gin.Context) {
	log.Printf("handling get task at %s %s\n", c.Request.Method, c.Request.URL.Path)
	id, err := strconv.Atoi(c.Params.ByName("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "invalid id")
		return
	}

	task, err := ts.store.GetTask(id)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}

	c.JSON(http.StatusOK, task)
}

func (ts *taskServer) getAllTasksHandler(c *gin.Context) {
	log.Printf("handling get all task at %s %s\n", c.Request.Method, c.Request.URL.Path)

	allTasks := ts.store.GetAllTasks()
	c.JSON(http.StatusOK, allTasks)
}

func (ts *taskServer) createTaskHandler(c *gin.Context) {
	log.Printf("handling create task at %s %s\n", c.Request.Method, c.Request.URL.Path)

	type RequestTask struct {
		Text string    `json:"text"`
		Tags []string  `json:"tags"`
		Due  time.Time `json:"due"`
	}

	type ResponseId struct {
		Id int `json:"id"`
	}

	var rt RequestTask
	if err := c.ShouldBindJSON(&rt); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	id := ts.store.CreateTask(rt.Text, rt.Tags, rt.Due)
	c.JSON(http.StatusOK, gin.H{"Id": id})
}

func (ts *taskServer) deleteAllTasksHandler(c *gin.Context) {
	log.Printf("handling delete all task at %s %s\n", c.Request.Method, c.Request.URL.Path)

	err := ts.store.DeleteAllTasks()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}
}

func (ts *taskServer) deleteTaskHandler(c *gin.Context) {
	log.Printf("handling delete task at %s %s\n", c.Request.Method, c.Request.URL.Path)

	id, err := strconv.Atoi(c.Params.ByName("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "invalid id")
		return
	}

	err = ts.store.DeleteTask(id)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}
}

func (ts *taskServer) tagHandler(c *gin.Context) {
	log.Printf("handling tags at %s %s\n", c.Request.Method, c.Request.URL.Path)

	tag := c.Params.ByName("tag")
	tasks := ts.store.GetTasksByTag(tag)
	c.JSON(http.StatusOK, tasks)
}

func (ts *taskServer) dueHandler(c *gin.Context) {
	log.Printf("handling due tasks at %s %s\n", c.Request.Method, c.Request.URL.Path)

	year, errYear := strconv.Atoi(c.Params.ByName("year"))
	day, errDay := strconv.Atoi(c.Params.ByName("day"))
	month, errMonth := strconv.Atoi(c.Params.ByName("month"))
	if errYear != nil || errMonth != nil || errDay != nil || month < int(time.January) || month > int(time.December) {
		c.String(http.StatusBadRequest, fmt.Sprintf("expect /due/<year>/<month>/<day> got %v", c.Request.URL.Path))
	}
	tasks := ts.store.GetTasksByDueDate(year, time.Month(month), day)
	c.JSON(http.StatusOK, tasks)
}

func main() {
	router := gin.Default()
	server := NewTaskServer()

	router.POST("/task/", server.createTaskHandler)
	router.GET("/task/:id", server.getTaskHandler)
	router.GET("/task/", server.getAllTasksHandler)
	router.DELETE("/task/", server.deleteAllTasksHandler)
	router.DELETE("/task/:id", server.deleteTaskHandler)
	router.GET("/tag/:tag", server.tagHandler)
	router.GET("/due/:year/:month/:day", server.dueHandler)

	router.Run()
}
