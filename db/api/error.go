package api

import "encoding/hex"

type (
	ErrNoTable string
	ErrNoKey   string

	tableNil int
	keyNil   int
)

var (
	// TableNilError error
	TableNilError tableNil = 2
	// KeyNilError error
	KeyNilError keyNil = 3

	tableIsNil = "table is <nil>"
	keyIsNil   = "key is <nil>"
)

//
// IsNoTableError checks if error is of type of noTableError
//
func IsNoTableError(err error) bool {
	_, ok := err.(ErrNoTable)
	return ok
}

func (e ErrNoTable) Error() string {
	return "table '" + string(e) + "' does not exist"
}

//
// IsNoKeyError checks if error is of type of noKeyError
//
func IsNoKeyError(err error) bool {
	_, ok := err.(ErrNoKey)
	return ok
}

func (e ErrNoKey) Error() string {
	return "key '" + hex.EncodeToString([]byte(e)) + "' does not exists"
}

//
// IsTableNil check if error is of type of tableNil
//
func IsTableNil(err error) bool {
	_, ok := err.(tableNil)
	return ok
}

func (e tableNil) Error() string {
	return tableIsNil
}

//
// IsKeyNil checks if error is of type of keyNil
//
func IsKeyNil(err error) bool {
	_, ok := err.(keyNil)
	return ok
}

func (e keyNil) Error() string {
	return keyIsNil
}
