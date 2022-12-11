package nowait

type immediateRequest struct {
	disposerFn func()
}

func (r immediateRequest) Finish(execute func()) {
	defer r.disposerFn()
	execute()
}
