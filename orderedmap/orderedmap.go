// Package orderedmap is a Go implementation of Python's OrderedDict class, a map
// that preserves the order of insertion.
package orderedmap

import "fmt"

// An element of an OrderedDict, forms a linked list ordered by insertion time
type node struct {
	Key   interface{}
	Value interface{}
	Next  *node
	Prev  *node
}

func (n *node) String() string {
	return fmt.Sprintf("Node{%v %v}", n.Key, n.Value)
}

// OrderedMap class
type OrderedMap struct {
	table map[interface{}]*node
	root  *node

	// Free node linked list
	free *node

	// slice of allocated nodes
	pool []node
}

// NewOrderedMap creates an empty OrderedMap, allocating size initial nodes
func NewOrderedMap(size int) *OrderedMap {
	root := &node{nil, nil, nil, nil} // sentinel Node
	root.Next, root.Prev = root, root
	
	pool := make([]node, size, size)
	
	//
	om := &OrderedMap{
		table: make(map[interface{}]*node),
		root:  root,
		pool:  pool,
		free:  nil,
	}
	
	// Add pool nodes to free linked list 
	for n, _ := range pool {
		pool[n].Next = om.free
		om.free = &pool[n]
	}

	return om
}

// Len returns the number of elements in the Map
func (om *OrderedMap) Len() int {
	return len(om.table)
}

// Cap returns the map capacity
func (om *OrderedMap) Cap() int {
	return len(om.pool)
}


// getNode a node from free pool
func (om *OrderedMap) getNode(key interface{}, value interface{}, 
				next *node, prev *node) (n *node, err error) {
	if om.free == nil {
		return nil, ErrFull
	}
					
	n = om.free
	om.free = om.free.Next

	n.Next  = next
	n.Prev  = prev
	n.Key   = key
	n.Value = value
	return n, nil
}

// freeNode returns a node to the free pool
func (om *OrderedMap) freeNode(n *node) {
	n.Key   = nil
	n.Value = nil
	n.Prev  = nil
	n.Next  = om.free
	om.free = n
}


// Set the key value, if the key overwrites an existing entry, the original
// insertion position is left unchanged, otherwise the key is inserted at the end.
func (om *OrderedMap) Set(key interface{}, value interface{}) (err error){
	if nd, ok := om.table[key]; !ok {
		// New entry
		root := om.root
		nd, err = om.getNode(key, value, root, root.Prev)
		if err == nil {
			root.Prev.Next = nd
			root.Prev = nd
			om.table[key] = nd
		}	
	} else {
		// Update existing entry value
		nd.Value = value
	}
	return err
}

// Get the value of an existing key, leaving the map unchanged
func (om *OrderedMap) Get(key interface{}) (value interface{}, ok bool) {
	if node, isOk := om.table[key]; !isOk {
		value, ok = nil, false
	} else {
		value, ok = node.Value, true
	}
	return
}

// GetLast return the key and value for the last element added, leaving
// the map unchanged
func (om *OrderedMap) GetLast() (key interface{}, value interface{}, ok bool) {
	if len(om.table) == 0 {
		key, value, ok = nil, nil, false
	} else {
		node := om.root.Prev
		key, value, ok = node.Key, node.Value, true
	}
	return
}

// GetFirst returns the key and value for the first element, leaving the map unchanged
func (om *OrderedMap) GetFirst() (key interface{}, value interface{}, ok bool) {
	if len(om.table) == 0 {
		key, value, ok = nil, nil, false
	} else {
		node := om.root.Next
		key, value, ok = node.Key, node.Value, true
	}
	return
}

// Delete a key:value pair from the map.
func (om *OrderedMap) Delete(key interface{}) {
	if node, ok := om.table[key]; ok {
		node.Next.Prev = node.Prev
		node.Prev.Next = node.Next

		delete(om.table, key)
		om.freeNode(node)
	}
}

// Pop and return key:value for the newest or oldest element on the OrderedMap
func (om *OrderedMap) Pop(last bool) (key interface{}, value interface{}, ok bool) {
	if last {
		key, value, ok = om.GetLast()
	} else {
		key, value, ok = om.GetFirst()
	}

	if ok {
		om.Delete(key)
	}
	return
}

// PopLast is a shortcut to Pop the last element
func (om *OrderedMap) PopLast() (key interface{}, value interface{}, ok bool) {
	return om.Pop(true)
}

// PopFirst is a shortcut to Pop the first element
func (om *OrderedMap) PopFirst() (key interface{}, value interface{}, ok bool) {
	return om.Pop(false)
}

// Move an existing key to either the end of the OrderedMap
func (om *OrderedMap) Move(key interface{}, last bool) (ok bool) {

	var moved *node

	// Remove from current position
	anode, ok := om.table[key]
	if !ok {
		return false
	}

	anode.Next.Prev = anode.Prev
	anode.Prev.Next = anode.Next
	moved = anode

	// Insert at the start or end
	root := om.root
	if last {
		moved.Next = root
		moved.Prev = root.Prev
		root.Prev.Next = moved
		root.Prev = moved
	} else {
		moved.Prev = root
		moved.Next = root.Next
		root.Next.Prev = moved
		root.Next = moved
	}

	return true
}

// MoveLast is a shortcut to Move a key to the end o the map
func (om *OrderedMap) MoveLast(key interface{}) (ok bool) {
	return om.Move(key, true)
}

// MoveFirst is a shortcut to Move a key to the beginning of the map
func (om *OrderedMap) MoveFirst(key interface{}) (ok bool) {
	return om.Move(key, false)
}

// String interface
func (om *OrderedMap) String() string {
	return fmt.Sprintf("OrderedMap(len: %v)", len(om.table))
}
