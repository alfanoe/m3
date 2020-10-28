// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package roaring

import (
	"errors"
	"fmt"
	"math/bits"
	"sync"

	"github.com/m3db/m3/src/m3ninx/postings"
)

var (
	// ErrNotReadOnlyBitmaps returned from operations that expect read only bitmaps.
	ErrNotReadOnlyBitmaps = errors.New("not read only bitmaps")
)

// UnionReadOnly expects postings lists to all be read only.
func UnionReadOnly(unions []postings.List) (postings.List, error) {
	union := make([]multiBitmapIterable, 0, len(unions))
	for _, elem := range unions {
		b, ok := elem.(*ReadOnlyBitmap)
		if ok {
			union = append(union, multiBitmapIterable{bitmap: b})
			continue
		}

		mb, ok := elem.(*multiBitmap)
		if ok {
			union = append(union, multiBitmapIterable{multiBitmap: mb})
			continue
		}

		return nil, ErrNotReadOnlyBitmaps
	}

	return newMultiBitmap(multiBitmapOptions{
		op:    multiBitmapOpUnion,
		union: union,
	})
}

// IntersectAndNegateReadOnly expects postings lists to all be read only.
func IntersectAndNegateReadOnly(
	intersects []postings.List,
	negates []postings.List,
) (postings.List, error) {
	intersect := make([]multiBitmapIterable, 0, len(intersects))
	for _, elem := range intersects {
		b, ok := elem.(*ReadOnlyBitmap)
		if ok {
			intersect = append(intersect, multiBitmapIterable{bitmap: b})
			continue
		}

		mb, ok := elem.(*multiBitmap)
		if ok {
			intersect = append(intersect, multiBitmapIterable{multiBitmap: mb})
			continue
		}

		return nil, ErrNotReadOnlyBitmaps
	}

	negate := make([]multiBitmapIterable, 0, len(negates))
	for _, elem := range negates {
		b, ok := elem.(*ReadOnlyBitmap)
		if ok {
			negate = append(negate, multiBitmapIterable{bitmap: b})
			continue
		}

		mb, ok := elem.(*multiBitmap)
		if ok {
			negate = append(negate, multiBitmapIterable{multiBitmap: mb})
			continue
		}

		return nil, ErrNotReadOnlyBitmaps
	}

	return newMultiBitmap(multiBitmapOptions{
		op:              multiBitmapOpIntersect,
		intersect:       intersect,
		intersectNegate: negate,
	})
}

var _ postings.List = (*multiBitmap)(nil)

type multiBitmapOp uint8

const (
	multiBitmapOpUnknown multiBitmapOp = iota

	// Place valid values between unknown and terminator
	multiBitmapOpUnion
	multiBitmapOpIntersect

	multiBitmapOpInvalidLast
)

// validateMultiBitmapOp can do fast validation because it's a range check.
func (op multiBitmapOp) validate() error {
	// Fast validation
	if op > multiBitmapOpUnknown && op < multiBitmapOpInvalidLast {
		return nil
	}
	return fmt.Errorf("invalid multi-iter op: %d", op)
}

// multiBitmap is a tree like iterator.
type multiBitmap struct {
	multiBitmapOptions
}

// multiBitmapIterable either contains a bitmap or another multi-iter.
type multiBitmapIterable struct {
	multiBitmap *multiBitmap
	bitmap      *ReadOnlyBitmap
}

func (i multiBitmapIterable) Contains(id postings.ID) bool {
	if i.multiBitmap != nil {
		return i.multiBitmap.Contains(id)
	}
	return i.bitmap.Contains(id)
}

type multiBitmapOptions struct {
	op multiBitmapOp

	// union is valid when multiBitmapOpUnion, no other options valid.
	union []multiBitmapIterable

	// intersect is valid when multiBitmapOpIntersect used.
	intersect []multiBitmapIterable
	// intersectNegate is valid when multiBitmapOpIntersect used.
	intersectNegate []multiBitmapIterable
}

