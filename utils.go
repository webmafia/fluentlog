package fluentlog

func tryWrite[T any](ch chan<- T, v T, n int) bool {
	for range n {
		select {
		case ch <- v:
			return true
		default:
		}
	}

	return false
}
