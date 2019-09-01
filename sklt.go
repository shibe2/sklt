// SKLT is a swaybar status program that outputs current keyboard layout and time.
// Sway has per-device layouts. This program outputs only the last layout that changed.
// When a new device is connected, its initial layout is shown.
// For command line reference, run:
//    sklt -h
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var progName = "sklt"

func usage(e bool) {
	var w io.Writer
	if e {
		w = os.Stderr
	} else {
		w = os.Stdout
	}
	fmt.Fprintln(w, "usage:", progName, "[-h] [-t interval] [-f format]")
	fmt.Fprintln(w, "\t-h - print this message and exit")
	fmt.Fprintln(w, "\t-t interval - time update interval; valid values are (case-insensitive):")
	fmt.Fprintln(w, "\t\ts or second")
	fmt.Fprintln(w, "\t\tm or minute (default)")
	fmt.Fprintln(w, "\t\th or hour")
	fmt.Fprintln(w, "\t\td or day (maximum is 1 hour, but \"day\" selects date-only format)")
	fmt.Fprintln(w, "\t-f format - time format as understood by Go time package")
	fmt.Fprintln(w, "\t\tthat is, how the time \"Mon Jan 2 15:04:05 -0700 MST 2006\" should be formatted")
	fmt.Fprintln(w, "\t\tsee https://golang.org/pkg/time/#Time.Format")
	fmt.Fprintln(w, "\t\texample: \"2006-01-02 15:04\" (year-month-day hour:minute)")
	if e {
		os.Exit(1)
	}
}

// Keyboards are organized in a doubly linked list in the order of recent layout changes.
type kbdDev struct{ layout, prevDev, nextDev string }

// Monitors Sway keyboard layouts.
type monitor struct {
	s                   net.Conn          // Sway IPC socket
	ch                  chan string       // layout change notifications
	kbds                map[string]kbdDev // indexed by device identifiers
	lastKbd, prevLayout string
}

// Delete a keyboard from the list.
func (self *monitor) del(id string) {
	if len(id) == 0 {
		return
	}
	if self.kbds == nil {
		return
	}
	l1 := self.kbds[id]
	delete(self.kbds, id)
	if len(l1.prevDev) > 0 {
		l2 := self.kbds[l1.prevDev]
		l2.nextDev = l1.nextDev
		self.kbds[l1.prevDev] = l2
	}
	if len(l1.nextDev) > 0 {
		l2 := self.kbds[l1.nextDev]
		l2.prevDev = l1.prevDev
		self.kbds[l1.nextDev] = l2
	}
	if self.lastKbd == id {
		self.lastKbd = l1.prevDev
	}
}

// Set keyboard's layout in the list. If the layout has changed or the identifier is new, put the keyboard at the end of the list.
func (self *monitor) set(id, l string) {
	if len(l) == 0 {
		self.del(id)
		return
	}
	if self.kbds == nil {
		self.kbds = make(map[string]kbdDev)
	}
	l1 := self.kbds[id]
	if l1.layout == l {
		return
	}
	l1.layout = l
	if self.lastKbd != id {
		if len(l1.prevDev) > 0 {
			l2 := self.kbds[l1.prevDev]
			l2.nextDev = l1.nextDev
			self.kbds[l1.prevDev] = l2
		}
		if len(l1.nextDev) > 0 {
			l2 := self.kbds[l1.nextDev]
			l2.prevDev = l1.prevDev
			self.kbds[l1.nextDev] = l2
		}
		l1.prevDev = self.lastKbd
		l1.nextDev = ""
		self.lastKbd = id
	}
	self.kbds[id] = l1
}

