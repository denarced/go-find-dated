package main

import (
	"fmt"
	"testing"
	"time"
)

func TestExtractDateFromFilepath_NoDate(t *testing.T) {
	_, success := extractDate("a/b/main.log")
	if success {
		t.Errorf("Date extraction should've failed but didn't")
	}
}

func TestExtractDateFromFilepath_ValidDate(t *testing.T) {
	expectedDate := time.Date(2018, 5, 5, 0, 0, 0, 0, time.UTC)
	filepath := fmt.Sprintf("a/b/main_%s.log", expectedDate.Format("2006-01-02"))
	actualDate, success := extractDate(filepath)
	if !success {
		t.Fatalf("Date extraction should've been successful")
	}
	if actualDate != expectedDate {
		t.Fatalf("Wrong date received. Expected: %v, got: %v", expectedDate, actualDate)
	}
}

func TestExtractDateFromFilepath_InvalidDate(t *testing.T) {
	date, success := extractDate("main_2018-02-30.log")
	if success {
		t.Fatalf("Extract should've failed but produced: %v", date)
	}
}

func TestExtractDateFromFilepath_MaxDate(t *testing.T) {
	_, success := extractDate("a/../hell-9999-12-31")
	if !success {
		t.Fatal("Extract should've succeeded")
	}
}

func TestExtractDateFromFilepath_MinDate(t *testing.T) {
	_, success := extractDate("a/../hell-0000-01-01")
	if !success {
		t.Fatal("Extract should've succeeded")
	}
}