func (o multiBitmapOptions) validate() error {
	if err := o.op.validate(); err != nil {
		return err
	}
	return nil
}

func newMultiBitmap(opts multiBitmapOptions) (*multiBitmap, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}
	return &multiBitmap{multiBitmapOptions: opts}, nil
}

func (i *multiBitmap) Contains(id postings.ID) bool {
	// Note: (Performance) Contains isn't used in the query path
	// so not important how fast this implementation is.
	switch i.op { // combineOp validated at creation, ignore invalid.
	case multiBitmapOpUnion:
		for _, b := range i.union {
			if b.Contains(id) {
				return true
			}
		}
		return false
	case multiBitmapOpIntersect:
		for _, b := range i.intersect {
			if !b.Contains(id) {
				return false
			}
		}
		for _, b := range i.intersectNegate {
			if b.Contains(id) {
				return false
			}
		}
		// Only valid if all intersecting actually matched,
		// if zero intersecting then postings does not contain ID.
		return len(i.intersect) > 0
	}
	return false
}

func (i *multiBitmap) IsEmpty() bool {
	iter := i.Iterator()
	hasAny := iter.Next()
	_ = iter.Err()
	_ = iter.Close()
	return hasAny
}

func (i *multiBitmap) CountFast() (int, bool) {
	// We only know length after iterating.
	return 0, false
}

func (i *multiBitmap) CountSlow() int {
	return postings.CountSlow(i)
}

func (i *multiBitmap) Iterator() postings.Iterator {
	return newMultiBitmapIterator(i.multiBitmapOptions)
}

func (i *multiBitmap) containerIterator() containerIterator {
	return newMultiBitmapContainersIterator(i.multiBitmapOptions)
}

func (i *multiBitmap) Equal(other postings.List) bool {
	return postings.Equal(i, other)
}

var _ postings.Iterator = (*multiBitmapIterator)(nil)

type multiBitmapIterator struct {
	multiBitmapOptions

	err                error
	initial            []containerIteratorAndOp
	iters              []containerIteratorAndOp
	filtered           []containerIteratorAndOp
	multiContainerIter multiBitmapContainerIterator
	bitmap             *bitmapContainer
	bitmapIter         bitmapContainerIterator
	tempBitmap         *bitmapContainer
}

type containerIteratorAndOp struct {
	it containerIterator
	op multiContainerOp
}

type multiContainerOp uint8

const (
	multiContainerOpUnion multiContainerOp = iota
	multiContainerOpIntersect
	multiContainerOpNegate
)

type containerIterator interface {
	NextContainer() bool
	ContainerKey() uint64
	ContainerUnion(ctx containerOpContext, target *bitmapContainer)
	ContainerIntersect(ctx containerOpContext, target *bitmapContainer)
	ContainerNegate(ctx containerOpContext, target *bitmapContainer)
	Err() error
	Close()
}

type containerOpContext struct {
	// siblings is how many other containers at this container there is
	// being operated on.
	siblings int
	// tempBitmap is useful for temporary scratch operations and allows
	// for all sub-operations to share it rather than one per underlying
	// container iterator.
	tempBitmap *bitmapContainer
}

func newMultiBitmapIterator(
	opts multiBitmapOptions,
) *multiBitmapIterator {
	var (
		n     = len(opts.union) + len(opts.intersect) + len(opts.intersectNegate)
		iters = make([]containerIteratorAndOp, 0, n)
	)
	iters = appendContainerItersWithOp(iters, opts.union, multiContainerOpUnion)
	iters = appendContainerItersWithOp(iters, opts.intersect, multiContainerOpIntersect)
	iters = appendContainerItersWithOp(iters, opts.intersectNegate, multiContainerOpNegate)
	i := &multiBitmapIterator{
		multiBitmapOptions: opts,
		initial:            iters,
		iters:              iters,
		bitmap:             getBitmapContainer(),
		tempBitmap:         getBitmapContainer(),
	}
	i.bitmapIter.Reset(0, i.bitmap)
	return i
}

