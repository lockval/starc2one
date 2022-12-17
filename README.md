## starc2one (starlark compile to a single file)
compile starlark source file(and the modules it depends on) to a single file OR execute compile file in REPL

### WHY
----------------------------
Sometimes you want a single starlark file to run in your host program, you can use it

### HOW
----------------------------
- compile a module to a single file :
```shell
starc2one -file main.star -output main.sbin
```

- compile src dir to a single file :
```shell
starc2one -file src -output main.sbin -suffix .star
```

- execute compile file in REPL (You can access all modules via module2exports) :
```shell
starc2one -file main.sbin
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

- There is a predeclared "globalThis" in starc2one. :
```go
predeclared["globalThis"] = &starlarkstruct.Module{Name: "testModule", Members: starlark.StringDict{"test": starlark.String("hello")}}
globals, err := program.Init(thread, predeclared)
```
```python
def test():
    print(globalThis.test)
```
- more help :
```shell
starc2one -help
```

### INSTALLATION
----------------------------
```shell
go install github.com/lockval/starc2one@latest
```

