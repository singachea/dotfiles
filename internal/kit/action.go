package kit

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ParseChoice(input string, next []Action) (picked []Action, quit bool, err error) {
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" || input == "q" || input == "quit" {
		return nil, true, nil
	}
	if input == "a" || input == "all" {
		for _, a := range next {
			if a.Runnable {
				picked = append(picked, a)
			}
		}
		if len(picked) == 0 {
			return nil, false, fmt.Errorf("nothing runnable")
		}
		return picked, false, nil
	}
	n, err := strconv.Atoi(input)
	if err != nil {
		return nil, false, fmt.Errorf("type a number, a, or q")
	}
	for _, a := range next {
		if a.Rank != n {
			continue
		}
		if !a.Runnable {
			return nil, false, fmt.Errorf("step %d cannot be run from here: %s", n, a.Do)
		}
		return []Action{a}, false, nil
	}
	return nil, false, fmt.Errorf("no step %d", n)
}

func RunAction(opt Options, a Action) (string, error) {
	switch a.ID {
	case "capture-skills":
		res, err := Capture(opt, CaptureFilter{Skills: true})
		return res.String(), err
	case "capture-plugins":
		res, err := Capture(opt, CaptureFilter{Plugins: true})
		return res.String(), err
	case "capture-settings":
		res, err := Capture(opt, CaptureFilter{Settings: true})
		return res.String(), err
	case "set-profile":
		home := opt.Home
		if home == "" {
			home = os.Getenv("HOME")
		}
		name := opt.Profile
		if name != "personal" && name != "work" {
			return "", fmt.Errorf("pick a profile: kit profile personal   or   kit profile work")
		}
		if err := WriteProfile(home, name); err != nil {
			return "", err
		}
		return "profile " + name + "\n", nil
	default:
		return "", fmt.Errorf("cannot run %s from doctor: %s", a.ID, a.Do)
	}
}