func appendContainerItersWithOp(
	slice []containerIteratorAndOp,
	iterables []multiBitmapIterable,
	op multiContainerOp,
) []containerIteratorAndOp {
	for _, elem := range iterables {
		var it containerIterator
		switch {
		case elem.multiBitmap != nil:
			it = elem.multiBitmap.containerIterator()

		case elem.bitmap != nil:
			it = elem.bitmap.containerIterator()
		}

		if !it.NextContainer() {
			continue
		}

		slice = append(slice, containerIteratorAndOp{
			it: it,
			op: op,
		})
	}
	return slice
}

func (i *multiBitmapIterator) Next() bool {
	if i.err != nil {
		return false
	}

	for !i.bitmapIter.Next() {
		// Reset to next containers.
		var (
			ok  bool
			err error
		)
		i.iters, ok, err = i.multiContainerIter.resetAndReturnValid(i.iters)
		if err != nil {
			i.err = err
			return false
		}
		if !ok {
			// Entirely exhausted valid iterators.
			return false
		}

		// Combine all current containers into single bitmap.
		switch i.op { // Op is already validated at creation time.
		case multiBitmapOpUnion:
			// Start bitmap as unset.
			i.bitmap.Reset(false)

			// All are unions.
			unions := i.filter(i.multiContainerIter.containerIters, multiContainerOpUnion)
			ctx := containerOpContext{
				siblings:   len(unions) - 1,
				tempBitmap: i.tempBitmap,
			}
			for _, iter := range unions {
				iter.it.ContainerUnion(ctx, i.bitmap)
			}
		case multiBitmapOpIntersect:
			totalIntersect := len(i.filter(i.initial, multiContainerOpIntersect))
			currIntersect := len(i.filter(i.multiContainerIter.containerIters, multiContainerOpIntersect))

			// NB(r): Only intersect if all iterators have a container, otherwise
			// there is zero overlap and so intersecting always results in
			// no results for this container.
			if totalIntersect != currIntersect {
				continue
			}

			if currIntersect == 0 {
				// No intersections so only possible negations of nothing.
				continue
			}

			// Start bitmap as set, guaranteed to have one intersect call.
			i.bitmap.Reset(true)

			currNegate := len(i.filter(i.multiContainerIter.containerIters, multiContainerOpNegate))
			ctx := containerOpContext{
				siblings:   currIntersect + currNegate - 1,
				tempBitmap: i.tempBitmap,
			}
			// Perform intersects.
			intersects := i.filter(i.multiContainerIter.containerIters, multiContainerOpIntersect)
			for _, iter := range intersects {
				iter.it.ContainerIntersect(ctx, i.bitmap)
			}
			// Now perform negations.
			negates := i.filter(i.multiContainerIter.containerIters, multiContainerOpNegate)
			for _, iter := range negates {
				iter.it.ContainerNegate(ctx, i.bitmap)
			}
		}

		// Reset the bitmap iterator to read from new bitmap with container key.
		i.bitmapIter.Reset(i.multiContainerIter.containerKey, i.bitmap)
	}

	// Otherwise multi container iterator has next value.
	return true
}

func (i *multiBitmapIterator) filter(
	iters []containerIteratorAndOp,
	op multiContainerOp,
) []containerIteratorAndOp {
	// Reuse filter slice.
	if i.filtered == nil {
		// Alloc at longest possible slice, which is total iters
		// created for the multi bitmap iterator.
		i.filtered = make([]containerIteratorAndOp, 0, len(i.iters))
	}
	i.filtered = i.filtered[:0]
	for _, iter := range iters {
		if iter.op == op {
			i.filtered = append(i.filtered, iter)
		}
	}
	return i.filtered
}

