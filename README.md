# Present-Plus

Present-Plus is a fork of the Go Present tool (golang.org/x/tools/cmd/present) that adds new features such as themes. Since the additional features provided by Present-Plus are implemented within comments, your .slide and .article files remain completely compatible with Go Present.

## Backlog

Present-Plus is currently in development. The following represents the active backlog - once the the V1 items are complete, Present-Plus will be released as a beta product.

- add theme support to articles
- add sample slides for using remote stylesheets
- add more built-in themes

## Installation

Make sure you have a working Go environment. [See the install instructions](http://golang.org/doc/install.html).

To install Present-Plus, simply run:
```
$ go get github.com/davelaursen/present-plus
```

Make sure your PATH includes the `$GOPATH/bin` directory so Present-Plus can easily be run:
```
export PATH=$PATH:$GOPATH/bin
```

## Getting Started

To learn how to create Go Present files, [check out the official documentation](https://godoc.org/golang.org/x/tools/present).

To learn how to create custom themes and take advantage of what Present-Plus has to offer, start Present-Plus from the examples directory and navigate to `localhost:4999` in your browser:

    $ cd $GOPATH/src/github.com/davelaursen/present-plus/examples
    $ present-plus

The sample slide decks provide details on how to use Present-Plus and create custom themes.
