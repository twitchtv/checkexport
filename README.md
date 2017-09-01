# checkexport #

checkexport is a tool to make sure that all the stuff you export is actually
used somewhere else.

You run it against a particular package, and within a scope. By default, the
repo root of the targeted package is used.

## Install ##
```bash
go get github.com/twitchtv/checkexport
```

## Examples ##

Check whether exported values in `github.com/golang/dep/internal/gps` are used
anywhere else in `github.com/golang/dep`:

```bash
$ checkexport -scope=github.com/golang/dep/... github.com/golang/dep/internal/gps
/Users/snelson/go/src/github.com/golang/dep/internal/gps/lock.go:31:6: func LocksAreEq is exported but not used anywhere else in github.com/golang/dep/...
/Users/snelson/go/src/github.com/golang/dep/internal/gps/lock.go:78:6: type SimpleLock is exported but not used anywhere else in github.com/golang/dep/...
/Users/snelson/go/src/github.com/golang/dep/internal/gps/lock.go:153:25: method LockedProject.Eq is exported but not used anywhere else in github.com/golang/dep/...
/Users/snelson/go/src/github.com/golang/dep/internal/gps/manifest.go:46:2: method RootManifest.IgnoredPackages is exported but not used anywhere else in github.com/golang/dep/...
```

Check across repositories, even across everything in your $GOPATH:

```bash
# This can take a very long time if you have a lot of packages - a more targeted
# search is better!
$ checkexport -scope="..." github.com/spenczar/tdigest
```

If `-scope` is unset, it will try to deduce the root of a package's repo and use
that.