func (i *multiBitmapIterator) Current() postings.ID {
	return postings.ID(i.bitmapIter.Current())
}

func (i *multiBitmapIterator) Err() error {
	return i.err
}

func (i *multiBitmapIterator) Close() error {
	// Close any iters that are left if we abort early.
	for _, iter := range i.iters {
		iter.it.Close()
	}

	// Return bitmaps to pool.
	putBitmapContainer(i.bitmap)
	i.bitmap = nil
	putBitmapContainer(i.tempBitmap)
	i.tempBitmap = nil
	// No longer reference the bitmap from iterator.
	i.bitmapIter.Reset(0, nil)
	return nil
}

type multiBitmapContainerIterator struct {
	containerIters []containerIteratorAndOp
	containerKey   uint64

	hasPrevContainerKey bool
}

func (i *multiBitmapContainerIterator) resetAndReturnValid(
	input []containerIteratorAndOp,
) ([]containerIteratorAndOp, bool, error) {
	// Reset current state.
	i.containerIters = i.containerIters[:0]

	var (
		// Track valid and reuse input slice.
		valid            = input[:0]
		nextContainerKey uint64
	)
	for _, iter := range input {
		iterContainerKey := iter.it.ContainerKey()
		if i.hasPrevContainerKey && iterContainerKey == i.containerKey {
			// Consequent iteration, bump to next container as needs to progress.
			if !iter.it.NextContainer() {
				// Don't include, exhausted.
				err := iter.it.Err()
				iter.it.Close() // Always close
				if err != nil {
					return nil, false, err
				}
				continue
			}

			// Get next container key.
			iterContainerKey = iter.it.ContainerKey()
		}

		// First iteration, lowest wins, everything always valid.
		valid = append(valid, iter)

		if len(i.containerIters) == 0 || iterContainerKey < nextContainerKey {
			// First or new lowest.
			i.containerIters = append(i.containerIters[:0], iter)
			nextContainerKey = iterContainerKey
		} else if iterContainerKey == nextContainerKey {
			// Enqueue if same.
			i.containerIters = append(i.containerIters, iter)
		}
	}

	i.containerKey = nextContainerKey
	i.hasPrevContainerKey = true

	return valid, len(valid) > 0, nil
}

var _ containerIterator = (*multiBitmapContainersIterator)(nil)

type multiBitmapContainersIterator struct {
	multiBitmapOptions

	err                error
	iters              []containerIteratorAndOp
	multiContainerIter multiBitmapContainerIterator
	first              bool
}

func newMultiBitmapContainersIterator(
	opts multiBitmapOptions,
) *multiBitmapContainersIterator {
	var (
		n     = len(opts.union) + len(opts.intersect) + len(opts.intersectNegate)
		iters = make([]containerIteratorAndOp, 0, n)
	)
	iters = appendContainerItersWithOp(iters, opts.union, multiContainerOpUnion)
	iters = appendContainerItersWithOp(iters, opts.intersect, multiContainerOpIntersect)
	iters = appendContainerItersWithOp(iters, opts.intersectNegate, multiContainerOpNegate)
	return &multiBitmapContainersIterator{
		multiBitmapOptions: opts,
	}
}

func (i *multiBitmapContainersIterator) NextContainer() bool {
	if i.err != nil || len(i.iters) != 0 {
		// Exhausted.
		return true
	}

	if i.first {
		// Always have some valid iterators since we wouldn't
		// have enqueued if not.
		i.first = false
		return true
	}

	var (
		ok  bool
		err error
	)
	i.iters, ok, err = i.multiContainerIter.resetAndReturnValid(i.iters)
	if err != nil {
		i.err = err
		return false
	}
	if !ok {
		// Exhausted.
		return false
	}

	return true
}

func (i *multiBitmapContainersIterator) ContainerKey() uint64 {
	return i.multiContainerIter.containerKey
}

