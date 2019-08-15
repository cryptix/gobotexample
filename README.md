# gobotexample

this is a starting example to use the go-ssb "ssb-server" as a package.


# gomobile

The [Mobile page on the golang wiki](https://github.com/golang/go/wiki/Mobile) is a good starting intro. Use the SDK instructions but change `golang.org/x/mobile/example/basic` with this package. Probably want to fork it first and then add all your stuff like passing configs from android and setting up storage locations and what not.... ;)

## issues

* there is some weirdness with go modules and gomobile, you want to use `export GO111MODULE=off` for some time. (TODO: issue link)
