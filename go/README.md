# Client module for RPdb

A CLI client and module for RPdb written in [Go](https://go.dev/). It provides both a CLI interface and an API that you can use in your own Go applications.

# Installation

The module can be `go get` and imported as usual.

```sh
go get github.com/RPJoshL/RPdb/v4
```

To install the CLI client into your `$GOBIN`, you can execute the following command.

```
go install github.com/RPJoshL/RPdb/v4/go/cmd/rpdb@latest 
```

# Documentation

Installation details for the CLI and general concepts of the application are described in our [documentation](https://rpdb.rpjosh.de/docs/getting-started/installation/#native-console-application).

Examples for using the API interface can be found in the folder [`cmd/rpdb`](cmd/rpdb/) and [`cmd/demo`](cmd/demo). For API examples of all operations please refer to [GoDoc](https://pkg.go.dev/github.com/RPJoshL/RPdb/go).

# License

This project is licensed under the MIT. Please see the [LICENSE](LICENSE) file for a full license.