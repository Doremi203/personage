package pushpostgres

import (
	"testing"

	"github.com/Doremi203/personage/backend/libs/go/postgres"
)

var tester postgres.Tester

func TestMain(m *testing.M) {
	postgres.SetupTests(m, &tester, "notificator")
}
