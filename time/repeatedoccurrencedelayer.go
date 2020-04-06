package time

import (
	"time"
)

// RepeatedOccurrenceDelayer is a RepeatedOccurrence where occurrences are given
// specific delays. By convention, an occurrence should not happen until after
// a delay has been explicitly set and subsequently completed.
type RepeatedOccurrenceDelayer interface {
	RepeatedOccurrence
	DelayForNext(time.Duration) error
}
