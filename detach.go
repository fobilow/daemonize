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
	flagName string
	flagSet  *flag.FlagSet
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
	flagSet.String(name, "", "start, status, stop or restart")

	var detachFlagFound bool
	for _, a := range os.Args {
		if strings.TrimLeft(a, "-") == flagName {
			detachFlagFound = true
			break
		}
	}

	if detachFlagFound {
		if err := run(); err != nil {
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

func run() error {
	action, args := parse()
	var err error
	switch action {
	case "start":
		err = start(args)
	case "status":
		err = status()
	case "stop":
		err = stop()
	case "restart":
		err = restart(args)
	default:
		err = errors.New("invalid detach option")
		flagSet.Usage()
	}
	return err
}

func parse() (action string, args []string) {
	var found bool
	for _, a := range os.Args {
		if strings.TrimLeft(a, "-") == flagName {
			found = true
			continue
		}

		if found {
			action = a
			found = false
			continue
		}

		args = append(args, a)
	}
	return
}

func start(args []string) error {
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

func restart(args []string) error {
	if err := stop(); err != nil {
		return err
	}

	return start(args)
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
