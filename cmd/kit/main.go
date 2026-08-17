package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/singachea/dotfiles/internal/kit"
)

func main() {
	args := os.Args[1:]
	opts := kit.Options{}
	asJSON := false
	noColor := false
	noOpen := false
	outPath := ""
	cmd := "help"
	helpFlag := false
	capFilter := kit.CaptureFilter{}
	noIgnore := false
	noPrompt := false

	var positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--json":
			asJSON = true
		case a == "--no-color":
			noColor = true
		case a == "--no-open":
			noOpen = true
		case a == "--skills":
			capFilter.Skills = true
		case a == "--plugins":
			capFilter.Plugins = true
		case a == "--settings":
			capFilter.Settings = true
		case a == "--no-ignore":
			noIgnore = true
		case a == "--no-prompt":
			noPrompt = true
		case a == "--root" && i+1 < len(args):
			i++
			opts.Root = args[i]
		case strings.HasPrefix(a, "--root="):
			opts.Root = strings.TrimPrefix(a, "--root=")
		case a == "--home" && i+1 < len(args):
			i++
			opts.Home = args[i]
		case strings.HasPrefix(a, "--home="):
			opts.Home = strings.TrimPrefix(a, "--home=")
		case a == "--profile" && i+1 < len(args):
			i++
			opts.Profile = args[i]
		case strings.HasPrefix(a, "--profile="):
			opts.Profile = strings.TrimPrefix(a, "--profile=")
		case a == "--out" && i+1 < len(args):
			i++
			outPath = args[i]
		case strings.HasPrefix(a, "--out="):
			outPath = strings.TrimPrefix(a, "--out=")
		case a == "-h" || a == "--help":
			helpFlag = true
		case strings.HasPrefix(a, "-"):
			fatal("unknown flag %s", a)
		default:
			positionals = append(positionals, a)
		}
	}
	if helpFlag {
		cmd = "help"
	} else if len(positionals) > 0 {
		cmd = positionals[0]
		if cmd == "profile" && opts.Profile == "" && len(positionals) > 1 {
			opts.Profile = positionals[1]
		}
	}

	switch cmd {
	case "help":
		fmt.Print(usage())
	case "status":
		rep := mustScan(opts)
		if asJSON {
			writeJSON(rep)
			return
		}
		fmt.Print(kit.FormatText(rep, wantColor(noColor)))
	case "doctor":
		rep := mustScan(opts)
		if asJSON {
			writeJSON(rep.Actions)
			return
		}
		color := wantColor(noColor)
		fmt.Print(kit.FormatDoctor(rep, color))
		if noPrompt || asJSON || !isTTY() {
			return
		}
		runDoctorPrompt(opts, color)
	case "recommend", "rec":
		rep := mustScan(opts)
		if asJSON {
			writeJSON(rep.Actions)
			return
		}
		fmt.Print(kit.FormatRecommend(rep, wantColor(noColor)))
	case "fix":
		rep := mustScan(opts)
		if asJSON {
			writeJSON(rep.Actions)
			return
		}
		fmt.Print(kit.FormatFix(rep))
	case "board":
		rep := mustScan(opts)
		path, err := kit.WriteBoard(rep, outPath)
		if err != nil {
			fatal("%v", err)
		}
		fmt.Println(path)
		if !noOpen {
			if err := kit.OpenBoard(path); err != nil {
				fmt.Fprintf(os.Stderr, "open board: %v\n", err)
			}
		}
	case "profile":
		home := opts.Home
		if home == "" {
			home = os.Getenv("HOME")
		}
		name := opts.Profile
		if name == "" {
			got, err := kit.ReadProfile(home)
			if err != nil {
				fatal("%v", err)
			}
			if got == "" {
				fmt.Println("unset")
				return
			}
			fmt.Println(got)
			return
		}
		if err := kit.WriteProfile(home, name); err != nil {
			fatal("profile must be personal or work")
		}
		fmt.Println(name)
	case "capture":
		res, err := kit.Capture(opts, capFilter)
		if err != nil {
			fatal("%v", err)
		}
		fmt.Print(res)
	case "remove", "rm":
		names := []string{}
		if len(positionals) > 1 {
			names = positionals[1:]
		}
		res, err := kit.Remove(opts, names, kit.RemoveOpts{NoIgnore: noIgnore})
		if err != nil {
			fatal("%v", err)
		}
		fmt.Print(res)
	case "apply", "pull", "push", "sync":
		fmt.Fprintf(os.Stderr, "kit %s is not implemented yet.\n", cmd)
		os.Exit(2)
	default:
		fatal("unknown command %s\n\n%s", cmd, usage())
	}
}

func mustScan(opts kit.Options) kit.Report {
	rep, err := kit.Scan(opts)
	if err != nil {
		fatal("%v", err)
	}
	return rep
}

func writeJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func wantColor(noColor bool) bool {
	return !noColor && os.Getenv("NO_COLOR") == ""
}

func usage() string {
	return `kit — inspect AI coding-tool homes

  kit doctor              what to do next (start here)
  kit recommend           same list, with item names
  kit fix                 copy-paste commands, in order
  kit status              full inventory
  kit board               tabbed HTML of the same report
  kit capture [--skills] [--plugins] [--settings]
  kit remove <skill> [skill...]   drop from the kit (and ignore)
  kit profile [personal|work]

  --root PATH   kit repo
  --skills --plugins --settings   limit capture (default: all)
  --home PATH   fake $HOME
  --json        machine-readable
  --no-color
  --no-prompt   doctor: print only, do not ask
  --no-open     (board) write HTML only
  --out FILE    (board) output path

capture / apply / pull / push / sync are not implemented yet.
`
}

func runDoctorPrompt(opts kit.Options, color bool) {
	in := bufio.NewScanner(os.Stdin)
	for {
		rep, err := kit.Scan(opts)
		if err != nil {
			fatal("%v", err)
		}
		next, _ := kit.SplitActions(rep.Actions)
		if kit.CountRunnable(next) == 0 {
			return
		}
		fmt.Print("> ")
		if !in.Scan() {
			fmt.Println()
			return
		}
		picked, quit, err := kit.ParseChoice(in.Text(), next)
		if err != nil {
			fmt.Fprintf(os.Stderr, "kit: %v\n", err)
			continue
		}
		if quit {
			return
		}
		for _, a := range picked {
			if a.ID == "set-profile" && opts.Profile == "" {
				fmt.Print("profile [personal/work]: ")
				if !in.Scan() {
					return
				}
				opts.Profile = strings.TrimSpace(in.Text())
			}
			fmt.Fprintf(os.Stderr, "running %s\n", a.Title)
			out, err := kit.RunAction(opts, a)
			fmt.Print(out)
			if err != nil {
				fmt.Fprintf(os.Stderr, "kit: %v\n", err)
			}
		}
		rep, err = kit.Scan(opts)
		if err != nil {
			fatal("%v", err)
		}
		fmt.Print(kit.FormatDoctor(rep, color))
	}
}

func isTTY() bool {
	st, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "kit: "+format+"\n", args...)
	os.Exit(1)
}
