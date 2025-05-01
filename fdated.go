// Find files based on dates in filenames.
package main

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kong"
	"github.com/denarced/gent"
)

const (
	errorCodeConcurrencyNegative int = iota + 3
	errorCodeConcurrencyTooMuch
	errorCodeDir
)

var (
	newer *time.Time
	older *time.Time

	re = regexp.MustCompile(`(\d{4})-?(\d{2})-?(\d{2})`)
)

var CLI struct {
	Dirs        []string `arg:"" help:"Dirs to search in."`
	Concurrency int      `short:"c" default:"8" help:"Maximum concurrent file accesses."`

	NewerDays int       `short:"n" default:"-1" help:"Days newer. Ignored when <0."`
	OlderDays int       `short:"o" default:"-1" help:"Days older. Ignored when <0."`
	Today     time.Time `short:"t" type:"date" format:"2006-01-02"`
	Future    bool      `help:"Include dates that are in the future."`

	Format string `short:"f" help:"%t for date, and %p for filepath."`
}

func reduceDaysFromToday(days int) *time.Time {
	result := CLI.Today.Add(time.Hour * time.Duration(24*-days))
	return &result
}

func finishCliDates() {
	if CLI.Today.Equal(time.Time{}) {
		t := time.Now()
		CLI.Today = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}

	if CLI.NewerDays >= 0 {
		newer = reduceDaysFromToday(CLI.NewerDays)
	}
	if CLI.OlderDays >= 0 {
		older = reduceDaysFromToday(CLI.OlderDays)
	}
}

func extractDate(filep string) (t time.Time, success bool) {
	groups := re.FindStringSubmatch(filep)
	if groups == nil {
		return
	}
	dateString := strings.Join(groups[1:], "-")
	var err error
	t, err = time.Parse("2006-01-02", dateString)
	success = err == nil
	return
}

func processFile(filep string, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	date, success := extractDate(filep)
	if !success {
		return
	}
	if !CLI.Future && time.Now().Before(date) {
		return
	}
	if older != nil && !date.Before(*older) {
		return
	}
	if newer != nil && !date.After(*newer) {
		return
	}
	if CLI.Format == "" {
		fmt.Println(filep)
	} else {
		fmt.Print(formatOutput(filep, date))
	}
}

func formatOutput(filep string, date time.Time) string {
	s := strings.Replace(CLI.Format, "%t", date.Format("2006-01-02"), 1)
	s = strings.Replace(s, "%p", filep, 1)
	return gent.OrPanic2(strconv.Unquote(`"` + s + `"`))("convert backslash characters failed")
}

func findInDir(dir string, externalWaitGroup *sync.WaitGroup, restrictor chan int) {
	defer externalWaitGroup.Done()
	<-restrictor
	files, err := os.ReadDir(dir)
	restrictor <- 0
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading dir %v: %v\n", dir, err)
		return
	}
	var internalWaitGroup sync.WaitGroup
	for _, f := range files {
		full := path.Join(dir, f.Name())
		if f.IsDir() {
			internalWaitGroup.Add(1)
			go findInDir(full, &internalWaitGroup, restrictor)
		} else {
			internalWaitGroup.Add(1)
			go processFile(full, &internalWaitGroup)
		}
	}
	internalWaitGroup.Wait()
}

func find() {
	// Restrict the number of opened files in order to avoid "too many open
	// files" error that you might get when looking into directories with >1000
	// sub dirs. Performance wise 8 seems to be the optimal number. At least in
	// a single test with thousands of dirs and tens of thousands of files.
	size := CLI.Concurrency
	// Prefill buffer channel because otherwise initial goprocesses couldn't
	// start.
	restrictor := make(chan int, size)
	for i := 0; i < size; i++ {
		restrictor <- 0
	}

	var waitGroup sync.WaitGroup
	for _, each := range CLI.Dirs {
		waitGroup.Add(1)
		go findInDir(each, &waitGroup, restrictor)
	}
	waitGroup.Wait()
}

func checkDirs() error {
	for _, each := range CLI.Dirs {
		stat, err := os.Stat(each)
		if err != nil {
			return err
		}
		if !stat.IsDir() {
			return fmt.Errorf("not a directory: %s", each)
		}
	}
	return nil
}

func main() {
	kong.Parse(&CLI)
	concurrencyErrorMessage := "Invalid concurrency. Must be >=1 and <=CPU's core count."
	if CLI.Concurrency < 0 {
		fmt.Fprintln(os.Stderr, concurrencyErrorMessage)
		os.Exit(errorCodeConcurrencyNegative)
	}
	if CLI.Concurrency > runtime.NumCPU() {
		fmt.Fprintln(os.Stderr, concurrencyErrorMessage)
		os.Exit(errorCodeConcurrencyTooMuch)
	}
	if err := checkDirs(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid directory: %s\n", err)
		os.Exit(errorCodeDir)
	}
	finishCliDates()
	find()
}
