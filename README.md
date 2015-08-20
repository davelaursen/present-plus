# Present-Plus

Present-Plus is a fork of the Go Present tool (golang.org/x/tools/cmd/present) that adds new features such as themes. Since the additional features provided by Present-Plus are implemented within comments, your .slide and .article files remain completely compatible with Go Present.

## ALPHA

Present-Plus is currently an alpha product. See the [beta](https://github.com/davelaursen/present-plus/milestones) milestone in the [Issues](https://github.com/davelaursen/present-plus/issues) section to see remaining items and the targetted beta release date.

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

## Contributions / Suggestions

This project will continue to evolve. The goal is to enhance Go Present to increase adoption and use, as I think that the idea of generating slide decks and articles from simple markdown is a great one.

If you are interested in submitting feature requests or making a pull request, please remember the prime directive for this project:

*At all times, articles and presentations that make use of Present-Plus's features must remain fully backwards-compatible with Go Present.*

Thanks!