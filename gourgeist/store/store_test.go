package store

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type StoreTestSuite struct {
	suite.Suite
	Store *Store
}

func TestStoreTestSuiteIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(StoreTestSuite))
}
