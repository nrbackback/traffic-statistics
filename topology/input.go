package topology

type InputWorker interface {
	ReadOneEvent() map[string]interface{}
	Shutdown()
}
