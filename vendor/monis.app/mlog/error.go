package mlog

var _ error = constableError("")

type constableError string

func (e constableError) Error() string {
	return string(e)
}
