# Go-Dump

Go-Dump is a package which helps you to dump a struct to `SdtOut`, any `io.Writer`, or a `map[string]string`.

[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/fsamin/go-dump) [![Build Status](https://travis-ci.org/fsamin/go-dump.svg?branch=master)](https://travis-ci.org/fsamin/go-dump) [![Go Report Card](https://goreportcard.com/badge/github.com/fsamin/go-dump)](https://goreportcard.com/report/github.com/fsamin/go-dump)

## Sample usage

````golang
type T struct {
    A int
    B string
}

a := T{23, "foo bar"}

dump.FDump(out, a)
````

Will print

````bash
T.A: 23
T.B: foo bar
````

## Usage with a map

```golang
type T struct {
    A int
    B string
}

a := T{23, "foo bar"}

m, _ := dump.ToMap(a)
```

Will return such a map:

| KEY           | Value         |
| ------------- | ------------- |
| T.A           | 23            |
| T.B           | foo bar       |

## Formatting keys

```golang
    dump.ToMap(a, dump.WithDefaultLowerCaseFormatter())
```

## Using go-dump to manage environment variables and using spf13/viper
```golang
    
    type MyStruct struct {
        A string
        B struct {
            InsideB string
        }
    }

    var myStruct MyStruct
    myStruct.A = "value A"
    myStruct.B.InsideB = "value B"
    

    dumper := dump.NewDefaultEncoder()
    dumper.DisableTypePrefix = true
    dumper.Separator = "_"
    dumper.Prefix = "MYSTRUCT"
    dumper.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultUpperCaseFormatter()}

    envs, _ := dumper.ToStringMap(&myStruct) // envs is the map of the dumped MyStruct 
```

Will return such a map:

| KEY                    | Value         |
| ---------------------- | ------------- |
| MYSTRUCT_A             | value A       |
| MYSTRUCT_B_INSIDEB     | value B       |

The environement variables can be handled by **viper** [spf13/viper](https://github.com/spf13/viper).

```golang
    ...
    for k := range envs {
        viper.BindEnv(dumper.ViperKey(k), k)
    }
    
    ...

    viperSettings := viper.AllSettings()
    for k, v := range viperSettings {
        fmt.Println(k, v)
    }
    ...
```

## More examples

See [unit tests](dump_test.go) for more examples.

## Dependencies

Go-Dump needs Go >= 1.8

No external dependencies :)
