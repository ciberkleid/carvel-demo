# carvel-demo

Demo app that returns "Hello world!" and a counter of requests.

Message "world" can be configured:
- at startup using env var HELLO_MSG
- dynamically per request (e.g. http :80/sunshine)
    
Counter tracks requests per message (_world_, _sunshine_, etc) and is stored in Redis.

## Run locally:

#### Start redis
```shell
docker run -d --rm --name hello-redis -p 6379:6379 redis
```

#### Start app
```shell
(cd src && go run app.go)
```
OR
```shell
(cd src && HELLO_MSG=friend go run app.go)
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

## Deploy to Kubernetes using Carvel

#### Optional: Sync dependencies and lock versions

Redis is vendored into the application packaging.

Sync vendored files and generate lock file for vendor files:
> Note: Uncomment _--locked_ to sync using existing vendir.lock.yml file
```shell
if [[ ! -f config/vendir.lock.yml ]]; then \
  vendir sync --chdir config
else \
  vendir sync --chdir config --locked
fi
```

#### Generate YAML config

Generate YAML
> Note: Uses locked image versions.
> Delete lock files to use latest available images.
```shell
ytt -f config/app \
    --data-values-file config/overrides/override-values.yml \
    | kbld -f- --imgpkg-lock-output config/app/images.lock.yml > hello-app.yml
```


### WIP ###

TO-DO:
- imgpkg package and unpackage (pull/copy)
- kapp -a hello-app -f dependencies.yml -f resources.yml



