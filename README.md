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

## HTTP request

It is a powerful-tool of request.It contains log/retry.

[click me to code](https://github.com/neo532/gofr/blob/master/request)

```go
    package main

    import (
        "github.com/neo532/gofr/request"
    )

    type Logger struct {
    }

    func (l *Logger) Log(c context.Context, statusCode int, curl string, limit time.Duration, cost time.Duration, resp []byte, err error) {
        var logMsg = fmt.Sprintf("[%s] [code:%+v] [limit:%+v] [cost:%+v] [%+v]",
                curl,
                statusCode,
                limit,
                cost,
                string(resp),
                )
            fmt.Println(logMsg)
    }

    type ReqParam struct {
        Directory string `form:"directory"`
    }

    type Body struct {
        Directory string `json:"directory"`
    }

    func main() {

        // register logger if it's necessary.
        request.RegLogger(&Logger{})

        // build args
        var p = request.HTTP{
            Method: "GET",
            URL:    "https://github.com/neo532/gofr",
            Limit:  time.Duration(3) * time.Second, // optional
            Retry:  1,                              // optional, default:1
        }.
        QueryArgs(&ReqParam{Directory: "request"}).                                // optional
        JsonBody(&Body{Directory: "request"}).                                     // optional
        Header(http.Header{"a": []string{"a1", "a2"}, "b": []string{"b1", "b2"}}). // optional
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

## Distributed lock

It is a distributed lock with signle instance by redis.

[click me to code](https://github.com/neo532/gofr/blob/master/tool)

```go
    package main

    import (
        "github.com/go-redis/redis"
        "github.com/neo532/gofr/tool"
    )

    type RedisOne struct {
        cache *redis.Client
    }

    func (l *RedisOne) Eval(c context.Context, cmd string, keys []string, args []interface{}) (rst interface{}, err error) {
        return l.cache.Eval(cmd, keys, args...).Result()
    }

    var Lock *tool.Lock

    func init(){

        var rdb := &RedisOne{
            redis.NewClient(&redis.Options{
                Addr:     "127.0.0.1:6379",
                Password: "password",
            })
        }

        var Lock = tool.NewLock(rdb)
    }

    func main() {

        var c = context.Background()
        var key = "IamAKey"
        var expire = time.Duration(10) * time.Second
        var wait = time.Duration(2) * time.Second

        code, err := Lock.Lock(c, key, expire, wait)
        Lock.UnLock(c, key, code)
    }
```

## Frequency controller

It is a frequency with signle instance by redis.

[click me to code](https://github.com/neo532/gofr/blob/master/tool)

```go
    package main

    import (
        "github.com/go-redis/redis"
        "github.com/neo532/gofr/tool"
    )

    type RedisOne struct {
        cache *redis.Client
    }

    func (l *RedisOne) Eval(c context.Context, cmd string, keys []string, args []interface{}) (rst interface{}, err error) {
        return l.cache.Eval(cmd, keys, args...).Result()
    }

    var Freq *tool.Freq

    func init(){

        var rdb := &RedisOne{
            redis.NewClient(&redis.Options{
                Addr:     "127.0.0.1:6379",
                Password: "password",
            })
        }

        var Freq = tool.NewFreq(rdb)
        Freq.Timezone("Local")
    }

    func main() {

        var c = context.Background()
        var preKey = "user.test"
        var rule = []tool.FreqRule{
            tool.FreqRule{Duri: "10000", Times: 80},
            tool.FreqRule{Duri: "day", Times: 5},
        }

        fmt.Println(Freq.IncrCheck(c, preKey, rule...))
        fmt.Println(Freq.Get(c, preKey, rule...))
    }
```

## Page Execute

It is a tool to page slice.

[click me to code](https://github.com/neo532/gofr/blob/master/tool)

```go
    package main

    import (
        "github.com/neo532/gofr/tool"
    )

    func main() {

        var arr = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

        tool.PageExec(len(arr), 3, func(b, e int) {
            fmt.Println(arr[b:e])
        })
        // [1 2 3] [4 5 6] [7 8 9] [10]
    }
```

## Guard panic
