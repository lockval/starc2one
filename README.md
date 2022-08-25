## starc2one (starlark compile to a single file)
compile starlark source file(and the modules it depends on) to a single file OR execute compile file in REPL

### WHY
----------------------------
Sometimes you want a single starlark file to run in your host program, you can use it

### HOW
----------------------------
- compile to a single file :
```shell
starc2one -file main.star -output main.sbin
```

- execute compile file in REPL (You can understand the usage and structure of this file here) :
```shell
starc2one -file main.sbin
```

- package multiple modules :
```text
load("path/module1.star","balabala")
load("path/module2.star","balabala")
load("path/module3.star","balabala")

save as "packages.star"
and then compile it
```

- load it in your golang program :
```go
file, err := os.OpenFile(*argFile, os.O_RDONLY, 0600)
check(err)
defer file.Close()
program, err := starlark.CompiledProgram(file)
check(err)
thread := &starlark.Thread{Name: "exec " + *argFile}
globals, err := program.Init(thread, nil)
check(err)
globals.Freeze()
```

- more help :
```shell
starc2one -help
```

### INSTALLATION
----------------------------
```shell
go install github.com/vanishs/starc2one@latest
```

