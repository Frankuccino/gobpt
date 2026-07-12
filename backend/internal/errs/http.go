package errs

import (
	"strings"
)

type FieldError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

type ActionType string

const (
	ActionTypeRedirect ActionType = "redirect"
)

type Action struct {
	Type    ActionType `json:"type"`
	Message string     `json:"message"`
	Value   string     `json:"value"`
}

type HTTPError struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Status   int    `json:"status"`
	Override bool   `json:"override"`
	// field level errors
	Errors []FieldError `json:"errors"`
	// action to be taken
	Action *Action `json:"action"`
}

func (e *HTTPError) Error() string {
	return e.Message
}

func (e *HTTPError) Is(target error) bool {
	_, ok := target.(*HTTPError)
	return ok
}

func (e *HTTPError) WithMessage(message string) *HTTPError {
	return &HTTPError{
		Code:     e.Code,
		Message:  message,
		Status:   e.Status,
		Override: e.Override,
		Errors:   e.Errors,
		Action:   e.Action,
	}
}

func MakeUpperCaseWithUnderscores(str string) string {
	return strings.ToUpper(strings.ReplaceAll(str, " ", "_"))
}

// The whole point of this file is to create a custom error type instead of whatever that's
// provided to us by the libraries like PGX, or Echo.
// We want to create our own custom error type solely for the purpose of sending to our client
// Later on this could be our OpenAPI testing interface once after we've integrated it.
// This is so that the client get a meaningful and clear message and actionable steps that
// they can use to fix the error, to determine the source of the error and focus there.

// FieldError struct - is let's say we're taking a form. We'll validate that particular payload, and
// if the format is incorrect, or it failed the validation, we want to send a message back
// to the client, signifying the Field and the Error that caused the error.
// This is used as an slice/array in its final form, see the HTTPError struct

// ActionType struct - This is useful when user session is expired.
// This is used in the Action struct which has a Type, Message, and Value.
// if the user session is expired we want to send a 401 unauthorized error, in some cases
// we also want the user to be redirected in another route. That's why we use this in HTTPError struct

// HTTPError struct - this is the composition of the FieldError and the Action struct.
// We're using a pointer to the Action strut which means that for most of the errors, we're not
// going to use the Action field. We'll mostly use the first four fields.
// Same goes with the Errors since it's a slice, the zero value will also be nil if there are no errors.
// Fields:

// Code - This is some uppercased snakecase custom error code for frontend can parse. Like "TODO_NOT_FOUND"
// This allows the frontend to send a more user friendly message, a more personalized message

// Message - The message we send from the backend. Sometimes it's short, or a userfirendly message
// This is a readable format instead of the uppercased snakecase format.

// Status - Simply means the status codes like 200, 201, or 400, etc.

// Override - This field is not used all the time, but we've added this in some cases like
// we have an email based OTP-based authentication process in our frontend, and for that particular workflow,
// we are limiting how many times a user can type a wrong OTP. e.g., Max limit set to 4
// That means after 4 failed attempts on the OTP, the account will be blocked for the next 24 hrs.
// The first time the user types a wrong OTP, we want to send an error.
// An error like 403 forbidden, and with that error, we want to send a message that:
// "you have typed a wrong OTP, you have three more attempts".
// It's a message coming directly from the backend. The frontend does not know it.
// So in these kind of cases, we want to override the message,
// the user friendly message that the frontend is trying to show.
// the frontend don't need to parse the code and show another message,
// we can directly show whatever message that's coming from the backend.

// With the methods of HTTPError we have: Error, Is, and WithMessage
// The Error, we are implicitly implmententing Go's built-in error interface
// The Is, this is for comparison purposes
// The WithMessage, and MakeUpperCaseWithUnderscores these are utility functions
