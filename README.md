# carvel-demo

[Carvel](https://carvel.dev) comprises a set of single-purpose, composable tools to facilitate a workflow for deploying applications to Kubernetes.

This repo contains a demo/tutorial of Carvel tools using a simple demo app with a Redis dependency.

## Demo application (hello-app)

The _hello-app_ demo application returns "Hello world!" and a counter of requests. The counter is store in Redis, and "world" is configurable.

To get a better understanding of the app, see [README2.md](README2.md).
However, you can also skip to the Carvel instructions below, as the precise behavior of the app is not germane to the Carvel workflow.

## Deploy to Kubernetes using Carvel

The following steps will introduce most of the Carvel tools to **distribute and/or deploy** hello-app and its Redis dependency, including:
- vendoring in Redis
- building and publishing the hello-app image
- managing YAML configuration using templates and overlays
- packaging for distribution to online or air-gapped environments
- deployment to Kubernetes

Carvel tools can be used separately to achieve any subset of these objectives.
They can also be combined with non-Carvel tools, such as helm, kustomize, kubectl, etc.

#### Scenario

For this tutorial, assume you have written hello-app, utilizing Redis as a data store, and you want to package it so that you can easily share it with your friends and co-workers. Your friends can deploy the app to environments with internet access, but your co-workers are restricted to air-gapped environments (no internet access). To make it easy for them to download and deploy, you decide you want to vendor Redis into the application package, and you want to provide the deployment YAML for Kubernetes, with hooks for your friends and co-workers to make configuration changes.

#### Vendor in Redis dependency

You can include installation instructions with your app stating that Redis is a prerequisite and leave it to each user to figure out how to provide a Redis instance.
However, it would make life easier for your friends and co-workers—and more predictable for hello-app deployments—if you packaged Redis into your application.

[vendir](https://carvel.dev/vendir) can be used to ensure data sources are provided in a consistent manner. In this case, you can use it to provide Redis as part of the hello-app package.

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

#### Generate YAML config

[ytt](https://carvel.dev/ytt) supports templating and overlaying YAML configuration.

For example, look at [config.yml](bundle/config/base/config.yml).
It contains comments that begin with '#@ ' with instructions for interpolating values or YAML fragments.
These comments are more powerful than simple templating; they are written in a Python-like language called [Starlark](https://github.com/google/starlark-go), so they can include modules, functions, and conditional/looping logic.
In this file you can see some examples:
- a module import (`load(...)`)
- a function definition (`def labels()`)
- conditional statements (`if/end`)

Default values for interpolation are specified in [default-values.yml](bundle/config/default-values.yml), and overrides to change the defaults are specified in [values.yml](values.yml).

ytt also supports overlays.
Take a look at [redis.yml](bundle/config/overlay/redis.yml).
The Starlark code in this file instructs ytt to identify all YAML resources with `metadata.labels.app: redis` and set the `metadata.namespace` field (the value for the namespace field is specified in the [default values file](bundle/config/default-values.yml)).
This overlay matching logic will match the Redis resources that you are vendoring in, so it makes sense to keep these files in their original state and modify the resources using overlays.

Run the following command to process the templates and overlays and produce pure YAML.
> Note: This command will simply output YAML to the terminal for review.
```shell
ytt -f bundle/config \
    --data-values-file values.yml
```

Notice that the YAML contains the Redis resources as well as the Deployment and Service for the demo app (3 Deployments and 3 Services in total).
All Starlark markup has been resolved.

This YAML can be applied directly to Kubernetes.
However, this YAML is not specific enough to guarantee reproducible deployments because the images use mutable tags instead of immutable SHAs. To see this, you can re-run the above command and add ` | grep "image:"` at the end. The output should look something like this:
```yaml
        image: gcr.io/fe-ciberkleid/carvel-demo/hello-app
        image: gcr.io/google_samples/gb-redis-follower:v2
        image: docker.io/redis:6.0.5
```

It would be better to replace these tags with their respective SHAs.
Carvel provides a tool to automate this process. Continue to the next step to learn more.

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

> **Important:** Update the value of `helloApp.pushImageTag` in the [default values file](bundle/config/default-values.yml) so that it points to a registry to which you can publish. 
> Also, make sure you are authenticated (i.e. run [docker login](https://docs.docker.com/engine/reference/commandline/login) at your command line). 

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

Check your registry to verify that the hello-app image was published. This shows that objective #1 above was achieved.

Also, check the lock file that was created, [images.yml](bundle/.imgpkg/images.yml).
Notice that the lock file contains a mapping of original image tags to resolved SHAs.
This validates that kbld resolved the tags, showing that objective #2 above was achieved.

To achieve #3 above, use the [images lock file](bundle/.imgpkg/images.yml) that you just created as input to kbld.

Run the following command to see this in action.
In this case, you are running kbld again, but here it is only replacing the image tags with the values in the lock file.
> Note: This command will simply output YAML to the terminal for review.
```shell
ytt -f bundle/config \
    -f bundle/.imgpkg/images.yml \
    --data-values-file values.yml \
    | kbld -f-
```

You can also combine the two commands above into a single command:
```shell
ytt -f bundle/kbld.yml \
    -f bundle/config \
    -f bundle/.imgpkg/images.yml \
    --data-values-file values.yml \
    | kbld -f- --imgpkg-lock-output bundle/.imgpkg/images.yml
```

Try deleting the lines that pertain to hello-app in [images lock file](bundle/.imgpkg/images.yml) (3 lines total to delete), leaving only the redis images.

Re-run the last command above.
Watch the output to see that the image is rebuilt.
Check the lock file again - it has been updated with the image SHA to enable re-using the image, rather than re-building, in future builds.

The YAML output can be applied directly to Kubernetes.
However, Carvel also provides a tool for packaging applications for distribution. Continue to the next step to learn more.

#### Package as image for distribution

[imgpkg](https://carvel.dev/imgpkg) packages files into an OCI image so that they can be easily stored and distributed using an OCI registry.
It also makes it easy to unpack the contents.
In addition, imgpkg can copy any images referenced to a local registry, making it a very useful tool for air-gapped environments.

For convenience, use an environment variable to set the image name and tag to publish:
```shell
# sample value:
# BUNDLE_IMG=gcr.io/fe-ciberkleid/carvel-demo/hello-app-bundle:v1.0.0
BUNDLE_IMG=<your-value-here>
```
Run the following command to package the demo app and publish it to a registry.

```shell
imgpkg push -b $BUNDLE_IMG \
            -f bundle \
            --lock-output bundle/bundle.lock.yml
```

Check your registry to verify that the hello-app-bundle image was published. 

#### Download distribution bundle

At this point, you can tell all your friends about your cool new hello-app-bundle that uses Redis as a data store.
They can all download the bundle and deploy it to their own Kubernetes environments!
To do this, they can run:
```shell
imgpkg pull -b $BUNDLE_IMG -o temp/hello-app-bundle
```

Notice that the pull command downloaded and unpacked the OCI image contents to the local directory.
Your friends can create/update their own [overrides values file](values.yml) and run ytt and kbld to process the configuration:
```shell
ytt -f temp/hello-app-bundle/config \
    -f temp/hello-app-bundle/.imgpkg/images.yml \
    --data-values-file values.yml \
    | kbld -f-
```

This YAML can be applied directly to Kubernetes.
Before doing that, however, it turns out your co-workers are also interested in deploying hello-app, but they are restricted to an air-gapped environment. The existing bundle references images on the public internet (on Docker Hub and GCR, in this case), so it will not work for them.

Luckily, imgpkg can also help with distribution of packages within air-gapped environments. Continue to the next step to learn more.

#### Copy distribution bundle for air-gapped environments

imgpkg has a "copy" command that copies not only the bundle itself, but also all **all of the images it references** to a destination of your choice.

For convenience, use an environment variable to store the address of the internal registry.
> Note: Since this is a demo, the sample value below actually uses a remote registry as well, but suspend disbelief for a moment and pretend that the `carvel-demo/internal` namespace is indeed only accessible on the internal network.
> Either way, please set the value to a registry to which you can publish and make sure you are authenticated at the command line.
```shell
# sample value:
# INTERNAL_REPO=gcr.io/fe-ciberkleid/carvel-demo/hello-app-bundle-internal
INTERNAL_REPO=<your-value-here>
```

Next, copy the bundle internally.
This command could, for example, be run from a jump box with access to both environments, or by copying to disk first.
```shell
imgpkg copy -b $BUNDLE_IMG --to-repo $INTERNAL_REPO
```

Check you internal registry to ensure that all images have been copied.
Notice that the original bundle is a single image, whereas the internal repo contains a number of images.

Your co-workers can now pull the internal copy of the bundle to their local machines:
```shell
imgpkg pull -b $INTERNAL_REPO:v1.0.0 -o temp/hello-app-bundle-internal
```

Compare the contents of the `.imgpkg/images.yml` files in both directories:
```shell
diff temp/hello-app-bundle/.imgpkg/images.yml \
     temp/hello-app-bundle-internal/.imgpkg/images.yml
```

The output might look something like this, with your respective registry values:
```shell
6c6
<   image: index.docker.io/library/redis@sha256:800f2587bf3376cb01e6307afe599ddce9439deafbd4fb8562829da96085c9c5
---
>   image: gcr.io/fe-ciberkleid/carvel-demo/hello-app-bundle-internal@sha256:800f2587bf3376cb01e6307afe599ddce9439deafbd4fb8562829da96085c9c5
9c9
<   image: gcr.io/fe-ciberkleid/carvel-demo/hello-app@sha256:7243c1434280d54579b39f1a26ec9a01b301a17bcb5e23a6b03fd4d1228bc549
---
>   image: gcr.io/fe-ciberkleid/carvel-demo/hello-app-bundle-internal@sha256:7243c1434280d54579b39f1a26ec9a01b301a17bcb5e23a6b03fd4d1228bc549
12c12
<   image: gcr.io/google_samples/gb-redis-follower@sha256:42707dbdccb4c8177523e2687c7b3cdb3d13473b0df1d3ef2c558fda09772c6f
---
>   image: gcr.io/fe-ciberkleid/carvel-demo/hello-app-bundle-internal@sha256:42707dbdccb4c8177523e2687c7b3cdb3d13473b0df1d3ef2c558fda09772c6f
```

All image references have been updated to point to internal copies.

Your co-workers can create/update their own [overrides values file](values.yml) and run ytt and kbld to process the configuration.
They must use the `.imgpkg/images.yml` to ensure all resources are pulled from the internal registry.
```shell
ytt -f temp/hello-app-bundle-internal/config \
    -f temp/hello-app-bundle-internal/.imgpkg/images.yml \
    --data-values-file values.yml \
    | kbld -f-
```

This YAML can be applied directly to Kubernetes.
However, rather than piping the YAML to `kubectl`, let's explore another Carvel tool that provides a richer deployment experience.
Continue to the next step to learn more.

#### Deploy to Kubernetes

[kapp](https://carvel.dev/kapp) deploys resources to Kubernetes and enables you to operate on them as a group, tracking their relationship to each other as part of a single application and providing improved behavior for ordering of resources, obtaining logs, and more.

Several times throughout this tutorial it was stated that YAML output could be directly applied to Kubernetes. You can choose any of those instances for this example. For simplicity, these commands will use the last example.

Re-run the last command, but this time, pass the output to kapp.
> Note: Using process substitution rather than piping to kapp preserves the ability to confirm at the prompt.
```shell
kapp deploy -a hello-app -c -f <(
  ytt -f temp/hello-app-bundle-internal/config \
  -f temp/hello-app-bundle-internal/.imgpkg/images.yml \
  --data-values-file values.yml \
  | kbld -f-)
```

Notice that kapp:
- orders the resources (e.g. Namespace is applied first)
- provides insight into what resources and operations that will be applied
- provides a prompt to enable evaluation before deploying

Enter `y` at the prompt.

Notice that kapp:
- provides output related to the deployment
- waits for resources to be ready before completing

Try a fe other kapp commands and notice how kapp treats the various resources as parts of a single aaplication

```shell
kapp list; kapp inspect -a educates; kapp logs -a educates
kapp inspect -a educates --raw --tty=false | kbld inspect -f -
```

#### TL;DR

Cheet sheat.
Remember to change registry values in the default-values.yml and the two env vars below.
```shell
vendir sync --chdir bundle

kbld  -f bundle/kbld.yml \
      -f bundle/config \
      --imgpkg-lock-output bundle/.imgpkg/images.yml \
      > /dev/null

BUNDLE_IMG=gcr.io/fe-ciberkleid/carvel-demo/hello-app-bundle:v1.0.0
INTERNAL_REPO=gcr.io/fe-ciberkleid/carvel-demo/hello-app-bundle-internal

imgpkg push -b $BUNDLE_IMG \
            -f bundle \
            --lock-output bundle/bundle.lock.yml

imgpkg copy -b $BUNDLE_IMG --to-repo $INTERNAL_REPO

imgpkg pull -b $BUNDLE_IMG -o temp/hello-app-bundle

imgpkg pull -b $INTERNAL_REPO:v1.0.0 -o temp/hello-app-bundle-internal


kapp deploy -a hello-app -c -f <(
  ytt -f temp/hello-app-bundle-internal/config \
  -f temp/hello-app-bundle-internal/.imgpkg/images.yml \
  --data-values-file values.yml \
  | kbld -f-)
```