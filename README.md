# carvel-demo

[Carvel](https://carvel.dev) comprises a set of single-purpose, composable tools to facilitate a workflow for deploying applications to Kubernetes.

This repo contains a demo/tutorial of Carvel tools using a simple demo app with a Redis dependency.

## Demo application (hello-app)

The _hello-app_ demo application returns "Hello world!" and a counter of requests. The counter is stored in Redis, and "world" is configurable.

To get a better understanding of the app, see [README2.md](README2.md).
However, you can also skip to the Carvel instructions below, as the precise behavior of the app is not germane to the Carvel workflow.

## Deploy to Kubernetes using Carvel

The following steps will introduce most of the Carvel tools to **configure, package, distribute and/or deploy** hello-app and its Redis dependency, including:
- vendoring in Redis
- building and publishing the hello-app image
- managing YAML configuration using templates and overlays
- packaging for distribution to online or air-gapped environments
- deployment to Kubernetes

Carvel tools can be used separately to achieve any subset of these objectives.
They can also be combined with non-Carvel tools, such as helm, kustomize, kubectl, etc.

#### Scenario

Assume you have written [hello-app](src/hello-app.go) (nice job!), utilizing Redis as a data store, and you want to share it with your friends and co-workers.
Your friends can deploy the app to environments with internet access, but your co-workers are restricted to air-gapped environments (no internet access).
You could simply build an image from your source code and let them worry about providing Redis and the YAML for Kubernetes. However, you're nicer than that and you've got this powerful suite of tools at your disposal, so you decide to make it easy for your friends and co-workers to download, configure, and deploy the application.
This tutorial will walk you through those steps.

#### Pre-requisites
- [Carvel](https://carvel.dev/#whole-suite) suite installed locally
- Docker installed locally
- Access to a Kubernetes cluster
- Access to an image registry (for publishing images)
- [pack](https://buildpacks.io/docs/tools/pack/) CLI (for building hello-app image without a Dockerfile)

One option for the cluster and registry is [kind with local registry](https://kind.sigs.k8s.io/docs/user/local-registry).
This starts a Kubernetes cluster running on Docker on your local machine, with an image registry listening on `localhost:5000`.
That is what is used in the examples below.

To set up the kind cluster, run:
```shell
curl https://kind.sigs.k8s.io/examples/kind-with-registry.sh -o kind-with-registry.sh \
  && chmod +x kind-with-registry.sh \
  && ./kind-with-registry.sh \
  && kubectl cluster-info --context kind-kind
```

**Let's get started!**

#### Vendor in Redis dependency

hello-app uses Redis to store request counters.
You could simply include installation instructions with your app stating that Redis is a prerequisite and leave it to your friends and co-workers to figure out how to provide a Redis instance.
However, it would make life easier for them—and more predictable for hello-app deployments—if you packaged Redis into your application.

[vendir](https://carvel.dev/vendir) can be used to ensure data sources are provided in a consistent manner. In this case, you can use it to provide Redis resources as part of the hello-app package.

Review [vendir.yml](bundle/vendir.yml).
It specifies a git repo as a source of Redis YAML config files, as well as a destination directory for the sync.

Run the following command to sync the remote files to your local machine.
This step also generates a lock file that can be used to pin the version in subsequent executions.
> Note: If you already had a vendir lock file and wanted to sync using the pinned version, you could uncomment the _--locked_ flag.
```shell
vendir sync --chdir bundle # --locked
```

This command can take some time to complete.
When it is done, use `git status` to see the changes.
You should see a new directory with Redis resources, as well as the vendir lock file.

#### Generate YAML config

[ytt](https://carvel.dev/ytt) supports templating and overlaying YAML configuration.

For example, look at [config.yml](bundle/config/base/config.yml).
It contains comments that begin with '#@ ' with instructions for interpolating values or YAML fragments.
These comments are more powerful than simple templating; they are written in a Python dialect called [Starlark](https://github.com/google/starlark-go), so they can include modules, functions, and conditional/looping logic.
In this file you can see some examples:
- a module import (`load(...)`)
- a function definition (`def labels()`)
- conditional statements (`if/end`)

Default values for interpolation are specified in [default-values.yml](bundle/config/default-values.yml), and overrides to change the defaults are specified in [values.yml](values.yml).

ytt also supports overlays.
Take a look at [redis.yml](bundle/config/overlay/redis.yml).
The Starlark code in this file instructs ytt to identify all YAML resources with `metadata.labels.app: redis` and set the `metadata.namespace` field (the default value for the namespace field is specified in the [default values file](bundle/config/default-values.yml)).
Overlays enable you to modify YAML that is not templated, and to modify resources without changing the original file.
Using overlays will enable you to sync updated Redis files in the future without overwriting the configuration changes you want to make.

Run the following command to process the templates and overlays and produce pure YAML.
> Note: This command will simply output YAML to the terminal for review.
```shell
ytt -f bundle/config
```

Notice that the YAML contains the Redis resources as well as the Deployment and Service for the demo app (3 Deployments and 3 Services in total).
All Starlark markup has been resolved.

This YAML can be applied directly to Kubernetes.
However, this YAML is not specific enough to guarantee reproducible deployments because the images use mutable tags instead of immutable SHAs.
To see this, you can re-run the above command and add ` | grep "image:"` at the end.
The output should look something like this:
```yaml
        image: localhost:5000/hello-app
        image: gcr.io/google_samples/gb-redis-follower:v2
        image: docker.io/redis:6.0.5
```

It would be better to replace these tags with their respective SHAs.
Carvel provides a tool to automate this process.
Continue to the next step to learn more.

#### Resolve images

[kbld](https://carvel.dev/kbld) replaces image tags with their respective SHAs.
It can also generate a lock file that can be used to pin the versions of images, and it can be used to automate building images (e.g. using Dockerfile or Buildpacks).

In this step, you will use kbld to:
1. build an image for hello-app and publish it to an OCI registry
2. resolve image tags to SHAs for all images (hello-app and the redis images)
3. create a lock file for pinning the versions to the resolved SHAs

To achieve #1 above, take a look at the kbld configuration file, [kbld.yml](bundle/kbld.yml).
- The `sources` section tells kbld to use buildpacks to create an image for hello-app, located in the [src](src) folder.
    - _Buildpacks_ are a separate technology that provides a consistent and structured way to build images, without the need to write Dockerfiles or other custom code.
      You can learn more [here](https://buildpacks.io).
- The `destinations` section says to push the image to Docker Hub if and only if the `helloApp.pushImageTag` value is set.

> **Important:** If you are not using kind with a local registry at `localhost:5000`, update the value of `helloApp.pushImageTag` in the [default values file](bundle/config/default-values.yml) so that it points to a registry to which you can publish, and make sure you are authenticated to your registry (i.e. run [docker login](https://docs.docker.com/engine/reference/commandline/login) at your command line). 

Run kbld on the YAML configuration files.
This can be done on the raw YAML or on the ytt-processed YAML.
In some cases (though not necessarily this demo), it is advantageous to run it on the raw YAML, as the default values may cause certain images to be filtered out.

For example, run the following command:
```shell
kbld  -f bundle/kbld.yml \
      -f bundle/config \
      --imgpkg-lock-output bundle/.imgpkg/images.yml \
      > /dev/null
```

Check your registry to verify that the hello-app image was published.
This shows that objective #1 above was achieved.
```shell
curl localhost:5000/v2/hello-app/tags/list
```

Also, verify that the lock file that was created, [images.yml](bundle/.imgpkg/images.yml).
Notice that the lock file contains a mapping of original image tags to resolved SHAs.
This validates that kbld resolved the tags, showing that objective #2 above was achieved.

To achieve #3 above, use the [images lock file](bundle/.imgpkg/images.yml) that you just created as input to kbld.

Run the following command to see this in action.
In this case, you are running kbld again, but here it is only replacing the image tags with the values in the lock file.
> Note: This command will simply output YAML to the terminal for review.
```shell
ytt -f bundle/config \
    -f bundle/.imgpkg/images.yml \
    | kbld -f-
```

Scroll through the output and notice that the image tags are now SHAs.

Re-run the last command above.
Notice that it runs much more quickly since it does not have to build hello-app again or resolve the redis tags.
You can delete the lock file (or the data for any particular image in the lock file) to force kbld to re-build or re-resolve a tag.

The YAML output can be applied directly to Kubernetes.
However, Carvel also provides a tool for packaging applications for distribution. Continue to the next step to learn more.

Note: You can combine the two commands above into a single command as shown below.
This is fine if your default values file does not filter out an image that a user might need if they set custom values.
```shell
ytt -f bundle/kbld.yml \
    -f bundle/config \
    -f bundle/.imgpkg/images.yml
    | kbld -f- --imgpkg-lock-output bundle/.imgpkg/images.yml
```

#### Package as image for distribution

[imgpkg](https://carvel.dev/imgpkg) packages files into an OCI image so that they can be easily stored and distributed using an OCI registry.
It also makes it easy to unpack the contents.
In addition, imgpkg can copy any images referenced to a local registry, making it a very useful tool for air-gapped environments.

Run the following command to package the demo app and publish it to a registry.

```shell
imgpkg push -b localhost:5000/hello-app-bundle:v1.0.0 \
            -f bundle \
            --lock-output bundle/bundle.lock.yml
```

Check your registry to verify that the hello-app-bundle image was published. 
```shell
curl localhost:5000/v2/hello-app-bundle/tags/list
```

The output should look something like this.
```json
{
  "name": "hello-app-bundle",
  "tags": [
    "v1.0.0"
  ]
}
```

In contrast to the hello-app image that you published previously, this bundle includes all of the YAML templates for hello-app and Redis (i.e. everything in the [bundle](bundle) directory). Your friends and co-workers can just download this bundle rather than cloning a git repo or other data source.

#### Download distribution bundle

The imgpkg bundle is all that your friends will need.
Let's go through the workflow they would follow.

Use `imgpkg pull` to download and unpack the imgpkg bundle to a local directory:
```shell
imgpkg pull -b localhost:5000/hello-app-bundle:v1.0.0 \
            -o temp/hello-app-bundle
```

Your friends can create/update their own [overrides values file](values.yml) and run ytt and kbld to process the configuration.
In this case, let's store the YAML output in a file to use later.
```shell
ytt -f temp/hello-app-bundle/config \
    -f temp/hello-app-bundle/.imgpkg/images.yml \
    --data-values-file values.yml \
    | kbld -f- \
    > hello-friends.yml
```

This YAML can be applied directly to Kubernetes.

Before doing that, however, notice that the tags in this YAML point to images on the internet. This will work for your friends, but not your co-workers. Let's first make sure your co-workers can access all the resources they need, too.

#### Copy distribution bundle for air-gapped environments

imgpkg has a "copy" command that copies a bundle **and all of the images it references** to a destination of your choice.
This is the perfect solution for your co-workers.

Let's go through the workflow they would follow.

Use `imgpkg copy` to copy the bundle to an internal registry.
For the purposes of this demo, we'll use another namespace in the same local registry to represent an internal registry in a co-worker's environment, and we'll assume there is a jump box with access to the internet and the local registry.
```shell
imgpkg copy -b localhost:5000/hello-app-bundle:v1.0.0 \
            --to-repo localhost:5000/hello-app-bundle-internal
```

Check the "internal registry" to ensure that all images have been copied.
```shell
curl localhost:5000/v2/hello-app-bundle-internal/tags/list
```

The output should look something like this.
You can compare the SHAs to those in the image lock to figure out which images are hello-app and Redis.
```json
{
  "name": "hello-app-bundle-internal",
  "tags": [
    "sha256-ab70844a842cf3b1440a733da9851174d2c981e926d23a35041b767bd8521c9b.imgpkg",
    "sha256-0a5f1a532ed79b7f8f0581b499f09f602c4cbc3dd6f87ba7282d78e42f3d6d68.imgpkg",
    "v1.0.0",
    "sha256-42707dbdccb4c8177523e2687c7b3cdb3d13473b0df1d3ef2c558fda09772c6f.imgpkg",
    "sha256-800f2587bf3376cb01e6307afe599ddce9439deafbd4fb8562829da96085c9c5.imgpkg",
    "sha256-0a5f1a532ed79b7f8f0581b499f09f602c4cbc3dd6f87ba7282d78e42f3d6d68.image-locations.imgpkg"
  ]
}
```

Your co-workers can now pull the internal copy of the bundle to their local machines:
```shell
imgpkg pull -b localhost:5000/hello-app-bundle-internal:v1.0.0 \
            -o temp/hello-app-bundle-internal
```

Compare the contents of the `.imgpkg/images.yml` files in both directories:
```shell
diff temp/hello-app-bundle/.imgpkg/images.yml \
     temp/hello-app-bundle-internal/.imgpkg/images.yml
```

The output might look something like this.
Notice that `imgpkg copy` updated the references to point to the target ("internal") registry.
```shell
6c6
<   image: index.docker.io/library/redis@sha256:800f2587bf3376cb01e6307afe599ddce9439deafbd4fb8562829da96085c9c5
---
>   image: localhost:5000/hello-app-bundle-internal@sha256:800f2587bf3376cb01e6307afe599ddce9439deafbd4fb8562829da96085c9c5
9c9
<   image: gcr.io/google_samples/gb-redis-follower@sha256:42707dbdccb4c8177523e2687c7b3cdb3d13473b0df1d3ef2c558fda09772c6f
---
>   image: localhost:5000/hello-app-bundle-internal@sha256:42707dbdccb4c8177523e2687c7b3cdb3d13473b0df1d3ef2c558fda09772c6f
12c12
<   image: localhost:5000/hello-app@sha256:ab70844a842cf3b1440a733da9851174d2c981e926d23a35041b767bd8521c9b
---
>   image: localhost:5000/hello-app-bundle-internal@sha256:ab70844a842cf3b1440a733da9851174d2c981e926d23a35041b767bd8521c9b
```

Your co-workers can create their own [overrides values file](values2.yml) and run ytt and kbld to process the configuration.
They **must** use the `.imgpkg/images.yml` to ensure all resources are pulled from the internal registry.
Again, let's store the YAML output in a file to use later.
```shell
ytt -f temp/hello-app-bundle-internal/config \
    -f temp/hello-app-bundle-internal/.imgpkg/images.yml \
    --data-values-file values2.yml \
    | kbld -f- \
    > hello-coworkers.yml
```

This YAML can be applied directly to Kubernetes.
However, rather than piping the YAML to `kubectl`, let's explore another Carvel tool that provides a richer deployment experience.
Continue to the next step to learn more.

#### Deploy to Kubernetes

[kapp](https://carvel.dev/kapp) deploys resources to Kubernetes and enables you to operate on them as a group, tracking their relationship to each other as part of a single application and providing improved behavior for ordering of resources, obtaining logs, and more.

Use kapp to apply the coworkers configuration.
```shell
kapp deploy -a hello-coworkers -f hello-coworkers.yml
```

Notice that kapp:
- orders the resources (e.g. Namespace is applied first)
- provides insight into the resources and operations that will be applied
- provides a prompt to enable evaluation before deploying

Enter `y` at the prompt.

Notice that kapp:
- provides output related to the deployment
- waits for resources to be ready before completing

Try a few other kapp commands and notice how kapp treats the various resources as parts of a single application:
```shell
kapp list
```

```shell
kapp inspect -a hello-coworkers
```

```shell
kapp logs -a hello-coworkers
```

```shell
kapp inspect -a hello-coworkers --raw --tty=false | kbld inspect -f -
```

With Carvel, it is also easier to deploy multiple instances of an application to the same namespace and manage each app as a whole.
For example, with a slight change configuration, we can deploy a second instance of hello-app to an existing namespace:
> Note: Using process substitution rather than piping to kapp preserves the ability to confirm at the prompt.
```shell
kapp deploy -a hello-partners -c -f <(
  ytt -f temp/hello-app-bundle-internal/config \
  -f temp/hello-app-bundle-internal/.imgpkg/images.yml \
  --data-values-file values3.yml \
  | kbld -f-)
```

Re-run the kapp commands above using `-a hello-partners`.

Delete the partner app:
```shell
kapp delete -a hello-partners
```