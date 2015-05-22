# present-plus

Present-Plus is a fork of the Go Present tool (golang.org/x/tools/cmd/present) that adds new features such as themes.  Since the additional features provided by Present-Plus are specified in separate files, the .slide and .article files remain untouched and can continue to be rendered with Go Present.

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

Once you have a presentation file created, check out the examples folder to see how you can customize your presentation.