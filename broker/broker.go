package broker

type Broker[T any] struct {
	stopCh    chan struct{}
	publishCh chan T
	subCh     chan chan<- T
	unsubCh   chan chan<- T
}

func New[T any]() *Broker[T] {
	return &Broker[T]{
		stopCh:    make(chan struct{}),
		publishCh: make(chan T, 1),
		subCh:     make(chan chan<- T, 1),
		unsubCh:   make(chan chan<- T, 1),
	}
}

func (b *Broker[T]) Start() {
	subscribers := make(map[chan<- T]struct{})
	for {
		select {
		case <-b.stopCh:
			return
		case msgCh := <-b.subCh:
			subscribers[msgCh] = struct{}{}
		case msgCh := <-b.unsubCh:
			delete(subscribers, msgCh)
			close(msgCh)
		case event := <-b.publishCh:
			for msgCh := range subscribers {
				select {
				case msgCh <- event:
				default:
				}
			}
		}
	}
}

func (b *Broker[T]) Stop() {
	close(b.stopCh)
}

func (b *Broker[T]) Subscribe(ch chan<- T) {
	b.subCh <- ch
}

func (b *Broker[T]) Unsubscribe(ch chan<- T) {
	b.unsubCh <- ch
}

func (b *Broker[T]) Publish(msg T) {
	b.publishCh <- msg
}
