package main

import (
    "errors"
    "os"
    "time"
    "bytes"
    "path/filepath"
    "bufio"
    "fmt"
    "sync"
    "math/rand"
    "strings"
    "encoding/base64"
    "strconv"

	"golang.org/x/crypto/ssh"
)


var (
    rouletteRoll int
    rouletteCounter int
)


const (
    // running states
    RUN_WAIT = iota
    RUN_ACTIVE
    RUN_ERROR
    RUN_END
)

const (
    // parsing states
    NONE = iota
    IF
    IF_FALSE
    ELSE
    IFCMD_ACTIVE
    ROULETTE_WAITING
    ROULETTE_TRUE
    ROULETTE_RAN
    OUTPUT_ACTIVE
)


func runner(i Instance, m Module, w *sync.WaitGroup) {

    defer w.Done()

    var files []string
    var wg sync.WaitGroup

    // TODO: insecure, allows path traversals
    // but only from those who can edit the modules


    // for each module, create list of all files
        rootPath := "../" + m.Name + "/"

        err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
            fileNameTmp := strings.TrimSpace(info.Name())
            if len(fileNameTmp) > 3 && fileNameTmp[len(fileNameTmp)-3:] == ".sh" {
                files = append(files, path)
            }
            return nil
        })

        if err != nil {
            Fatal(err)
        }


    if c.ConnectionsPerBox == 0 {
        c.ConnectionsPerBox = 3
    }

    fileChan := make(chan string)

    tid := 0
    for j := 0; j < c.ConnectionsPerBox && j < len(files); j++ {
        wg.Add(1)
        i.Id = string(i.Id[0]) + "-" + strconv.Itoa(tid)
        go ssher(fileChan, m, i, &wg)
        tid++
    }

    for _, file := range files {
        fileChan <- file
    }

    close(fileChan)

    wg.Wait()
}

