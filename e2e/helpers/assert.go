// Package helpers provides narrowly-scoped utilities for E2E testing.
package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Assert provides assertion capabilities for E2E tests.
//
// This is a thin wrapper around testify/assert to provide a consistent
// interface for E2E test assertions. All assertions log failures but
// do not stop test execution (use require package for fatal assertions).
//
// Usage:
//
//	a := NewAssert(t)
//	a.Equal(200, resp.StatusCode, "expected success status")
//	a.Contains(resp.String(), "Welcome", "page should contain greeting")
type Assert struct {
	t *testing.T
}

// NewAssert creates a new assertion helper for the given test.
func NewAssert(t *testing.T) *Assert {
	return &Assert{t: t}
}

// Equal asserts that expected and actual are equal.
//
//	a.Equal(200, resp.StatusCode)
//	a.Equal("admin", user.Role, "user should have admin role")
func (a *Assert) Equal(expected, actual interface{}, msgAndArgs ...interface{}) bool {
	return assert.Equal(a.t, expected, actual, msgAndArgs...)
}

// NotEqual asserts that expected and actual are not equal.
//
//	a.NotEqual("", user.ID, "user ID should not be empty")
func (a *Assert) NotEqual(expected, actual interface{}, msgAndArgs ...interface{}) bool {
	return assert.NotEqual(a.t, expected, actual, msgAndArgs...)
}

// Nil asserts that the specified object is nil.
//
//	a.Nil(err, "operation should succeed without error")
func (a *Assert) Nil(object interface{}, msgAndArgs ...interface{}) bool {
	return assert.Nil(a.t, object, msgAndArgs...)
}

// NotNil asserts that the specified object is not nil.
//
//	a.NotNil(user, "user should be returned")
func (a *Assert) NotNil(object interface{}, msgAndArgs ...interface{}) bool {
	return assert.NotNil(a.t, object, msgAndArgs...)
}

// True asserts that the specified value is true.
//
//	a.True(user.IsActive, "user should be active")
func (a *Assert) True(value bool, msgAndArgs ...interface{}) bool {
	return assert.True(a.t, value, msgAndArgs...)
}

// False asserts that the specified value is false.
//
//	a.False(user.IsDeleted, "user should not be deleted")
func (a *Assert) False(value bool, msgAndArgs ...interface{}) bool {
	return assert.False(a.t, value, msgAndArgs...)
}

// NoError asserts that err is nil.
//
//	resp, err := api.Call("GET", "/api/health", nil)
//	a.NoError(err, "health check should not return error")
func (a *Assert) NoError(err error, msgAndArgs ...interface{}) bool {
	return assert.NoError(a.t, err, msgAndArgs...)
}

// Error asserts that err is not nil.
//
//	_, err := api.Call("GET", "/api/protected", nil)
//	a.Error(err, "should return error for unauthenticated request")
func (a *Assert) Error(err error, msgAndArgs ...interface{}) bool {
	return assert.Error(a.t, err, msgAndArgs...)
}

// Contains asserts that the string s contains the substring.
//
//	a.Contains(resp.String(), "success", "response should contain success message")
func (a *Assert) Contains(s, contains string, msgAndArgs ...interface{}) bool {
	return assert.Contains(a.t, s, contains, msgAndArgs...)
}

// NotContains asserts that the string s does not contain the substring.
//
//	a.NotContains(resp.String(), "error", "response should not contain error")
func (a *Assert) NotContains(s, contains string, msgAndArgs ...interface{}) bool {
	return assert.NotContains(a.t, s, contains, msgAndArgs...)
}

// Len asserts that the specified object has the expected length.
//
//	a.Len(users, 3, "should return 3 users")
func (a *Assert) Len(object interface{}, length int, msgAndArgs ...interface{}) bool {
	return assert.Len(a.t, object, length, msgAndArgs...)
}

// Empty asserts that the specified object is empty.
//
//	a.Empty(errors, "should have no validation errors")
func (a *Assert) Empty(object interface{}, msgAndArgs ...interface{}) bool {
	return assert.Empty(a.t, object, msgAndArgs...)
}

// NotEmpty asserts that the specified object is not empty.
//
//	a.NotEmpty(users, "should return at least one user")
func (a *Assert) NotEmpty(object interface{}, msgAndArgs ...interface{}) bool {
	return assert.NotEmpty(a.t, object, msgAndArgs...)
}
