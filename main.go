package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type Item struct {
	from       string  // FooQueue
	methodname string  // 50%Enqueue50%Dequeue
	cpu        int     // cpu numbers
	timeop     float64 // ns/op
	timeopstr  string  // 32.23 (default ".2f")
	delta      int     // %
}

var (
	cpuFilter     = flag.Int("cpu", -1, "filter cpu numbers")
	methodFilter  = flag.String("method", "", "filter method name")
	removedPrefix = flag.String("prefix", "", "remove prefix string")
)

func main() {
	flag.Parse()
	if flag.Arg(0) == "" {
		panic("need bench file")
	}
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	cmd := exec.Command("bash", "-c", "benchstat -csv "+flag.Arg(0))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
	inputStr := stdout.String()
	// Parse csv.
	reader := csv.NewReader(bytes.NewReader([]byte(inputStr)))
	lines, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}
	if len(lines) <= 1 {
		panic(fmt.Sprintln("too few lines: ", len(lines)))
	}
	firstLine := lines[0]
	if firstLine[0] != "name" || firstLine[1] != "time/op (ns/op)" || firstLine[2] != "±" {
		panic(fmt.Sprintf("invalid first line: %v, want (%s)", len(lines), "name,time/op (ns/op),±"))
	}

	var allItems []*Item
	// Parse all data.
	for _, v := range lines[1:] {
		// Default/70Enqueue30Dequeue/LinkedQ-100,1.00808E+02,3%
		if len(v) != 3 {
			panic(fmt.Sprintf("invalid line: %v", v))
		}
		item := new(Item)
		name := v[0]
		// Find CPU numbers.
		nameFindSnake := strings.LastIndex(name, "-")
		if nameFindSnake != -1 {
			cpu, err := strconv.Atoi(name[nameFindSnake+1:])
			if err == nil {
				item.cpu = cpu
			}
		} else {
			item.cpu = 1
		}
		if nameFindSnake != -1 && item.cpu != 1 { // remove "-128", 128 is the CPU numbers
			name = name[:nameFindSnake]
		}
		// Find from.
		nameFindFrom := strings.LastIndex(name, "/")
		if nameFindFrom == -1 || nameFindFrom == len(name)-1 {
			panic(fmt.Sprintf("invalid name: %s", v[0]))
		}
		item.from = name[nameFindFrom+1:]
		item.methodname = name[:nameFindFrom]
		// Find timeop.
		timeop, err := strconv.ParseFloat(v[1], 64)
		if err != nil {
			panic(fmt.Sprintln("invalid time/op: ", v[1]))
		}
		item.timeop = timeop
		item.timeopstr = fmt.Sprintf("%.2f", timeop)
		delta, err := strconv.Atoi(v[2][:len(v[2])-1])
		if err != nil {
			panic(fmt.Sprintln("invalid delta: ", v[2]))
		}
		item.delta = delta
		// Add this item.
		allItems = append(allItems, item)
	}
	// Result.
	// Filter all items.
	filteredAllItems := make([]*Item, 0, len(allItems))
	for _, item := range allItems {
		if *cpuFilter >= 0 {
			if item.cpu != *cpuFilter {
				continue
			}
		}
		if *methodFilter != "" {
			if item.methodname != *methodFilter {
				continue
			}
		}
		if *removedPrefix != "" {
			item.methodname = strings.TrimPrefix(item.methodname, *removedPrefix)
		}
		filteredAllItems = append(filteredAllItems, item)
	}

	methods := make([]string, 0, 10)
	for _, item := range filteredAllItems {
		method := item.methodname
		if !inStringSlice(methods, method) {
			methods = append(methods, method)
		}
	}

	froms := make([]string, 0, 5)
	for _, item := range filteredAllItems {
		from := item.from
		if !inStringSlice(froms, from) {
			froms = append(froms, from)
		}
	}

	println(strings.Join(methods, " "))
	for _, from := range froms {
		output := from + " "
		for _, method := range methods {
			for _, item := range filteredAllItems {
				if item.from != from || item.methodname != method {
					continue
				}
				output += " " + item.timeopstr
			}
		}
		println(output)
	}
}

func inStringSlice(ss []string, s string) bool {
	for _, v := range ss {
		if s == v {
			return true
		}
	}
	return false
}
