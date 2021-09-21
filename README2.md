# hello-app

The app used in this tutorial is a simple application that returns "Hello world!" and a counter of requests.
You can view the source code here: [src/hello-app.go](src/hello-app.go).

The message "world" can be configured:
- at startup using env var HELLO_MSG
- dynamically per request (e.g. http :8080/sunshine)

The counter tracks requests per message (_world_, _sunshine_, etc) and is stored in Redis.

## Run locally

#### Start redis
```shell
docker run -d --rm --name hello-redis -p 6379:6379 redis
```

#### Start app
```shell
(cd src && go run hello-app.go)
```
or:
```shell
(cd src && HELLO_MSG=friend go run hello-app.go)
```

#### Send requests
Use a browser or separate terminal window:
```shell
curl localhost:8080            # returns world (or value of HELLO_MSG) and counter
curl localhost:8080/sunshine   # returns sunshine and counter
```

Sample response:
`<h1>Hello sunshine 11!</h1>`

#### Stop
Stop app with `<Ctrl+C>`

Stop redis with:
```shell
docker stop hello-redis
```

## Deploy to Kubernetes using Carvel

See [README.md](README.md)