func interpret(line string, lineNum int, i Instance, s *Script, m Module) (string, error) {

    line = strings.TrimSpace(line)
    if len(line) == 0 {
        return "", nil
    }

    // string replacements
    if strings.Contains(line, "#CALLBACK_IP") {
        rand.Seed(time.Now().UTC().UnixNano())
        callBack := c.CallBackIps[rand.Intn(len(c.CallBackIps))]
        strings.Replace(line, "#CALLBACK_IP", callBack, -1)
    }

    if m.Debug {
        InfoExtra(i, m, *s, lineNum, line)
    }

    firstChar := line[0]
    switch firstChar {
    case '#':
        splitLine := strings.Split(line, " ")
        if len(splitLine) == 0 {
            return "", nil
        }

        if s.IfState == IF_FALSE {
            if splitLine[0] != "#ELSE" && splitLine[0] != "#ENDIF" {
                if m.Debug {
                    Info("Skipping line due to being IF_FALSE:", splitLine)
                }
                return "", nil
            }
        }

        if s.Level < i.Level {
            // level too low, skip
            return "", errors.New("level below configured value")
        }

        switch splitLine[0] {
            case "#LEVEL":
                if len(splitLine) != 2 {
                    return "", lineError(s, lineNum, line, "malformed level directive")
                }

                if lvl, err := strconv.Atoi(splitLine[1]); err != nil {
                    return "", err
                } else {
                    s.Level = lvl
                }
            case "#IF":
                if len(splitLine) != 2 {
                    return "", lineError(s, lineNum, line, "malformed if")
                }
                if s.IfState != NONE {
                    return "", lineError(s, lineNum, line, "cannot nest if statements")
                }
                switch strings.ToLower(splitLine[1]) {
                case "stealthy":
                    if i.Stealthy {
                        s.IfState = IF
                    } else {
                        s.IfState = IF_FALSE
                    }
                default:
                    return "", lineError(s, lineNum, line, "invalid if variable")
                }
            case "#IFCMD":
                if len(splitLine) < 2 {
                    return "", lineError(s, lineNum, line, "malformed if")
                }
                if s.IfState != NONE {
                    return "", lineError(s, lineNum, line, "cannot nest if statements")
                }
                s.IfState = IFCMD_ACTIVE
                return strings.Join(splitLine[1:], " "), nil
            case "#ELSE":
                if s.IfState == IF {
                    s.IfState = IF_FALSE
                } else if s.IfState == IF_FALSE {
                    s.IfState = IF
                } else {
                    return "", lineError(s, lineNum, line, "unexpected else")
                }
            case "#ENDIF":
                if s.IfState == IF || s.IfState == IF_FALSE {
                    s.IfState = NONE
                } else {
                    return "", lineError(s, lineNum, line, "unexpected endif")
                }
            case "#STARTROULETTE":
                if len(splitLine) != 2 {
                    return "", lineError(s, lineNum, line, "malformed startroulette")
                }

                if s.RouletteState != NONE {
                    return "", lineError(s, lineNum, line, "cannot nest roulettes")
                }

                if rouletteMax, err := strconv.Atoi(splitLine[1]); err != nil {
                    return "", lineError(s, lineNum, line, "invalid number passed to startroulette")
                } else {
                    rand.Seed(time.Now().UTC().UnixNano())
                    rouletteRoll = rand.Intn(rouletteMax)
                    if m.Debug {
                        Info("Roulette roll is", rouletteRoll)
                    }
                }

                s.RouletteState = ROULETTE_WAITING
                rouletteCounter = 0
            case "#ROLL":
                if len(splitLine) != 2 {
                    return "", lineError(s, lineNum, line, "malformed roll")
                }

                if s.RouletteState != NONE {
                    return "", lineError(s, lineNum, line, "cannot nest roulettes")
                }

                if rouletteMax, err := strconv.Atoi(splitLine[1]); err != nil {
                    return "", lineError(s, lineNum, line, "invalid number passed to roll")
                } else {
                    if rouletteMax <= 0 {
                        return "", lineError(s, lineNum, line, "invalid number (too low) passed to roll")
                    }
                    rand.Seed(time.Now().UTC().UnixNano())
                    rouletteRoll = rand.Intn(rouletteMax)
                    if m.Debug {
                        Info("Roulette roll is", rouletteRoll)
                    }

                    if rouletteRoll == 0 {
                        if m.Debug {
                            Info("Roll was successful")
                        }
                        s.RouletteState = ROULETTE_TRUE
                    } else {
                        if m.Debug {
                            Info("Roll failed")
                        }
                        s.RouletteState = ROULETTE_RAN
                    }
                }
            case "#ROULETTE":
                // if it matches, do it
                if len(splitLine) != 1 {
                    return "", lineError(s, lineNum, line, "malformed roulette")
                }
                if s.RouletteState == NONE {
                    return "", lineError(s, lineNum, line, "roulette directive without startroulette")
                } else if s.RouletteState == ROULETTE_TRUE {
                    s.RouletteState = ROULETTE_RAN
                    if m.Debug {
                        Info("Roulette has concluded", rouletteRoll)
                    }
                } else if s.RouletteState == ROULETTE_WAITING && rouletteRoll == rouletteCounter {
                    if m.Debug {
                        Info("Roulette is now active")
                    }
                    s.RouletteState = ROULETTE_TRUE
                } else {
                    if m.Debug {
                        Info("Roulette was not chosen, roll was", rouletteRoll, "count is", rouletteCounter)
                    }
                }
                rouletteCounter++
            case "#ENDROULETTE":
                if s.RouletteState == ROULETTE_RAN {
                    s.RouletteState = NONE
                } else if s.RouletteState == ROULETTE_WAITING {
                    return "", lineError(s, lineNum, line, "roulette did not run, did you specify the correct number in startroulette?")
                } else if s.RouletteState == NONE {
                    return "", lineError(s, lineNum, line, "unexpected endroullete")
                }
            case "#DROP":
                if len(splitLine) != 3 {
                    return "", lineError(s, lineNum, line, "malformed drop")
                }

                // TODO: fix insecure file path handling
                filePath := "../" + m.Name + "/drops/" + splitLine[1]

                fileContent, err := os.ReadFile(filePath)
                if err != nil {
                    return "", lineError(s, lineNum, line, "invalid file specified to drop at " + filePath)
                }

                // base64 encode file contents
                encoded := base64.StdEncoding.EncodeToString([]byte(fileContent))
                return fmt.Sprintf("echo '%s' | base64 -d > %s", encoded, splitLine[2]), nil
            case "#OUTPUT":
                if len(splitLine) < 2 {
                    return "", lineError(s, lineNum, line, "malformed output")
                }
                s.OutputState = OUTPUT_ACTIVE
                return strings.Join(splitLine[1:], " "), nil
            case "#PRINT_RED":
                if len(splitLine) < 2 {
                    return "", lineError(s, lineNum, line, "malformed printred")
                }
                PrintRed(i, m, *s, strings.Join(splitLine[1:], " "))
            case "#PRINT_GREEN":
                if len(splitLine) < 2 {
                    return "", lineError(s, lineNum, line, "malformed printgreen")
                }
                PrintGreen(i, m, *s, strings.Join(splitLine[1:], " "))
            }
    default:
        if s.IfState == IF_FALSE {
            return "", nil
        }

        if s.RouletteState == ROULETTE_WAITING || s.RouletteState == ROULETTE_RAN {
            return "", nil
        }

        return line, nil
    }
    return "", nil
}


