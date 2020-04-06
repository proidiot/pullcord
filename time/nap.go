package time

type Nap interface {
	WaitToBeAwoken() error
}
