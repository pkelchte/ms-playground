ms-playground
=============

The repository for [MS Playground](http://ms-playground.appspot.com/), a modified [Go Playground](http://play.golang.org/) with some extras.

The [wiki pages for MS-related algorithms](https://github.com/pkelchte/ms-playground/wiki) are open to edit for everyone with a github account.

---

The go-playground folder contains the front-end of the playground, which is deployed to appspot.

backend.go in the root folder is supposed to run on another server.

That server should have a working golang 1.3 compiler toolchain (for all platforms, including nacl),
as well as a version of the NaCl SDK, to run programs in a sandboxed environment.

