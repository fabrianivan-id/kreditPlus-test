package mysql

import (
	"testing"

	driver "github.com/go-sql-driver/mysql"
)

func TestIsDuplicate(t *testing.T) {
	dup := &driver.MySQLError{Number: 1062}
	if !isDuplicate(dup) {
		t.Fatalf("expected duplicate error to be detected")
	}

	notDup := &driver.MySQLError{Number: 1215}
	if isDuplicate(notDup) {
		t.Fatalf("expected non-duplicate error to be false")
	}
}
