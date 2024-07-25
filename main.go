package main

import (
	"fmt"
	"log"
	"net/http"
	"rest/dbx"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type taskDb struct {
	svc *dbx.Service
}

func NewDbServer() *taskDb {
	service := dbx.New()
	return &taskDb{svc: service}
}

func (td *taskDb) getTaskHandler(c *gin.Context) {
	log.Printf("handling get task at %s %s\n", c.Request.Method, c.Request.URL.Path)
	id, err := strconv.Atoi(c.Params.ByName("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "invalid id")
		return
	}

	task, err := td.svc.GetTask(id)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}

	c.JSON(http.StatusOK, task)
}

func (td *taskDb) getAllTasksHandler(c *gin.Context) {
	log.Printf("handling get all task at %s %s\n", c.Request.Method, c.Request.URL.Path)

	allTasks := td.svc.GetAllTasks()
	c.JSON(http.StatusOK, allTasks)
}

func (td *taskDb) createTaskHandler(c *gin.Context) {
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

	id := td.svc.CreateTask(rt.Text, rt.Tags, rt.Due)
	c.JSON(http.StatusOK, gin.H{"Id": id})
}

func (td *taskDb) deleteAllTasksHandler(c *gin.Context) {
	log.Printf("handling delete all task at %s %s\n", c.Request.Method, c.Request.URL.Path)

	err := td.svc.DeleteAllTasks()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}
}

func (td *taskDb) deleteTaskHandler(c *gin.Context) {
	log.Printf("handling delete task at %s %s\n", c.Request.Method, c.Request.URL.Path)
	id, err := strconv.Atoi(c.Params.ByName("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "invalid id")
		return
	}

	err = td.svc.DeleteTask(id)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}
}

func (td *taskDb) tagHandler(c *gin.Context) {
	log.Printf("handling tags at %s %s\n", c.Request.Method, c.Request.URL.Path)

	tag := c.Params.ByName("tag")
	tasks := td.svc.GetTasksByTag(tag)
	c.JSON(http.StatusOK, tasks)
}

func (td *taskDb) dueHandler(c *gin.Context) {
	log.Printf("handling due tasks at %s %s\n", c.Request.Method, c.Request.URL.Path)

	year, errYear := strconv.Atoi(c.Params.ByName("year"))
	day, errDay := strconv.Atoi(c.Params.ByName("day"))
	month, errMonth := strconv.Atoi(c.Params.ByName("month"))
	if errYear != nil || errMonth != nil || errDay != nil || month < int(time.January) || month > int(time.December) {
		c.String(http.StatusBadRequest, fmt.Sprintf("expect /due/<year>/<month>/<day> got %v", c.Request.URL.Path))
	}
	tasks := td.svc.GetTasksByDueDate(year, time.Month(month), day)
	c.JSON(http.StatusOK, tasks)
}

func main() {
	router := gin.Default()
	server := NewDbServer()

	router.POST("/task/", server.createTaskHandler)
	router.GET("/task/:id", server.getTaskHandler)
	router.GET("/task/", server.getAllTasksHandler)
	router.DELETE("/task/", server.deleteAllTasksHandler)
	router.DELETE("/task/:id", server.deleteTaskHandler)
	router.GET("/tag/:tag", server.tagHandler)
	router.GET("/due/:year/:month/:day", server.dueHandler)

	router.Run()
}
