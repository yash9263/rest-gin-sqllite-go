# REST server in go using gin and sqllite

it uses sqlite db to store the tasks.

run server

```bash
go run main.go
```

routes

```bash
POST "/task/" 
post request body
{
    "text": "hello, world",
    "tags": ["urgent"],
    "due": "2024-07-25T17:58:51.658Z"
}
GET "/task/:id"
/task/1
GET "/task/"
DELETE "/task/"
DELETE "/task/:id"
/task/1
GET "/tag/:tag"
/task/urgent
GET "/due/:year/:month/:day"
/task/2024/7/25
```
