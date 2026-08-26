package database_test

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"fyp/database"

	"github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Every other test in this repo builds its own pq.Error to drive these checks,
// which proves the branch is wired up but never that the name in the branch is
// the name Postgres actually reports. That gap let "staff_user_id_key" sit in
// the code while the database raised "fyp_fuli_staff_user_id_key": the check
// silently never matched, and the raw driver error reached the user instead of
// the validation message. This suite closes it by deriving the real names from
// schema.sql — the same file the database is built from — and checking every
// constant against them.
var _ = Describe("Constraint names", func() {
	var known map[string]bool

	BeforeEach(func() {
		schema, err := os.ReadFile("schema.sql")
		Expect(err).NotTo(HaveOccurred())
		known = constraintNamesIn(string(schema))
		Expect(known).NotTo(BeEmpty())
	})

	It("names a constraint that schema.sql actually creates", func() {
		for _, name := range []string{
			database.ConstraintUserEmail,
			database.ConstraintStaffUserID,
			database.ConstraintBusinessOwner,
			database.ConstraintBusinessOwnerRef,
			database.ConstraintServiceName,
			database.ConstraintServiceOptionName,
			database.ConstraintServiceOptionItemName,
			database.ConstraintActiveBookingPerSlot,
		} {
			Expect(known).To(HaveKey(name), fmt.Sprintf("%q is not a constraint schema.sql creates — a check using it can never match, so the raw database error would reach the user", name))
		}
	})

	It("only matches the exact name, so a shortened one is not close enough", func() {
		reported := &pq.Error{Code: "23505", Constraint: database.ConstraintStaffUserID}

		Expect(database.IsUniqueViolation(reported, database.ConstraintStaffUserID)).To(BeTrue())
		// The bug this suite exists for: dropping the table prefix leaves a
		// name that reads right and matches nothing.
		Expect(database.IsUniqueViolation(reported, "staff_user_id_key")).To(BeFalse())
	})
})

var (
	createTable    = regexp.MustCompile(`(?i)CREATE TABLE (?:IF NOT EXISTS )?(\w+)\s*\(`)
	createUniqueIx = regexp.MustCompile(`(?i)CREATE UNIQUE INDEX (?:IF NOT EXISTS )?(\w+)`)
	columnName     = regexp.MustCompile(`^(\w+)\s`)
)

// constraintNamesIn returns every constraint name the given schema produces:
// the indexes it names itself, plus the <table>_<column>_key / _fkey names
// Postgres generates for a column declared UNIQUE or REFERENCES.
func constraintNamesIn(schema string) map[string]bool {
	names := map[string]bool{}
	for _, m := range createUniqueIx.FindAllStringSubmatch(schema, -1) {
		names[m[1]] = true
	}

	var table string
	for _, raw := range strings.Split(schema, "\n") {
		line := strings.TrimSpace(raw)
		if m := createTable.FindStringSubmatch(line); m != nil {
			table = m[1]
			continue
		}
		if table == "" || strings.HasPrefix(line, ")") {
			if strings.HasPrefix(line, ")") {
				table = ""
			}
			continue
		}
		// Only column definitions carry a generated name; table-level
		// UNIQUE (a, b) is named after its columns and is not used here.
		m := columnName.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		column := m[1]
		if strings.EqualFold(column, "unique") || strings.EqualFold(column, "check") ||
			strings.EqualFold(column, "primary") || strings.EqualFold(column, "constraint") {
			continue
		}
		upper := strings.ToUpper(line)
		if strings.Contains(upper, " UNIQUE") {
			names[fmt.Sprintf("%s_%s_key", table, column)] = true
		}
		if strings.Contains(upper, " REFERENCES ") {
			names[fmt.Sprintf("%s_%s_fkey", table, column)] = true
		}
	}
	return names
}
