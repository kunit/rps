package rps

import (
	"bytes"
	"fmt"
	"github.com/kunit/rprocs/proc"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/dghubble/sling"
	"github.com/jessevdk/go-flags"
)

const (
	// ExitOK for exit code
	ExitOK int = 0

	// ExitErr for exit code
	ExitErr int = 1
)

type cli struct {
	env     Env
	args    []string
	Hosts   string `long:"hosts" short:"H" description:"Connect remote hosts"`
	Help    bool   `long:"help" short:"h" description:"show this help message and exit"`
	Version bool   `long:"version" short:"v" description:"prints the version number"`
}

// Env struct
type Env struct {
	Out, Err io.Writer
	Args     []string
	Version  string
}

// RunCLI runs as cli
func RunCLI(env Env) int {
	cli := &cli{env: env}
	return cli.run()
}

func (c *cli) buildHelp(names []string) []string {
	var help []string
	t := reflect.TypeOf(cli{})

	for _, name := range names {
		f, ok := t.FieldByName(name)
		if !ok {
			continue
		}

		tag := f.Tag
		if tag == "" {
			continue
		}

		var o, a string
		if a = tag.Get("arg"); a != "" {
			a = fmt.Sprintf("=%s", a)
		}
		if s := tag.Get("short"); s != "" {
			o = fmt.Sprintf("-%s, --%s%s", tag.Get("short"), tag.Get("long"), a)
		} else {
			o = fmt.Sprintf("--%s%s", tag.Get("long"), a)
		}

		desc := tag.Get("description")
		if i := strings.Index(desc, "\n"); i >= 0 {
			var buf bytes.Buffer
			buf.WriteString(desc[:i+1])
			desc = desc[i+1:]
			const indent = "                        "
			for {
				if i = strings.Index(desc, "\n"); i >= 0 {
					buf.WriteString(indent)
					buf.WriteString(desc[:i+1])
					desc = desc[i+1:]
					continue
				}
				break
			}
			if len(desc) > 0 {
				buf.WriteString(indent)
				buf.WriteString(desc)
			}
			desc = buf.String()
		}
		help = append(help, fmt.Sprintf("  %-40s %s", o, desc))
	}

	return help
}

func (c *cli) showHelp() {
	opts := strings.Join(c.buildHelp([]string{
		"Hosts",
	}), "\n")

	help := `
Usage: rps [--version] [--help] <options>

Options:
%s
`
	fmt.Fprintf(c.env.Out, help, opts)
}

func (c *cli) run() int {
	p := flags.NewParser(c, flags.PassDoubleDash)
	_, err := p.ParseArgs(c.env.Args)
	if err != nil || c.Help {
		c.showHelp()
		return ExitErr
	}

	if c.Version {
		fmt.Fprintf(c.env.Err, "rps version %s\n", c.env.Version)
		return ExitOK
	}

	if c.Hosts == "" {
		fmt.Fprintf(c.env.Err, "host required\n")
		c.showHelp()
		return ExitErr
	}

	fmt.Println("HOST            USER       PID %CPU %MEM    VSZ    RSS TTY      STAT START   TIME COMMAND")

	hosts := strings.Split(c.Hosts, ",")
	for _, host := range hosts {
		procs := &proc.Procs{}
		_, err := sling.New().Get("http://" + host + "/v1/proc").ReceiveSuccess(procs)
		if err != nil {
			fmt.Fprintf(c.env.Err, "invalid host = %v, error = %v\n", host, err)
			return ExitErr
		}

		sp := strings.Split(host, ":")
		for _, proc := range procs.Procs {
			userStr := buildUser(proc)
			ttyStr := buildTty(proc)
			stateStr := buildState(proc)
			startStr := buildStart(proc)
			timeStr := buildTime(proc)
			cmdStr := buildCmd(proc)

			fmt.Printf("%-15s %-8s %6d %3s  %3s %6d %6d %-8s %-4s %-7s %5s %s\n", sp[0], userStr, proc.Stat.Pid, proc.Cpu, proc.Memory, proc.Status.VmSize, proc.Status.VmRSS, ttyStr, stateStr, startStr, timeStr, cmdStr)
		}
	}

	return ExitOK
}

// builduser USER string
func buildUser(proc proc.Proc) string {
	s := proc.UserName
	if len(s) > 8 {
		s = fmt.Sprintf("%s+", s[:7])
	}

	return s
}

// buildTty TTY string
func buildTty(proc proc.Proc) string {
	s := strconv.FormatInt(proc.Stat.TtyNr, 10)
	if s == "0" {
		s = "?"
	}

	return s
}

// buildState STAT string
func buildState(proc proc.Proc) string {
	s := proc.Stat.State
	if proc.Stat.Nice < 0 {
		s = s + "<"
	}
	if proc.Stat.Nice > 0 {
		s = s + "N"
	}
	if proc.Status.VmLck != 0 {
		s = s + "L"
	}
	if proc.Stat.Session == proc.Status.Tgid {
		s = s + "s"
	}
	if proc.Stat.NumThreads > 1 {
		s = s + "l"
	}
	if proc.Stat.Pgrp == proc.Stat.Tpgid {
		s = s + "+"
	}

	return s
}

// buildStart START string
func buildStart(proc proc.Proc) string {
	t := time.Now()
	t = t.Truncate(time.Hour).Add(- time.Duration(t.Hour()) * time.Hour)
	s := time.Unix(proc.Start, 0).Format("15:04")
	if proc.Start < t.Unix() {
		s = time.Unix(proc.Start, 0).Format("01/02")
	}

	return s
}

// buildTime TIME string
func buildTime(proc proc.Proc) string {
	return time.Unix(proc.Time, 0).Format("4:05")
}

// buildCmd COMMAND string
func buildCmd(proc proc.Proc) string {
	s := strings.Join(proc.Cmdline.Args, " ")
	if s == "" {
		s = fmt.Sprintf("[%s]", proc.Status.Name)
	}

	return s
}
