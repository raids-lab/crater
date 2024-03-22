package aitaskctl

// Attributes parsed from yaml
type TaskAttr struct {
}

// ******************* LinkList ***********************************

type LinkList struct {
	Prev *LinkList
	Next *LinkList
	Task *InternalTask
}

// Insert src before dst
func (dst *LinkList) InsertBefore(src *LinkList) {
	prev := dst.Prev
	if prev != nil {
		prev.Next = src
	}
	dst.Prev = src

	src.Prev = prev
	src.Next = dst
}

// Cut src out of LinkList
func (src *LinkList) CutOut() {
	prev := src.Prev
	if prev != nil {
		prev.Next = src.Next
	}
	next := src.Next
	if next != nil {
		next.Prev = src.Prev
	}
}

// ******************* ExecuteEngine ***********************************

type InternalTask struct {
	Attr         *TaskAttr
	LinkListItem *LinkList
	ID           int32
}

func (t *InternalTask) Run() {}

type ExecuteEngine struct {
	LinkListHead *LinkList
	LinkListTail *LinkList
	UniqueID     int32
	// map UniqueID to Internal Task
	ID2Task map[int32]*InternalTask
}

// Init an execute engine.
// Make a special LinkListItem as LinkListHead and LinkListTail.
func (e *ExecuteEngine) Init() {
	e.LinkListHead = &LinkList{}
	e.LinkListTail = e.LinkListHead
	e.UniqueID = 0
	e.ID2Task = make(map[int32]*InternalTask)
}

// Submit a task to execute engine and get an unique ID back
func (e *ExecuteEngine) SubmitTask(attr *TaskAttr) (uniqueID int32) {
	uniqueID = e.UniqueID
	e.UniqueID += 1

	// construct new task
	task := &InternalTask{
		Attr: attr,
		ID:   uniqueID,
	}
	linkList := &LinkList{
		Prev: nil,
		Next: nil,
		Task: task,
	}
	task.LinkListItem = linkList

	// insert just before LinkListTail
	e.LinkListTail.InsertBefore(linkList)

	// current queue is empty
	if e.LinkListHead == e.LinkListTail {
		e.LinkListHead = linkList
	}

	// record the mapping
	e.ID2Task[uniqueID] = task
	return
}

// Adjust srcID task before dstID task
func (e *ExecuteEngine) AdjustTask(srcID int32, dstID int32) {
	// find internal task
	srcTask := e.ID2Task[srcID]
	dstTask := e.ID2Task[dstID]

	srcTaskLink := srcTask.LinkListItem
	dstTaskLink := dstTask.LinkListItem

	// cut srcTask out of link list
	srcTaskLink.CutOut()

	// add srcTask before dstTaskLink
	dstTaskLink.InsertBefore(srcTaskLink)
}

func (e *ExecuteEngine) Run() {
	for e.LinkListHead != e.LinkListTail {
		linkList := e.LinkListHead
		go linkList.Task.Run()
		e.LinkListHead = linkList.Next
	}
}