func (i *multiBitmapContainersIterator) ContainerUnion(
	ctx containerOpContext,
	target *bitmapContainer,
) {
	switch i.op { // Validated at creation
	case multiBitmapOpUnion:
		// Can just blindly union into target since also a union.
		for _, iter := range i.multiContainerIter.containerIters {
			if iter.op == multiContainerOpUnion {
				iter.it.ContainerUnion(ctx, target)
			}
		}
	case multiBitmapOpIntersect:
		// Need to build intermediate and union with target.
		// Note: Cannot use ctx.tempBitmap here since downstream
		// may use it when we call iter.it.ContainerFoo(...) so
		// we use a specific intermediary here.
		tempBitmap := i.getTempIntersectAndNegate(ctx)
		defer putBitmapContainer(tempBitmap)

		unionBitmapInPlace(target.bitmap, tempBitmap.bitmap)
	}
}

func (i *multiBitmapContainersIterator) ContainerIntersect(
	ctx containerOpContext,
	target *bitmapContainer,
) {
	switch i.op { // Validated at creation
	case multiBitmapOpUnion:
		// Need to build intermediate and intersect with target.
		// Note: Cannot use ctx.tempBitmap here since downstream
		// may use it when we call iter.it.ContainerFoo(...) so
		// we use a specific intermediary here.
		tempBitmap := i.getTempUnion(ctx)
		defer putBitmapContainer(tempBitmap)

		intersectBitmapInPlace(target.bitmap, tempBitmap.bitmap)
	case multiBitmapOpIntersect:
		// Need to build intermediate and intersect with target.
		// Note: Cannot use ctx.tempBitmap here since downstream
		// may use it when we call iter.it.ContainerFoo(...) so
		// we use a specific intermediary here.
		tempBitmap := i.getTempIntersectAndNegate(ctx)
		defer putBitmapContainer(tempBitmap)

		intersectBitmapInPlace(target.bitmap, tempBitmap.bitmap)
	}
}

func (i *multiBitmapContainersIterator) ContainerNegate(
	ctx containerOpContext,
	target *bitmapContainer,
) {
	switch i.op { // Validated at creation
	case multiBitmapOpUnion:
		// Need to build intermediate and intersect with target.
		// Note: Cannot use ctx.tempBitmap here since downstream
		// may use it when we call iter.it.ContainerFoo(...) so
		// we use a specific intermediary here.
		tempBitmap := i.getTempUnion(ctx)
		defer putBitmapContainer(tempBitmap)

		differenceBitmapInPlace(target.bitmap, tempBitmap.bitmap)
	case multiBitmapOpIntersect:
		// Need to build intermediate and intersect with target.
		// Note: Cannot use ctx.tempBitmap here since downstream
		// may use it when we call iter.it.ContainerFoo(...) so
		// we use a specific intermediary here.
		tempBitmap := i.getTempIntersectAndNegate(ctx)
		defer putBitmapContainer(tempBitmap)

		differenceBitmapInPlace(target.bitmap, tempBitmap.bitmap)
	}
}

func (i *multiBitmapContainersIterator) Err() error {
	return i.err
}

func (i *multiBitmapContainersIterator) Close() {
}

func (i *multiBitmapContainersIterator) getTempUnion(
	ctx containerOpContext,
) *bitmapContainer {
	tempBitmap := getBitmapContainer()
	for _, iter := range i.multiContainerIter.containerIters {
		if iter.op == multiContainerOpUnion {
			iter.it.ContainerUnion(ctx, tempBitmap)
		}
	}
	return tempBitmap
}

func (i *multiBitmapContainersIterator) getTempIntersectAndNegate(
	ctx containerOpContext,
) *bitmapContainer {
	tempBitmap := getBitmapContainer()
	for _, iter := range i.multiContainerIter.containerIters {
		if iter.op == multiContainerOpIntersect {
			iter.it.ContainerIntersect(ctx, tempBitmap)
		}
	}
	for _, iter := range i.multiContainerIter.containerIters {
		if iter.op == multiContainerOpIntersect {
			iter.it.ContainerNegate(ctx, tempBitmap)
		}
	}
	return tempBitmap
}

