package main

import (
    "errors"
    "sync"
    "strings"
    "strconv"
    "os"

	"github.com/pelletier/go-toml"
)

type Config struct {
    NoTTY, Verbose bool
    ConnectionsPerBox int
    CallBackIps []string
    Target []Target
    Module []Module
}

type Target struct {
    Ips []string
    Modules []string
    Username, Password []string
    ReplaceX string
    Stealthy bool
    Port, Level int
}

type Instance struct {
    Id, Ip string
    Username, Password []string
    Stealthy bool
    Level, Port int
}

type Module struct {
    Name string
    Debug bool
    Enabled []string
}

type Script struct {
    Name string
    Level int
    IfState int
    RouletteState int
    OutputState int
}

var c = Config{}

func main() {
    InitLogger()


	// config handling
    if configData, err := os.ReadFile("coordinate.toml"); err != nil {

        Fatal(err)
    } else {
        err = toml.Unmarshal(configData, &c)
        if err != nil {
            Fatal(err)
        }
    }

    // distributes runner tasks among ips
    var wg sync.WaitGroup
    tid := 0
    for _, target := range c.Target {
        for _, ip := range target.Ips {
            ip = strings.ToLower(ip)
            if strings.Contains(ip, "x") {
                splitReplace := strings.Split(target.ReplaceX, "-")
                if len(splitReplace) != 2 {
                    Err("invalid format of ReplaceX", splitReplace)
                    break
                }
                minNum, err := strconv.Atoi(splitReplace[0])
                if err != nil {
                    Err("invalid range in ReplaceX", splitReplace)
                    break
                }
                maxNum, err := strconv.Atoi(splitReplace[1])
                if err != nil {
                    Err("invalid range in ReplaceX", splitReplace)
                    break
                }
                for i := minNum; i <= maxNum; i++ {
                    for _, m := range target.Modules {
                        i := Instance {
                            Id: strconv.Itoa(tid),
                            Ip: strings.Replace(ip, "x", strconv.Itoa(i), 1),
                            Port: 22,
                            Username: target.Username,
                            Password: target.Password,
                            Stealthy: target.Stealthy,
                        }
                        wg.Add(1)
                        tid++
                        go runner(i, moduleLookup(m), &wg)
                    }
                }
            } else {
                for _, m := range target.Modules {
                    i := Instance {
                        Id: strconv.Itoa(tid),
                        Ip: ip,
                        Port: 22,
                        Username: target.Username,
                        Password: target.Password,
                        Stealthy: target.Stealthy,
                    }
                    wg.Add(1)
                    tid++
                    go runner(i, moduleLookup(m), &wg)
                }
            }
        }
    }
    wg.Wait()
}

func moduleLookup(moduleName string) Module {
    for _, module := range c.Module {
        if module.Name == moduleName {
            return module
        }
    }
    Fatal(errors.New("invalid module name: " + moduleName))
    return Module{}
}
