package main

import (
	"errors"

	validation "github.com/pocketbase/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/tools/router"
)

// userError is an API error the UI can translate: it is sent as
//
//	{"status": 409, "message": "Someone already reserved this.", "data": {"error": {"code": "already_reserved", "params": {…}}}}
//
// The English message stays as the fallback for clients that don't know the code.
type userError struct {
	status int
	code   string
	msg    string
	params map[string]any
}

func (u *userError) Error() string { return u.msg }

// uerr builds a userError; kv are optional param pairs ("host", "example.com", …).
func uerr(status int, code, msg string, kv ...any) *userError {
	u := &userError{status: status, code: code, msg: msg}
	if len(kv) > 1 {
		u.params = map[string]any{}
		for i := 0; i+1 < len(kv); i += 2 {
			if k, ok := kv[i].(string); ok {
				u.params[k] = kv[i+1]
			}
		}
	}
	return u
}

func (u *userError) api() *router.ApiError {
	ve := validation.NewError(u.code, u.msg)
	if u.params != nil {
		ve = ve.SetParams(u.params)
	}
	return router.NewApiError(u.status, u.msg, map[string]validation.Error{"error": ve})
}

// apiError converts a userError (possibly wrapped) into its API form; other errors pass through.
func apiError(err error) error {
	var u *userError
	if errors.As(err, &u) {
		return u.api()
	}
	return err
}
