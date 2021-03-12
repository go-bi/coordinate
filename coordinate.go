package main

import (
    "errors"
    "sync"
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
    //Ports []int
    Modules []string
    Username, Password string
}

type Instance struct {
    Id, Ip, Username, Password string
    Port int
}

type Module struct {
    Name string
    Stealthy bool
    UseRoulette bool
    Level int
    Enabled []string
    Disabled []string // cannot have both enabled and disabled
}

type Script struct {
    Name string
    Level int
    Options map[string]string
    // Source
    IfState int
    RouletteState int
    OutputState int
    Debug bool
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
            // if strings.Contains(ip, "x")
            // parse number range, function returns iterable
            // for each num
                // strings.Replace(ip, "x", num 
                for _, m := range target.Modules {
                    i := Instance {
                        Id: strconv.Itoa(tid),
                        Ip: ip,
                        Port: 22,
                        Username: target.Username,
                        Password: target.Password,
                    }
                    wg.Add(1)
                    tid++
                    go runner(i, moduleLookup(m), &wg)
                }

            // else
                //runner code again
        }
    //}
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