func connect(i Instance) (*ssh.Session, *ssh.Client, int, int, error) {
    var usernameIndex int
    var passwordIndex int
    for {
        rand.Seed(time.Now().UTC().UnixNano())
        jitter := rand.Intn(10)
        time.Sleep(time.Duration(jitter) * 100 * time.Millisecond)
        // SSH client config
        Info("[T" + i.Id + "] Trying", i.Username[usernameIndex]  + ":" + i.Password[passwordIndex], "for", i.Ip)
        config := &ssh.ClientConfig{
            User: i.Username[usernameIndex],
            Auth: []ssh.AuthMethod{
                ssh.Password(i.Password[passwordIndex]),
            },
            // We don't care about host keys
            HostKeyCallback: ssh.InsecureIgnoreHostKey(),
        }

        // Connect to host
        client, err := ssh.Dial("tcp", i.Ip+":"+strconv.Itoa(i.Port), config)
        if err != nil {
            Info("login failed lol")
        } else {

            // Create sesssion
            sess, err := client.NewSession()
            if err != nil {
                Info("session creation failed lol")
            } else {
                return sess, client, usernameIndex, passwordIndex, nil
            }
        }

        if usernameIndex == len(i.Username) - 1 {
            if passwordIndex == len(i.Password) - 1 {
                return &ssh.Session{}, client, usernameIndex, passwordIndex, err
            }
        }
        if passwordIndex == len(i.Password) - 1 {
            usernameIndex++
            passwordIndex = 0
        } else {
            passwordIndex++
        }
    }

}

