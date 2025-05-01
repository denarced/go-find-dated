package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createDate(year, month, day int) *time.Time {
	d := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return &d
}

func TestExtractDate(t *testing.T) {
	run := func(name, filep string, expectedDate *time.Time, shouldSucceed bool) {
		t.Run(name, func(t *testing.T) {
			ass := assert.New(t)
			date, success := extractDate(filep)
			ass.Equal(shouldSucceed, success)
			if expectedDate != nil {
				ass.Equal(*expectedDate, date)
			}
		})
	}

	run("no date", "a/b/main.log", nil, false)
	run("valid date", "a/b/main_2018-05-05.log", createDate(2018, 5, 5), true)
	run("invalid date", "main_2018-02-30.log", nil, false)
	run("max", "a/../hell-9999-12-31", createDate(9999, 12, 31), true)
	run("min", "a/../hell-00000101", createDate(0, 1, 1), true)
	run("short at the end", "alpha-20000101", createDate(2000, 1, 1), true)
	run("invalid short at the end", "alpha-2000010", nil, false)
	run("long at the end", "omega-1999-12-31", createDate(1999, 12, 31), true)
	run("invalid long at the end", "omega-1999-12-3", nil, false)
}

func BenchmarkExtractDate(b *testing.B) {
	for range b.N {
		extractDate("finance-2024-01-02-2024-02-04-engineers.mp4")
	}
}
