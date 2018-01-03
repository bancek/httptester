# httptester

Go HTTP testing DSL

## Example

```go
GET("/").Do().Status(409).Contains("Already exists")

result := &SearchResult{}
GET("/search").Q("query", "test").Do().Status(200).JSON(&result)

newArticle := &Article{}
POST("/articles/").JSON(&Article{
  Title: "Test",
  Content: "Lorem ipsum",
}).Do().Status(201).JSON(&newArticle)
```

See `request_test.go` for more info.
