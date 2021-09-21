package main

import (
    "fmt"
    "github.com/go-redis/redis"
    "log"
    "net/http"
    "os"
    "strconv"
    "strings"
)

// global variables
var defaultKey string = getEnv("HELLO_MSG", "world")
var rdb *redis.Client

func handler(w http.ResponseWriter, r *http.Request) {
    log.Print("Request received")

    key := defaultKey
    uriSegments := strings.Split(r.URL.Path, "/")
    if uriSegments[1] != "" {
        key = uriSegments[1]
    }

    counter, err := rdb.Incr(key).Result()
    if err != nil {
        fmt.Fprintf(w, "<h1>Hello, stranger!</h1>")
        log.Println("Error: " + err.Error())
    } else {
        fmt.Fprintf(w, "<h1>Hello %s %s!</h1>", key, strconv.FormatInt(counter, 10))
    }
}

func main() {
    log.Print("Server starting...")
    address := getEnv("REDIS_ADDRESS", "localhost:6379")
    password := getEnv("REDIS_PASSWORD", "")
    db, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
    rdb = redis.NewClient(&redis.Options{
        Addr:	  address,
        Password: password,
        DB:		  db,
    })
    _, err := rdb.Ping().Result()
    if err != nil {
        panic(err)
    }
    log.Print("Connected to Redis: " + address)
    log.Print("Server started.")

    mux := http.NewServeMux()
    mux.HandleFunc("/", handler)
    mux.HandleFunc("/*", handler)
    log.Fatal(http.ListenAndServe(":8080", mux))
}

func getEnv(key, fallback string) string {
    if value, ok := os.LookupEnv(key); ok {
        return value
    }
    return fallback
}
