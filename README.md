# This is an App github.com/Harardin/rate-limit.

# MOST Interesting files are: server.go and server_test.go

This is an example service. github.com/Harardin/rate-limit used only to have a base for executing simple rate limiting task.

# To execute:

    - to run test and see how logic works simply run:
        `make test`
    - to start program run:
        `make start`

Please re-init go modules according to your repo address.

# Use GPG

Commands to use GPG this is just example

To generate GPG keys:

    `gpg --batch --generate-key ./gpg/reciever.params`

    To see full list of params look in docs: https://www.gnupg.org/documentation/manuals/gnupg-devel/Unattended-GPG-key-generation.html

To export private key to use in GO:

    To get private key id: `gpg --list-secret-keys`

    `gpg --export-secret-keys --armor KEY_ID > private.gpg`

To export public key to use in GO client who sends message:
    To get public keys id: `gpg --list-keys`

    `gpg --export --armor KEY_ID > public.gpg`

GPG keys should be stored in a secure vault

# Introduction

This is a template for doing go-micro development using GitLab. It's based on the
[helloworld](https://github.com/micro/examples/tree/master/helloworld) Go Micro
template.

# Reference links

- [GitLab CI Documentation](https://docs.gitlab.com/ee/ci/)
- [Go Micro Overview](https://micro.mu/docs/go-micro.html)
- [Go Micro Toolkit](https://micro.mu/docs/go-micro.html)

# Getting started

First thing to do is update `main.go` with your new project path:

```diff
-       proto "gitlab.com/gitlab-org/project-templates/go-micro/proto"
+       proto "gitlab.com/$YOUR_NAMESPACE/$PROJECT_NAME/proto"
```

Note that these are not actual environment variables, but values you should
replace.

## What's contained in this project

- main.go - is the main definition of the service, handler and client
- proto - contains the protobuf definition of the API

## Dependencies

Install the following

- [micro](https://github.com/micro/micro)
- [protoc-gen-micro](https://github.com/micro/protoc-gen-micro)

## Run Service

```shell
go run main.go
```

## Query Service

```
micro call greeter Greeter.Hello '{"name": "John"}'
```
