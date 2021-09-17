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

[Carvel](https://carvel.dev) comprises a set of single-purpose, composable tools to facilitate a workflow for deploying applications to Kubernetes.

The following steps will introduce most of the Carvel tools to deploy this demo app and its Redis dependency, including:
- vendoring in Redis
- building and publishing the demo app image
- managing YAML configuration using templates and overlays
- packaging for online or air-gapped environments
- deployment to Kubernetes

#### Vendor in Redis dependency

[vendir](https://carvel.dev/vendir/) can be used to vendor a dependency into an application.

Review the file [config/vendir.yml](config/vendir.yml).
It specifies a git repo as a source of Redis YAML config files.

Run the following command to sync the remote files to your local machine.
This step also generates a lock file that can be used to pin the version of the vendored files in subsequent syncs.
> Note: If you already have a vendir lock file and want to sync using the pinned version, uncomment the  _--locked_ flag.
```shell
vendir sync --chdir config # --locked
```

Use `git status` to see the changes.

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