// Very small isolated bitmap container pool, since in reality
// if you are looping over a lot of postings lists as long as you
// iterate each one, then progress to next they shouldn't all need
// a lot around and each bitmap is expensive.
var bitmapContainerPool = sync.Pool{
	New: func() interface{} {
		return newBitmapContainer()
	},
}

func getBitmapContainer() *bitmapContainer {
	v := bitmapContainerPool.Get().(*bitmapContainer)
	v.Reset(false)
	return v
}

func putBitmapContainer(v *bitmapContainer) {
	bitmapContainerPool.Put(v)
}

type bitmapContainer struct {
	// allocated is the allocated slice used for intermediate results.
	allocated []uint64
	// bitmap is the current bitmap, sometimes used to refer to
	// an external bitmap instead of the local allocated one.
	// NB(r): This is so if there's only a single bitmap for union
	// or intersect operation it doesn't need to copy the origin
	// bitmap to the intermediate results.
	bitmap []uint64
}

func newBitmapContainer() *bitmapContainer {
	return &bitmapContainer{allocated: make([]uint64, bitmapN)}
}

func (b *bitmapContainer) Reset(set bool) {
	if !set {
		// Make sure "0" is the default value allocated here
		// so this is compiled into a memclr optimization.
		// https://codereview.appspot.com/137880043
		for i := range b.allocated {
			b.allocated[i] = 0
		}
	} else {
		// Manually unroll loop to make it a little faster.
		for i := 0; i < bitmapN; i += 4 {
			b.allocated[i] = maxBitmap
			b.allocated[i+1] = maxBitmap
			b.allocated[i+2] = maxBitmap
			b.allocated[i+3] = maxBitmap
		}
	}

	// Always set curr to the current allocated slice.
	b.bitmap = b.allocated
}

func (b *bitmapContainer) SetReadOnly(curr []uint64) {
	// SetReadOnly should be used with care, only for single bitmap
	// iteration.
	b.bitmap = curr
}

type bitmapContainerIterator struct {
	containerKey     uint64
	bitmap           *bitmapContainer
	bitmapCurr       uint64
	bitmapCurrBase   uint64
	bitmapCurrShifts uint64
	entryIndex       int
	currValue        uint64
}

func (i *bitmapContainerIterator) Reset(
	containerKey uint64,
	bitmap *bitmapContainer,
) {
	*i = bitmapContainerIterator{}
	i.containerKey = containerKey
	i.bitmap = bitmap
	i.entryIndex = -1
}

func (i *bitmapContainerIterator) Next() bool {
	// Bitmap container.
	for i.bitmapCurr == 0 {
		// All zero bits, progress to next uint64.
		i.entryIndex++
		if i.entryIndex >= len(i.bitmap.bitmap) {
			// Exhausted.
			return false
		}

		i.bitmapCurr = i.bitmap.bitmap[i.entryIndex]
		i.bitmapCurrBase = uint64(64 * i.entryIndex)
		i.bitmapCurrShifts = 0
	}

	// Non-zero bitmap uint64, work out next bit set and add together with
	// base and current shifts made within this bitmap.
	firstBitSet := uint64(bits.TrailingZeros64(i.bitmapCurr))
	bitmapValue := i.bitmapCurrBase +
		i.bitmapCurrShifts +
		firstBitSet

	// Now shift for the next value.
	shifts := firstBitSet + 1
	i.bitmapCurr = i.bitmapCurr >> shifts
	i.bitmapCurrShifts += shifts

	i.currValue = i.containerKey<<16 | bitmapValue
	return true
}

func (i *bitmapContainerIterator) Current() uint64 {
	return i.currValue
}