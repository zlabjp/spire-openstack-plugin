# Contributing

## Prerequisites

- Go 1.11 or higher

## Building

We use `dep` for vendoring Go dependencies.
If you add new dependencies, resolve dependencies as follows.

```
$ make vendor
```

And you can test and build codes as follows.

```
$ make test

$ make build
```

## Contributor License Agreement

Contributions to this project must be accompanied by a Contributor License Agreement(CLA). Please read our [CLA](https://zlabjp.github.io/cla/). 

### How to sign

We'll confirm your agreement as part of the PR merge operation.
If you accept it, please say so in the thread.

Generally, it's necessary to submit CLA only once, so if you have already submitted it, you don't need to submit it again.
