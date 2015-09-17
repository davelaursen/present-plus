# Present-Plus

Present-Plus is a fork of the Go Present tool (golang.org/x/tools/cmd/present) that adds new features such as themes. Since the additional features provided by Present-Plus are implemented within comments, your .slide and .article files remain completely compatible with Go Present.

If you're looking for themes, check out the [Present-Plus-Themes](https://github.com/davelaursen/present-plus-themes) repo.

## BETA

Present-Plus is currently a beta product. Please submit any issues that you find so that they can be addressed before the 1.0 Release.

See the [1.0 Release](https://github.com/davelaursen/present-plus/milestones) milestone in the [Issues](https://github.com/davelaursen/present-plus/issues) section to see remaining items and the targeted 1.0 release date.

## Installation

Make sure you have a working Go environment. [See the install instructions](http://golang.org/doc/install.html).

To install (or update) Present-Plus, simply run:
```
$ go get -u github.com/davelaursen/present-plus
```

Make sure your GOPATH environment variable is set and your PATH includes the `$GOPATH/bin` directory so Present-Plus can easily be run.

## Getting Started

To learn how to create Go Present files, [check out the official documentation](https://godoc.org/golang.org/x/tools/present).

To learn how to create custom themes and take advantage of what Present-Plus has to offer, start Present-Plus from the examples directory and navigate to `localhost:4999` in your browser:

    $ cd $GOPATH/src/github.com/davelaursen/present-plus/examples
    $ present-plus

The sample slide decks provide details on how to use Present-Plus and create custom themes.

## Feature Overview

#### Themes

Present-Plus provides the ability to easily style your Go Present slides and articles. You can even apply a theme to the directory listing page. Present-Plus comes with two simple themes, but the 'install' command allows you to easily download and install additional ones. Or if you have some CSS skills, create your own!

#### Formatting

Present-Plus includes the ability to tweak how your presentations are rendered. For example, you can hide the last 'Thank you' slide for internal or informal presentations, and you can customize multiple aspects of the directory view.

#### Share Your Style

Share your creations! If a theme is accessible on GitHub, then it can be downloaded and installed using the 'install' command. And while Present-Plus only has two built-in themes ('white' and black'), the [Present-Plus-Themes](https://github.com/davelaursen/present-plus-themes) repo will continue to grow with new themes that you can install and use.

For more details, see the Getting Started section to view a detailed presentation on Present-Plus's features.

## Contributions / Suggestions

This project will continue to evolve. The goal is to enhance Go Present to increase adoption and use, as I think that the idea of generating slide decks and articles from simple markdown is a great one.

If you are interested in submitting feature requests or making a pull request, please remember the prime directive for this project:

*At all times, articles and presentations that make use of Present-Plus's features must remain fully backwards-compatible with Go Present.*

Thanks!