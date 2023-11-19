package exector

type Finisher interface {
	Finish(func())
}
