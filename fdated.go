package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sync"
	"time"
)

var re = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)

type specs struct {
	dirs  []string
	newer *time.Time
	older *time.Time
	today time.Time
}

func parseCli() (theSpecs specs) {
	var newer int
	flag.IntVar(&newer, "newer", -1, "days newer")
	flag.IntVar(&newer, "n", -1, "days newer")
	var older int
	flag.IntVar(&older, "older", -1, "days older")
	flag.IntVar(&older, "o", -1, "days older")
	var today string
	flag.StringVar(&today, "today", "", "")
	flag.StringVar(&today, "t", "", "")
	flag.Parse()

	if len(today) > 0 {
		parsedDate, err := time.Parse("2006-01-02", today)
		if err == nil {
			theSpecs.today = parsedDate
		}
	} else {
		now := time.Now()
		theSpecs.today = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	}

	if newer >= 0 {
		date := theSpecs.today.Add(time.Hour * time.Duration(24*-newer))
		theSpecs.newer = &date
	}
	if older >= 0 {
		date := theSpecs.today.Add(time.Hour * time.Duration(24*-older))
		theSpecs.older = &date
	}
	theSpecs.dirs = flag.Args()
	return
}

func extractDate(filepath string) (time.Time, bool) {
	dateString := re.FindString(filepath)
	if len(dateString) > 0 {
		parsed, err := time.Parse("2006-01-02", dateString)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func processFile(filepath string, theSpecs specs, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	date, success := extractDate(filepath)
	if !success {
		return
	}
	if theSpecs.older != nil && !date.Before(*theSpecs.older) {
		return
	}
	if theSpecs.newer != nil && !date.After(*theSpecs.newer) {
		return
	}
	fmt.Println(filepath)
}

func findInDir(dir string, theSpecs specs, externalWaitGroup *sync.WaitGroup, restrictor chan int) {
	defer externalWaitGroup.Done()
	<-restrictor
	restrictorFilled := false
	defer func() {
		if !restrictorFilled {
			restrictor <- 0
		}
	}()
	files, err := ioutil.ReadDir(dir)
	restrictor <- 0
	restrictorFilled = true
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading dir %v: %v\n", dir, err)
		return
	}
	var internalWaitGroup sync.WaitGroup
	for _, f := range files {
		full := path.Join(dir, f.Name())
		if f.IsDir() {
			internalWaitGroup.Add(1)
			go findInDir(full, theSpecs, &internalWaitGroup, restrictor)
		} else {
			internalWaitGroup.Add(1)
			go processFile(full, theSpecs, &internalWaitGroup)
		}
	}
	internalWaitGroup.Wait()
}

func find(theSpecs specs) {
	// Restrict the number of opened files in order to avoid "too many open
	// files" error that you might get when looking into directories with >1000
	// sub dirs. Performance wise 8 seems to be the optimal number. At least in
	// a single test with thousands of dirs and tens of thousands of files.
	size := 8
	restrictor := make(chan int, size)
	for i := 0; i < size; i++ {
		restrictor <- 0
	}

	var waitGroup sync.WaitGroup
	for _, each := range theSpecs.dirs {
		waitGroup.Add(1)
		go findInDir(each, theSpecs, &waitGroup, restrictor)
	}
	waitGroup.Wait()
}

func main() {
	find(parseCli())
}
