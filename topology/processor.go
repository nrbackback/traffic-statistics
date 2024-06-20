package topology

type Processor interface {
	Process(map[string]interface{}) map[string]interface{}
}

type ProcessorNode struct {
	Processor Processor
	Next      *ProcessorNode
}

func (node *ProcessorNode) Process(event map[string]interface{}) map[string]interface{} {
	event = node.Processor.Process(event)
	if event == nil || node.Next == nil {
		return event
	}
	return node.Next.Process(event)
}

// AppendProcessorsToLink 将 processors 放到 head 后面
func AppendProcessorsToLink(head *ProcessorNode, processors ...Processor) *ProcessorNode {
	preHead := &ProcessorNode{nil, head}
	n := preHead
	for n.Next != nil {
		n = n.Next
	}
	for _, processor := range processors {
		n.Next = &ProcessorNode{processor, nil}
		n = n.Next
	}

	return preHead.Next
}