func (self *monitor) processMsg(t MessageType, payload io.Reader) error {
	switch t {
	case SUBSCRIBE:
		var p struct {
			Success bool `json:"success"`
		}
		err := json.NewDecoder(payload).Decode(&p)
		if err != nil {
			return err
		}
		if !p.Success {
			return errors.New("failed to subscribe to Sway events")
		}
		err = WriteEmptyMessage(self.s, GET_INPUTS)
		if err != nil {
			return err
		}
	case GET_INPUTS:
		var p []InputDevice
		err := json.NewDecoder(payload).Decode(&p)
		if err != nil {
			return err
		}
		for _, i := range p {
			self.set(i.Identifier, i.XkbActiveLayoutName)
		}
	case InputEvent:
		var p InputEventPayload
		err := json.NewDecoder(payload).Decode(&p)
		if err != nil {
			return err
		}
		switch p.Change {
		case "removed":
			self.del(p.Input.Identifier)
		default:
			self.set(p.Input.Identifier, p.Input.XkbActiveLayoutName)
		}
	}
	return nil
}

func (self *monitor) watchLayouts() {
	err := WriteJSONMessage(self.s, SUBSCRIBE, []string{"input"})
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to send Sway message:", err)
		os.Exit(1)
	}
	for {
		err = ReadMessage(self.s, self.processMsg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Sway IPC failure:", err)
			os.Exit(1)
		}
		nl := self.kbds[self.lastKbd].layout
		if self.prevLayout == nl {
			continue
		}
		self.prevLayout = nl
		self.ch <- nl
	}
}

func timer(interval time.Duration, ch chan<- time.Time) {
	for {
		t := time.Now()
		ch <- t
		dt := t.Truncate(interval).Add(interval).Sub(t)
		if dt > 0 {
			time.Sleep(dt)
		}
	}
}

func getArg(i *int) string {
	if *i > len(os.Args)-2 {
		fmt.Fprintln(os.Stderr, "missing value for the parameter", os.Args[*i])
		usage(true)
	}
	*i++
	return os.Args[*i]
}

func main() {
	if len(os.Args) > 0 {
		_, p := filepath.Split(os.Args[0])
		if len(p) > 0 {
			progName = p
		}
	}
	interval := time.Minute
	var format string
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-h", "--help":
			usage(false)
			return
		case "-t":
			ti := getArg(&i)
			switch strings.ToLower(ti) {
			case "s", "second":
				interval = time.Second
			case "m", "minute":
				interval = time.Minute
			case "h", "hour":
				interval = time.Hour
			case "d", "day":
				interval = 24 * time.Hour
			default:
				fmt.Fprintln(os.Stderr, "invalid interval:", ti)
				usage(true)
			}
		case "-f":
			format = getArg(&i)
		default:
			fmt.Fprintln(os.Stderr, "unknown parameter:", os.Args[i])
			usage(true)
		}
	}
	if len(format) == 0 {
		switch interval {
		case time.Second:
			format = "2006-01-02 15:04:05"
		case time.Minute:
			format = "2006-01-02 15:04"
		case time.Hour:
			format = "2006-01-02 15h"
		case 24 * time.Hour:
			format = "2006-01-02"
		}
	}
	if interval > time.Hour {
		interval = time.Hour
	}
	m := monitor{ch: make(chan string)}
	var err error
	m.s, err = Connect("")
	if err != nil && err != ErrNoIPC {
		fmt.Fprintln(os.Stderr, "failed to connect to Sway:", err)
		os.Exit(1)
	}
	if m.s != nil {
		go m.watchLayouts()
		format = " " + format
	}
	format += "\n"
	tch := make(chan time.Time, 1)
	go timer(interval, tch)
	var t time.Time
	var layout, prevStatus string
	for {
		select {
		case layout = <-m.ch:
			t = time.Now()
		case t = <-tch:
		}
		status := layout + t.Format(format)
		if status != prevStatus {
			n, err := io.WriteString(os.Stdout, status)
			if err == nil && n < len(status) {
				err = io.ErrShortWrite
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, "failed to output status line:", err)
				os.Exit(1)
			}
			prevStatus = status
		}
	}
}
