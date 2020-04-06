package time

// RepeatedOccurrence provides an abstraction by which one can wait for another
// of a presumably repeating occurrence. If another occurrence has happened,
// then WaitForNext() should return nil. This interface is not intended to imply
// that occurrences will or will not continue without end, or even happen at
// all. In addition, this interface is not intended to imply that WaitForNext()
// would necesarilly ever complete, just that if it does complete with a return
// value of nil then the occurrence being waited for has indeed happened. It
// should be safe for a caller to call WaitForNext() successively, and it should
// be the responsibility of a called implementation to return some error if it
// is contextually invalid to make further calls.
type RepeatedOccurrence interface {
	WaitForNext() error
}
