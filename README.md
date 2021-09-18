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
or:
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

[vendir](https://carvel.dev/vendir) can be used to vendor a dependency into an application.

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

[ytt](https://carvel.dev/ytt) supports templating and overlaying YAML configuration.

For example, look at [config/app/base/config.yml](config/app/base/config.yml).
It contains comments that begin with '#@ ' with instructions for interpolating values or YAML fragments.
These comments are more powerful than simple templating; they are written in a Python-like language called Starlark, so they can include modules, functions, and conditional/looping logic.
In this file you can see some examples:
- a module import (`load(...)`)
- a function definition (`def labels()`)
- conditional statements (`if/end`)

Default values for interpolation are specified in [config/app/base/values.yml](config/app/base/values.yml), and overrides to change the defaults are specified in [config/app/overrides/override-values.yml](config/overrides/override-values.yml).

ytt also supports overlays.
Take a look at [config/app/base/redis-overlay.yml](config/app/base/redis-overlay.yml).
The Starlark code in this file instructs ytt to identify all YAML resources with `metadata.labels.app: redis` and set the `metadata.namespace` field (the value for the namespace field is specified in the [default values file](config/app/base/values.yml)).
This overlay matching logic will match the Redis resources that you are vendoring in, so it makes sense to keep these files in their original state and modify the resources using overlays.

Run the following command to process the templates and overlays and produce pure YAML.
> Note: This command will simply output YAML to the terminal for review.
```shell
ytt -f config/app \
    --data-values-file config/overrides/override-values.yml
```

Notice that the YAML contains the resources from the vendored Redis files (2 Deployments and 2 Services), as well as the Deployment and Service for the demo app.

It also contains two resources of type `Config` with `apiVersion: kbld.k14s.io/v1alpha1`.
These are not intended for deployment to Kubernetes.
Rather, they are intended as input for another Carvel tool.
Continue to the next step to learn more.

#### Generate YAML config

Primarily, [kbld](https://carvel.dev/kbld) replaces image tags with their respective SHAs.
It can also generate a lock file that can be used to pin the versions of images, and it can be used to automate building images (e.g. using Dockerfile or Buildpacks).

Take a look at the kbld configuration file, [config/app/base/build.yml](config/app/base/build.yml).
- The `sources` section says to use Paketo Buildpacks to build the demo app that is located in the [src](src) folder.
(Buildpacks provide a consistent and structured way to build images, without the need to write Dockerfiles or other custom code).
- The `destinations` section says to push the image to Docker Hub if and only if the `helloApp.pushImageTag` value is set.
You can see that, by default, this value is set (check the [default values file](config/app/base/values.yml)).

**Important:** Update the value of `helloApp.pushImageTag` in the default values file or in the override values file so that it points to a registry to which you can publish.

Run the following command to pipe the processed YAML to kbld.
When the command has finished, you should see an image published to your registry, and the YAML output will contain SHAs instead of mutable tags in all `image:` fields.
Additionally, notice that a new file was created that will pin demo app and vendored image versions in subsequent executions ([config/app/images.lock.yml](config/app/images.lock.yml)).
> Note: This command will simply output YAML to the terminal for review.
```shell
ytt -f config/app \
    --data-values-file config/overrides/override-values.yml \
    | kbld -f- --imgpkg-lock-output config/app/images.lock.yml
```

Notice that the YAML output no longer contains the resources of type `Config` with `apiVersion: kbld.k14s.io/v1alpha1`.
This YAML can be applied directly to Kubernetes.
However, Carvel also provides a tool for packaging applications for distribution. Continue to the next step to learn more.

### WIP ###

TO-DO:
- imgpkg package and unpackage (pull/copy)
- kapp -a hello-app -f dependencies.yml -f resources.yml



