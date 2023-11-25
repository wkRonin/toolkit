/*
 *    Copyright 2023 wkRonin
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package queue

import (
	"errors"

	"github.com/wkRonin/toolkit/containerx"
	"github.com/wkRonin/toolkit/containerx/slice"
)

var (
	ErrOutOfCapacity = errors.New("priorityQueue exceeding capacity limit ")
	ErrEmptyQueue    = errors.New("priorityQueue has no elements")
)

type PriorityQueue[T any] struct {
	compare  containerx.Comparator[T]
	capacity int
	data     []T
	length   int
}

// NewPriorityQueue 创建优先队列 capacity <= 0 时，为无界队列，否则为有界队列
func NewPriorityQueue[T any](capacity int, compare containerx.Comparator[T]) *PriorityQueue[T] {
	sliceCap := capacity
	if capacity < 1 {
		capacity = 0
		sliceCap = 64
	}
	return &PriorityQueue[T]{
		compare:  compare,
		capacity: capacity,
		data:     make([]T, 0, sliceCap),
		length:   0,
	}
}

func (p *PriorityQueue[T]) isEmpty() bool {
	return p.length == 0
}

func (p *PriorityQueue[T]) isFull() bool {
	return p.capacity > 0 && p.length == p.capacity
}

func (p *PriorityQueue[T]) Len() int {
	return p.length
}

// Cap 无界队列返回0，有界队列返回创建队列时设置的值
func (p *PriorityQueue[T]) Cap() int {
	return p.capacity
}

func (p *PriorityQueue[T]) IsBoundless() bool {
	return p.capacity <= 0
}

func (p *PriorityQueue[T]) shrinkIfNecessary() {
	if p.IsBoundless() {
		p.data = slice.Shrink[T](p.data)
	}
}

func (p *PriorityQueue[T]) Dequeue() (T, error) {
	if p.isEmpty() {
		var t T
		return t, ErrEmptyQueue
	} else if p.length == 1 {
		result := p.data[0]
		p.length--
		p.data = p.data[1:]
		return result, nil
	} else {
		result := p.data[0]
		p.data[0], p.data[p.length-1] = p.data[p.length-1], p.data[0]
		p.data = p.data[:p.length-1]
		p.shrinkIfNecessary()
		p.length--
		p.data = p.heapifyUp(p.data, 0, p.length-1)
		return result, nil
	}
}
func (p *PriorityQueue[T]) Peek() (T, error) {
	if p.isEmpty() {
		var t T
		return t, ErrEmptyQueue
	}
	return p.data[0], nil
}

func (p *PriorityQueue[T]) Enqueue(t T) error {
	if p.isFull() {
		return ErrOutOfCapacity
	}
	p.data = append(p.data, t)
	p.length++
	p.data = p.heapifyDown(p.data, p.length-1, 0)
	return nil
}

// 堆自上向下堆化
func (p *PriorityQueue[T]) heapifyUp(data []T, start, end int) []T {
	left := 2*start + 1
	for left <= end {
		tmp := left
		right := left + 1
		if right <= end && p.compare(data[right], data[tmp]) {
			tmp = right
		}
		if p.compare(data[tmp], data[start]) {
			data[tmp], data[start] = data[start], data[tmp]
			start = tmp
			left = start*2 + 1
		} else {
			break
		}
	}
	return data
}

// 堆自下向上堆化
func (p *PriorityQueue[T]) heapifyDown(data []T, start, end int) []T {
	father := (start - 1) / 2
	for father >= end && father != start {
		if p.compare(data[start], data[father]) {
			data[father], data[start] = data[start], data[father]
			start = father
			father = (start - 1) / 2
		} else {
			break
		}
	}
	return data
}
