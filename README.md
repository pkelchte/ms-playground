ms-playground
=============

The repository for [MS Playground](http://ms-playground.appspot.com/), an extended [Go Playground](http://play.golang.org/) with download and graphing buttons. It was developed to be able to go to users computers and develop tools live in the browser. It was developed in parallel with the mass spectrometry data library [unthermo])(https://godoc.org/bitbucket.org/proteinspector/ms/unthermo)

The [wiki pages for MS-related algorithms](https://github.com/pkelchte/ms-playground/wiki) are open to edit for everyone with a github account.

---

The go-playground folder contains the front-end of the playground, which is deployed to appspot.

backend.go in the root folder is supposed to run on another server.

That server should have a working golang 1.3 compiler toolchain (for all platforms, including nacl),
as well as a version of the NaCl SDK, to run programs in a sandboxed environment.

