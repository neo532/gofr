# Gofr Web Framework


[![Go Report Card](https://goreportcard.com/badge/github.com/neo532/gofr)](https://goreportcard.com/report/github.com/neo532/gofr)
[![Sourcegraph](https://sourcegraph.com/github.com/neo532/gofr/-/badge.svg)](https://sourcegraph.com/github.com/neo532/gofr?badge)

Gofr is a web framework written in Go (Golang).It aims to be a more easy framework.


## Contents

- [Gofr Web Framework](#gofr-web-framework)
  - [Installation](#installation)
  - [Validator](#validator)


## Installation

To install Gofr package, you need to install Go and set your Go workspace first.

1. The first need [Go](https://golang.org/) installed (**version 1.12+ is required**), then you can use the below Go command to install Gofr.

```sh
    $ go get -u github.com/neo532/gofr
```

2. Import it in your code:

```go
    import "github.com/neo532/gofr"
```

## HTTP request

It is a powerful-tool of request.It contains log/retry.

[click me to code](https://github.com/neo532/gofr/blob/master/request)

```go
    package main

    import (
        "github.com/neo532/gofr/request"
    )

    type ReqParam struct {
        Directory string `form:"directory"`
    }
    type Body struct {
        Directory string `json:"directory"`
    }

    func main() {
        // register logger if you like.
        // gofr/request/logger.go

        // build args
        var p = request.HTTP{
            Method: "GET",
            URL: "https://github.com/neo532/gofr",
            Limit: time.Duration(3)*time.Second,    // optional
            Retry: 2,                               // optional, default:1
        }.
        QueryArgs(&ReqParam{Directory: "request"}). // optional
        JsonBody(&Body{Directory: "request"}).      // optional
        Header(http.Header{"a": []string{"a1", "a2"}, "b":[]string{"b1", "b2"}}). // optional
        CheckArgs()

        // check arguments
        if p.Err() != nil {
            fmt.Println(p.Err())
            return
        }

        // request
        p.Do(context.Background())
    }
```

<!--- Deprecated
## Validator

It is a powerful-tool of verification,conversion and filter. So simply,good expansibility and good for using.

[click me to code](https://github.com/neo532/gofr/blob/master/inout/vcf.go)

```go
    package main

    import (
        "fmt"
        
        "github.com/neo532/gofr/inout"
    )

    func main() {
        //You can input parameters with one struct,one map of string or one by one.
        //The below is a method,inputting with one by one.
        vcf := inout.NewVCF(map[string]inout.Ido{
            "param_int1": inout.NewInt().IsGte(10).IsLte(90).InInt64(20),
            "param_str1": inout.NewStr("def1").IsGte(2).IsLte(5).InStr("string1"),
            "param_str2": inout.NewStr().RegExp(inout.Venum).InStr("str2"),
            "param_str3": inout.NewStr("def3").IsInMap(map[string]string{"a": "aVal"}).InStr("a"),
            "param_str4": inout.NewStr("def4").IsInArr("a", "b").InStr("a"),
            "param_str5": inout.NewStr().Slash().InStr(`\`),
            //...
        }).Do()

        if !vcf.IsOk() {
            fmt.Println(vcf.Err()) // param_str1:Length is too long!
            return
        }

        fmt.Println(vcf.Int64("param_int1"))  // 20
        fmt.Println(vcf.String("param_str1")) // def1
        fmt.Println(vcf.String("param_str2")) // str2
        fmt.Println(vcf.String("param_str3")) // aVal
        fmt.Println(vcf.String("param_str4")) // a
        fmt.Println(vcf.String("param_str5")) // "\\"
    }
```
-->

## Distributed lock
## Frequency controller
## Guard panic
