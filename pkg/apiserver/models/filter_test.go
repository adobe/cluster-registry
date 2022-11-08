package models

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewFilterConditionFromQuery(t *testing.T) {
	test := assert.New(t)

	successTestCases := []struct {
		query                                         string
		expectedField, expectedOperand, expectedValue string
	}{
		{
			"foo.bar:<10",
			"foo.bar", "<", "10",
		},
		{
			"foo.bar:=",
			"foo.bar", "=", "",
		},
	}

	failureTestCases := []struct {
		query         string
		expectedError error
	}{
		{
			"foo.bar :<10",
			fmt.Errorf("invalid query"),
		},
		{
			"foo.bar: < 10",
			fmt.Errorf("invalid query"),
		},
		{
			"foo.bar:*10",
			fmt.Errorf("invalid query"),
		},
	}

	for _, tc := range successTestCases {
		condition, err := NewFilterConditionFromQuery(tc.query)
		test.NoError(err)
		if test.NotNil(condition) {
			test.Equal(tc.expectedField, condition.Field)
			test.Equal(tc.expectedOperand, condition.Operand)
			test.Equal(tc.expectedValue, condition.Value)
		}
	}

	for _, tc := range failureTestCases {
		condition, err := NewFilterConditionFromQuery(tc.query)
		test.Error(tc.expectedError, err)
		test.Nil(condition)
	}
}
