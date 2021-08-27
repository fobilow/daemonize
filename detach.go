package detach

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const pidFileSuffix = "detach.pid"

var (
	flagName  string
	flagSet   *flag.FlagSet
	flagValue *string
)

type Process struct {
	Pid       int
	Args      []string
	StartTime time.Time
}

func (p Process) PidFile() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("%d_%s", p.Pid, pidFileSuffix))
}

func (p Process) String() string {
	return fmt.Sprintf(
		"PID: %d\nPID File: %s \nArgs: %s\nStarted: %s\nDuration: %s",
		p.Pid,
		p.PidFile(),
		strings.Join(p.Args, " "),
		p.StartTime,
		time.Since(p.StartTime),
	)
}

func Setup(name string, set *flag.FlagSet) func() {
	flagSet = set
	if flagSet == nil {
		flagSet = flag.CommandLine
	}
	flagName = name
	flagValue = flagSet.String(name, "", "start, status, stop or restart")

	var detachFlagFound bool
	for _, a := range os.Args {
		if a == "-"+flagName || a == "--"+flagName {
			detachFlagFound = true
		}
	}

	if detachFlagFound {
		if err := parse(); err != nil {
			fmt.Println("ERROR: parse error:", err)
		}
		os.Exit(0)
	}
	// nothing to do, return clean up function
	return func() {
		cleanup()
	}
}

func cleanup() {
	d := Process{Pid: os.Getpid()}
	if err := os.Remove(d.PidFile()); err != nil {
		fmt.Println("ERROR: cleanup:", err)
	}
}

func parse() error {
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		return err
	}

	var err error
	switch *flagValue {
	case "start":
		err = start()
	case "status":
		err = status()
	case "stop":
		err = stop()
	case "restart":
		err = restart()
	default:
		err = errors.New("invalid detach option")
		flagSet.Usage()
	}
	return err
}

func start() error {
	var args []string
	var found bool
	for _, a := range os.Args {
		if a != "-"+flagName && a != "--"+flagName {
			if !found {
				args = append(args, a)
			} else {
				found = false
			}
		} else {
			found = true
		}
	}

	fmt.Println("running in detached mode")
	var sysproc = &syscall.SysProcAttr{} // Noctty: true
	var attr = os.ProcAttr{
		Dir: ".",
		Env: os.Environ(),
		Files: []*os.File{
			os.Stdin,
			nil,
			nil,
		},
		Sys: sysproc,
	}

	startTime := time.Now()
	process, err := os.StartProcess(os.Args[0], args, &attr)
	if err != nil {
		return err
	}

	fmt.Println("Flags:", flag.Args())
	p := Process{
		Pid:       process.Pid,
		Args:      args,
		StartTime: startTime,
	}

	data, err := json.Marshal(p)
	if err != nil {
		return process.Kill()
	}

	// write PID file
	if err := os.WriteFile(p.PidFile(), data, os.ModePerm); err != nil {
		return process.Kill()
	}

	return process.Release()
}

func status() error {
	processes := findAllProcesses()
	if len(processes) == 0 {
		fmt.Println("no processes running")
		return nil
	}
	// print some useful info
	for i, d := range processes {
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("Process #%d\n", i+1)
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println(d)
	}

	return nil
}

func stop() error {
	processes := findAllProcesses()
	if len(processes) == 0 {
		fmt.Println("no processes running")
		return nil
	}

	var errs []string
	for _, d := range processes {
		err := killProcess(d)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, ","))
	}
	fmt.Println("processes stopped")
	return nil
}

func restart() error {
	if err := stop(); err != nil {
		return err
	}

	return start()
}

func findAllProcesses() []Process {
	de, err := os.ReadDir(os.TempDir())
	if err != nil {
		return nil
	}

	var processes []Process
	for _, entry := range de {
		var d Process
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), pidFileSuffix) {
			f, err := os.Open(filepath.Join(os.TempDir(), entry.Name()))
			if err != nil {
				fmt.Println("ERROR:", err)
				continue
			}
			if err := json.NewDecoder(f).Decode(&d); err != nil {
				fmt.Println("ERROR:", err)
				continue
			}
			processes = append(processes, d)
		}
	}

	return processes
}

func killProcess(d Process) error {
	process, err := os.FindProcess(d.Pid)
	if err != nil {
		return err
	}

	if err := os.Remove(d.PidFile()); err != nil {
		return err
	}

	return process.Kill()
}
