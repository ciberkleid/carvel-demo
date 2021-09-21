# hello-app

The app used in this tutorial is a simple application that returns "Hello world!" and a counter of requests.


The message "world" can be configured in two ways:
- at startup using environment variable HELLO_MSG
- dynamically using the request path (e.g. curl localhost:8080/sunshine)

The counter tracks requests per message (_world_, _sunshine_, etc) and is stored in Redis.
A sample response looks like this:
```
<h1>Hello sunshine 6!</h1>
```
You can view the source code here: [src/hello-app.go](src/hello-app.go).

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

#### Stop
Stop app with `<Ctrl+C>`

Stop redis with:
```shell
docker stop hello-redis
```