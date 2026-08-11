package migration

import (
	"strings"
	"testing"
)

func TestMigrationStepsAreSequential(t *testing.T) {
	t.Parallel()
	steps := migrationSteps()
	if len(steps) != LatestVersion {
		t.Fatalf("migration count = %d, LatestVersion = %d", len(steps), LatestVersion)
	}
	for index, migration := range steps {
		wantVersion := index + 1
		if migration.version != wantVersion {
			t.Fatalf("migration at index %d has version %d, want %d", index, migration.version, wantVersion)
		}
		if migration.name == "" || migration.up == nil {
			t.Fatalf("migration version %d is incomplete", migration.version)
		}
	}
}

func TestRetouchCreditConstraintSupportsReleaseAndRefund(t *testing.T) {
	t.Parallel()
	required := []string{
		"spent_credits <= reserved_credits",
		"refunded_credits <= reserved_credits",
	}
	for _, expression := range required {
		if !strings.Contains(retouchTicketCreditConstraint, expression) {
			t.Fatalf("retouch constraint is missing %q", expression)
		}
	}
	if strings.Contains(retouchTicketCreditConstraint, "refunded_credits <= spent_credits") {
		t.Fatal("retouch constraint still rejects release before settlement")
	}
}
