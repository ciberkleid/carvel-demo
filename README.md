# carvel-demo

Demo app that returns "Hello world!" and a counter of requests.

Message "world" can be configured:
- at startup using env var HELLO_MSG
- dynamically per request (e.g. http :80/sunshine)
    
Counter tracks requests per message (_world_, _sunshine_, etc) and is stored in Redis.

## To run locally:

#### Start redis
```shell
docker run --name hello-redis -p 6379:6379 redis
```

#### Start app
```shell
go run app.go
```
OR
```shell
HELLO_MSG=friend go run app.go
```

#### Send requests:
Use a browser or separate terminal window:
```shell
curl localhost:80            # returns world (or value of HELLO_MSG) and counter
curl localhost:80/sunshine   # returns sunshine and counter
```

Sample response:
`<h1>Hello sunshine 11!</h1>`

#### Stop:
Stop app with `<Ctrl+C>`

Stop redis with:
```shell
docker stop hello-redis
```