func ssher(fileChan chan string, m Module, i Instance, wg *sync.WaitGroup) {

    defer wg.Done()
    s := Script{}

    sess, client, uIndex, pIndex, err := connect(i)
    if err != nil {
        Crit(i, m, s, "Failed to log in for usernames:", i.Username, ", passwords:", i.Password, "(" + err.Error() + ")")
        return
    }
    defer client.Close()
	defer sess.Close()

	// I/O for shell
	stdin, err := sess.StdinPipe()
	if err != nil {
		Err(err)
        return
	}

    var stdoutBytes bytes.Buffer
    var stderrBytes bytes.Buffer
    sess.Stdout = &stdoutBytes
    sess.Stderr = &stderrBytes

	// Start remote shell
	err = sess.Shell()
	if err != nil {
		Err(err)
        return
	}

    index := 1
    escalated := false
    stdoutOffset := 0
    stderrOffset := 0

    Info("[T" + i.Id + "] Logged in to", i.Ip)
    stdoutOffset = stdoutBytes.Len()
    stderrOffset = stderrBytes.Len()

    if i.Username[uIndex] != "root" {
        // escalate to root
        escalated = true
        time.Sleep(time.Second)
        fmt.Fprintf(stdin, "echo '%s' | sudo -S whoami\n", i.Password[pIndex])
        time.Sleep( 2 * time.Second)
        stderrOffset = stderrBytes.Len()
        fmt.Fprintf(stdin, "sudo -i\n")
        time.Sleep( 1 * time.Second)
        if stderrBytes.Len() - stderrOffset > 0 {
            Stderr(strings.TrimSpace(stderrBytes.String()))
            Crit(i, m, s, "Failed to escalate from", i.Username, "to root on", i.Ip)
            return
        }
        Info("Successfully elevated to root on", i.Ip)
    }

    for {

        // grab file
        fileName, ok := <-fileChan
        if !ok {
            return
        }

        s := Script{}

        // TODO improve this file handling
        // pass os.File from filechan?
        fileSplit := strings.Split(fileName, "/")
        s.Name = fileSplit[len(fileSplit)-1][:len(fileSplit[len(fileSplit)-1])-3]

        if len(m.Enabled) > 0 {
            enabled := false
            for _, modEn := range m.Enabled {
                if modEn == s.Name {
                    enabled = true
                    break
                }
            }
            if !enabled {
                continue
            }
        }

        // read file for module
        file, err := os.Open(fileName)
        if err != nil {
            Crit(i, m, s, errors.New("Error opening " + fileName + ": " + err.Error()))
            return
        }

        scanner := bufio.NewScanner(file)


        for scanner.Scan() {
            index++

            // if IFCMD ran, get output
            if s.IfState == IFCMD_ACTIVE {
                // TODO: will fail for fish and weird shells
                stdoutOffset = stdoutBytes.Len()
                fmt.Fprintf(stdin, "echo $?\n")

                // replace with shell-reading construct
                // read status from last cmd
                time.Sleep(10 * time.Second)

                cmdOutput := strings.TrimSpace(stdoutBytes.String()[stdoutOffset:])
                cmdResult, err := strconv.Atoi(cmdOutput)
                if err != nil {
                    Crit(i, m, s, "Failed to read result from #IFCMD for", i.Ip, "line", index-1)
                    s.IfState = IF_FALSE
                } else if cmdResult == 0 {
                    s.IfState = IF
                    if m.Debug {
                        Info("IFCMD passed, cmdResult was", cmdResult)
                    }
                } else {
                    s.IfState = IF_FALSE
                    if m.Debug {
                        Info("IFCMD failed, cmdResult was", cmdResult)
                    }
                }
            }


            line, err := interpret(scanner.Text(), index, i, &s, m)
            if err != nil {
                Crit(i, m, s, errors.New("Error: " + fileName + ": " + err.Error()))
                return
            }

            if line == "" {
                continue
            }

            // workaround while i dont have a real stdout/stderr waiter
            if s.OutputState == OUTPUT_ACTIVE {
                time.Sleep(2 * time.Second)
                stdoutOffset = stdoutBytes.Len()
            }

            _, err = fmt.Fprintf(stdin, "%s\n", line)
            if err != nil {
                Crit(i, m, s, errors.New("Error submitting line to stdin: " + err.Error()))
                return
            }


            if m.Debug {
                time.Sleep(1 * time.Second);
                if stdoutBytes.Len() - stdoutOffset > 0 {
                    Stdout(strings.TrimSpace(stdoutBytes.String()[stdoutOffset:]))
                }
            }

            if s.OutputState == OUTPUT_ACTIVE {
                s.OutputState = NONE

                // replace with shell-reading construct
                // read status from last cmd
                time.Sleep(2 * time.Second)

                if stdoutBytes.Len() - stdoutOffset > 0 {
                    cmdOutput := strings.TrimSpace(stdoutBytes.String()[stdoutOffset:])
                    PrintGreen(i, m, s, cmdOutput)
                } else {
                    Crit(i, m, s, errors.New("Failed to get output for #OUTPUT"))
                }
            }

            if stderrBytes.Len() - stderrOffset > 0 {
                Stderr(strings.TrimSpace(stderrBytes.String()[stderrOffset:]))
            }

            stdoutOffset = stdoutBytes.Len()
            stderrOffset = stderrBytes.Len()
        }

        if err := scanner.Err(); err != nil {
            Crit(i, m, s, errors.New("scanner error: " + err.Error()))
        }
        InfoExtra(i, m, s, "Finished running script")
    }
    _, err = fmt.Fprintf(stdin, "logout\n")
    if err != nil {
        Crit(i, m, s, errors.New("Error submitting logout command: " + err.Error()))
    }

    if escalated {
        _, err = fmt.Fprintf(stdin, "logout\n")
        if err != nil  {
            Crit(i, m, s, errors.New("Error submitting logout command: " + err.Error()))
        }
    }

    // Wait for sess to finish with timeout
    errChan := make(chan error)
    go func() {
        errChan <- sess.Wait()
    }()

    select {
    case <-errChan:
    case <-time.After(30 * time.Second):
        Err("shell close wait timed out");
        sess.Close()
    }

}

func lineError(s *Script, lineNum int, line string, err string) error {
    return errors.New(s.Name + ": " + "line " + strconv.Itoa(lineNum) + ": " + err + ": " + line)
}

func shellWaitStdout() (string, error) {
    // select
    // go func to busy wait & read from stdout new characters
        // if new chars, return yeet
    // timeout 
    return "", errors.New("Timeout waiting for Stdout")
}

