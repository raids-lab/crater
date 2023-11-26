/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// heap.Interface implementation inspired by
// https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/internal/heap/heap.go
package queue

import (
	"sort"
)

// lessFunc is a function that receives two items and returns true if the first
// item should be placed before the second one when the list is sorted.
type lessFunc func(a, b interface{}) bool

// KeyFunc is a function type to get the key from an object.
type keyFunc func(obj interface{}) string

type queueItem struct {
	obj   interface{}
	index int
}

type itemKeyValue struct {
	key string
	obj interface{}
}

// data is an internal struct that implements the standard heap interface
// and keeps the data stored in the heap.
type data struct {
	// items is a map from key of the objects to the objects and their index
	items map[string]*queueItem
	// keys keeps the keys of the objects ordered according to the heap invariant.
	keys     []string
	keyFunc  keyFunc
	lessFunc lessFunc
}

// Less compares two objects and returns true if the first one should go
// in front of the second one in the heap.
func (h *data) Less(i, j int) bool {
	if i > h.Len() || j > h.Len() {
		return false
	}
	a, ok := h.items[h.keys[i]]
	if !ok {
		return false
	}
	b, ok := h.items[h.keys[j]]
	if !ok {
		return false
	}
	return h.lessFunc(a.obj, b.obj)
}

// Swap implements swapping of two elements in the heap. This is a part of standard
// heap interface and should never be called directly.
func (h *data) Swap(i, j int) {
	h.keys[i], h.keys[j] = h.keys[j], h.keys[i]
	h.items[h.keys[i]].index = i
	h.items[h.keys[j]].index = j
}

// Len returns the number of items in the Heap.
func (h *data) Len() int {
	return len(h.keys)
}

// Push is supposed to be called by heap.Push only.
func (h *data) push(kv interface{}) {
	keyValue := kv.(*itemKeyValue)
	h.items[keyValue.key] = &queueItem{
		obj:   keyValue.obj,
		index: len(h.keys),
	}
	h.keys = append(h.keys, keyValue.key)
}

// Pop is supposed to be called by heap.Pop only.
func (h *data) pop() interface{} {
	key := h.keys[len(h.keys)-1]
	h.keys = h.keys[:len(h.keys)-1]
	item, ok := h.items[key]
	if !ok {
		// This is an error
		return nil
	}
	delete(h.items, key)
	return item.obj
}

// Pop is supposed to be called by heap.Pop only.
func (h *data) top() interface{} {
	key := h.keys[len(h.keys)-1]
	h.keys = h.keys[:len(h.keys)-1]
	item, ok := h.items[key]
	if !ok {
		// This is an error
		return nil
	}
	return item.obj
}

func (h *data) delete(key string) interface{} {
	item, ok := h.items[key]
	if !ok {
		return nil
	}
	for i, k := range h.keys {
		if k == key {
			h.keys = append(h.keys[:i], h.keys[i+1:]...)
			break
		}
	}

	delete(h.items, key)
	return item.obj

}

// Queue is a producer/consumer queue that implements a queue data structure.
// It can be used to implement priority queues and similar data structures.
type Queue struct {
	data
}

// PushOrUpdate inserts an item to the queue.
// The item will be updated if it already exists.
func (q *Queue) PushOrUpdate(obj interface{}) {
	key := q.data.keyFunc(obj)
	if _, exists := q.data.items[key]; exists {
		q.data.items[key].obj = obj
	} else {
		q.push(&itemKeyValue{key, obj})
	}
}

// PushIfNotPresent inserts an item to the queue. If an item with
// the key is present in the map, no changes is made to the item.
func (q *Queue) PushIfNotPresent(obj interface{}) (added bool) {
	key := q.keyFunc(obj)
	if _, exists := q.items[key]; exists {
		return false
	}

	q.push(&itemKeyValue{key, obj})
	return true
}

// DeleteByKey removes an item by key
func (q *Queue) DeleteByKey(key string) interface{} {
	return q.delete(key)
}

// Delete removes an item by key
func (q *Queue) Delete(obj interface{}) interface{} {
	key := q.keyFunc(obj)
	return q.delete(key)
}

// Pop returns the head of the heap and removes it.
func (q *Queue) Pop() interface{} {
	return q.pop()
}

// 
func (q *Queue) Top() interface{} {
	return q.top()
}

// Get returns the requested item, exists, error.
func (q *Queue) Get(obj interface{}) (item interface{}) {
	key := q.keyFunc(obj)
	return q.GetByKey(key)
}

// GetByKey returns the requested item, or sets exists=false.
func (q *Queue) GetByKey(key string) interface{} {
	item, exists := q.items[key]
	if !exists {
		return nil
	}
	return item.obj
}

// List returns a list of all the items.
func (q *Queue) List() []interface{} {
	list := make([]interface{}, 0, q.Len())
	for _, key := range q.keys {
		list = append(list, q.items[key].obj)
	}
	return list
}

// Reorder reorders the queue.
func (q *Queue) Reorder() {
	sort.Sort(&q.data)
}

// ReorderWithFunc reorders the queue with lessFn.
func (q *Queue) ReorderWithFunc(lessFn lessFunc) {
	q.lessFunc = lessFn
	sort.Sort(&q.data)
}

// New returns a Queue which can be used to queue up items to process.
func New(keyFn keyFunc, lessFn lessFunc) Queue {
	return Queue{
		data: data{
			items:    map[string]*queueItem{},
			keyFunc:  keyFn,
			lessFunc: lessFn,
		},
	}
}
