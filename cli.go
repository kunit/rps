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
		return "?"
	}

	maj := uint64(proc.Stat.TtyNr)
	min := uint64(proc.Stat.TtyNr)
	maj = maj >> 8 & 0xfff
	min = min&0xff | (min&0xfff00000)>>12
	tmpmin := min

	s = "?"

	switch maj {
	case 3:
		if min <= 255 {
			p1 := tmpmin >> 4
			t0 := "pqrstuvwxyzabcde"[p1 : p1+1]
			p2 := tmpmin & 0x0f
			t1 := "0123456789abcdef"[p2 : p2+1]
			s = fmt.Sprintf("tty%s%s", t0, t1)
		}
	case 4:
		if min < 64 {
			s = fmt.Sprintf("tty%d", min)
		} else {
			s = fmt.Sprintf("ttyS%d", min-64)
		}
	case 11:
		s = fmt.Sprintf("ttyB%d", min)
	case 17:
		s = fmt.Sprintf("ttyH%d", min)
	case 19:
		s = fmt.Sprintf("ttyC%d", min)
	case 22:
		s = fmt.Sprintf("ttyD%d", min)
	case 23:
		s = fmt.Sprintf("ttyD%d", min)
	case 24:
		s = fmt.Sprintf("ttyE%d", min)
	case 32:
		s = fmt.Sprintf("ttyX%d", min)
	case 43:
		s = fmt.Sprintf("ttyI%d", min)
	case 46:
		s = fmt.Sprintf("ttyR%d", min)
	case 48:
		s = fmt.Sprintf("ttyL%d", min)
	case 57:
		s = fmt.Sprintf("ttyP%d", min)
	case 71:
		s = fmt.Sprintf("ttyF%d", min)
	case 75:
		s = fmt.Sprintf("ttyW%d", min)
	case 78:
		s = fmt.Sprintf("ttyM%d", min)
	case 105:
		s = fmt.Sprintf("ttyV%d", min)
	case 112:
		s = fmt.Sprintf("ttyM%d", min)
	case 136, 137, 138, 139, 140, 141, 142, 143:
		s = fmt.Sprintf("pts/%d", min+(maj-136)*256)
	case 148:
		s = fmt.Sprintf("ttyT%d", min)
	case 154:
		s = fmt.Sprintf("ttySR%d", min)
	case 156:
		s = fmt.Sprintf("ttySR%d", min+256)
	case 164:
		s = fmt.Sprintf("ttyCH%d", min)
	case 166:
		s = fmt.Sprintf("ttyACM%d", min)
	case 172:
		s = fmt.Sprintf("ttyMX%d", min)
	case 174:
		s = fmt.Sprintf("ttySI%d", min)
	case 188:
		s = fmt.Sprintf("ttyUSB%d", min)
	case 204:
		lowDensityNames := []string{
			"LU0", "LU1", "LU2", "LU3",
			"FB0",
			"SA0", "SA1", "SA2",
			"SC0", "SC1", "SC2", "SC3",
			"FW0", "FW1", "FW2", "FW3",
			"AM0", "AM1", "AM2", "AM3", "AM4", "AM5", "AM6", "AM7",
			"AM8", "AM9", "AM10", "AM11", "AM12", "AM13", "AM14", "AM15",
			"DB0", "DB1", "DB2", "DB3", "DB4", "DB5", "DB6", "DB7",
			"SG0",
			"SMX0", "SMX1", "SMX2",
			"MM0", "MM1",
			"CPM0", "CPM1", "CPM2", "CPM3", /* "CPM4", "CPM5", */ // bad allocation?
			"IOC0", "IOC1", "IOC2", "IOC3", "IOC4", "IOC5", "IOC6", "IOC7",
			"IOC8", "IOC9", "IOC10", "IOC11", "IOC12", "IOC13", "IOC14", "IOC15",
			"IOC16", "IOC17", "IOC18", "IOC19", "IOC20", "IOC21", "IOC22", "IOC23",
			"IOC24", "IOC25", "IOC26", "IOC27", "IOC28", "IOC29", "IOC30", "IOC31",
			"VR0", "VR1",
			"IOC84", "IOC85", "IOC86", "IOC87", "IOC88", "IOC89", "IOC90", "IOC91",
			"IOC92", "IOC93", "IOC94", "IOC95", "IOC96", "IOC97", "IOC98", "IOC99",
			"IOC100", "IOC101", "IOC102", "IOC103", "IOC104", "IOC105", "IOC106", "IOC107",
			"IOC108", "IOC109", "IOC110", "IOC111", "IOC112", "IOC113", "IOC114", "IOC115",
			"SIOC0", "SIOC1", "SIOC2", "SIOC3", "SIOC4", "SIOC5", "SIOC6", "SIOC7",
			"SIOC8", "SIOC9", "SIOC10", "SIOC11", "SIOC12", "SIOC13", "SIOC14", "SIOC15",
			"SIOC16", "SIOC17", "SIOC18", "SIOC19", "SIOC20", "SIOC21", "SIOC22", "SIOC23",
			"SIOC24", "SIOC25", "SIOC26", "SIOC27", "SIOC28", "SIOC29", "SIOC30", "SIOC31",
			"PSC0", "PSC1", "PSC2", "PSC3", "PSC4", "PSC5",
			"AT0", "AT1", "AT2", "AT3", "AT4", "AT5", "AT6", "AT7",
			"AT8", "AT9", "AT10", "AT11", "AT12", "AT13", "AT14", "AT15",
			"NX0", "NX1", "NX2", "NX3", "NX4", "NX5", "NX6", "NX7",
			"NX8", "NX9", "NX10", "NX11", "NX12", "NX13", "NX14", "NX15",
			"J0", // minor is 186
			"UL0", "UL1", "UL2", "UL3",
			"xvc0", // FAIL -- "xvc0" lacks "tty" prefix
			"PZ0", "PZ1", "PZ2", "PZ3",
			"TX0", "TX1", "TX2", "TX3", "TX4", "TX5", "TX6", "TX7",
			"SC0", "SC1", "SC2", "SC3",
			"MAX0", "MAX1", "MAX2", "MAX3",
		}
		if min < uint64(len(lowDensityNames)) {
			s = fmt.Sprintf("tty%.*s", len(lowDensityNames), lowDensityNames[min])
		}
	case 208:
		s = fmt.Sprintf("ttyU%d", min)
	case 216:
		s = fmt.Sprintf("ttyUB%d", min)
	case 224:
		s = fmt.Sprintf("ttyY%d", min)
	case 227:
		s = fmt.Sprintf("3270/tty%d", min)
	case 229:
		s = fmt.Sprintf("iseries/vtty%d", min)
	case 256:
		s = fmt.Sprintf("ttyEQ%d", min)
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
