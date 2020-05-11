// Generated by tmpl
// https://github.com/benbjohnson/tmpl
//
// DO NOT EDIT!
// Source: array_cursor.gen.go.tmpl

package reads

import (
	"errors"

	"github.com/influxdata/influxdb/v2/storage/reads/datatypes"
	"github.com/influxdata/influxdb/v2/tsdb/cursors"
)

const (
	// MaxPointsPerBlock is the maximum number of points in an encoded
	// block in a TSM file. It should match the value in the tsm1
	// package, but we don't want to import it.
	MaxPointsPerBlock = 1000
)

// ********************
// Float Array Cursor

type floatArrayFilterCursor struct {
	cursors.FloatArrayCursor
	cond expression
	m    *singleValue
	res  *cursors.FloatArray
	tmp  *cursors.FloatArray
}

func newFloatFilterArrayCursor(cond expression) *floatArrayFilterCursor {
	return &floatArrayFilterCursor{
		cond: cond,
		m:    &singleValue{},
		res:  cursors.NewFloatArrayLen(MaxPointsPerBlock),
		tmp:  &cursors.FloatArray{},
	}
}

func (c *floatArrayFilterCursor) reset(cur cursors.FloatArrayCursor) {
	c.FloatArrayCursor = cur
	c.tmp.Timestamps, c.tmp.Values = nil, nil
}

func (c *floatArrayFilterCursor) Stats() cursors.CursorStats { return c.FloatArrayCursor.Stats() }

func (c *floatArrayFilterCursor) Next() *cursors.FloatArray {
	pos := 0
	c.res.Timestamps = c.res.Timestamps[:cap(c.res.Timestamps)]
	c.res.Values = c.res.Values[:cap(c.res.Values)]

	var a *cursors.FloatArray

	if c.tmp.Len() > 0 {
		a = c.tmp
	} else {
		a = c.FloatArrayCursor.Next()
	}

LOOP:
	for len(a.Timestamps) > 0 {
		for i, v := range a.Values {
			c.m.v = v
			if c.cond.EvalBool(c.m) {
				c.res.Timestamps[pos] = a.Timestamps[i]
				c.res.Values[pos] = v
				pos++
				if pos >= MaxPointsPerBlock {
					c.tmp.Timestamps = a.Timestamps[i+1:]
					c.tmp.Values = a.Values[i+1:]
					break LOOP
				}
			}
		}

		// Clear bufferred timestamps & values if we make it through a cursor.
		// The break above will skip this if a cursor is partially read.
		c.tmp.Timestamps = nil
		c.tmp.Values = nil

		a = c.FloatArrayCursor.Next()
	}

	c.res.Timestamps = c.res.Timestamps[:pos]
	c.res.Values = c.res.Values[:pos]

	return c.res
}

type floatArrayCursor struct {
	cursors.FloatArrayCursor
	cursorContext
	filter *floatArrayFilterCursor
}

func (c *floatArrayCursor) reset(cur cursors.FloatArrayCursor, cursorIterator cursors.CursorIterator, cond expression) {
	if cond != nil {
		if c.filter == nil {
			c.filter = newFloatFilterArrayCursor(cond)
		}
		c.filter.reset(cur)
		cur = c.filter
	}

	c.FloatArrayCursor = cur
	c.cursorIterator = cursorIterator
	c.err = nil
}

func (c *floatArrayCursor) Err() error { return c.err }

func (c *floatArrayCursor) Stats() cursors.CursorStats {
	return c.FloatArrayCursor.Stats()
}

func (c *floatArrayCursor) Next() *cursors.FloatArray {
	for {
		a := c.FloatArrayCursor.Next()
		if a.Len() == 0 {
			if c.nextArrayCursor() {
				continue
			}
		}
		return a
	}
}

func (c *floatArrayCursor) nextArrayCursor() bool {
	if c.cursorIterator == nil {
		return false
	}

	c.FloatArrayCursor.Close()

	cur, _ := c.cursorIterator.Next(c.ctx, c.req)
	c.cursorIterator = nil

	var ok bool
	if cur != nil {
		var next cursors.FloatArrayCursor
		next, ok = cur.(cursors.FloatArrayCursor)
		if !ok {
			cur.Close()
			next = FloatEmptyArrayCursor
			c.cursorIterator = nil
			c.err = errors.New("expected float cursor")
		} else {
			if c.filter != nil {
				c.filter.reset(next)
				next = c.filter
			}
		}
		c.FloatArrayCursor = next
	} else {
		c.FloatArrayCursor = FloatEmptyArrayCursor
	}

	return ok
}

type floatArraySumCursor struct {
	cursors.FloatArrayCursor
	ts  [1]int64
	vs  [1]float64
	res *cursors.FloatArray
}

func newFloatArraySumCursor(cur cursors.FloatArrayCursor) *floatArraySumCursor {
	return &floatArraySumCursor{
		FloatArrayCursor: cur,
		res:              &cursors.FloatArray{},
	}
}

func (c floatArraySumCursor) Stats() cursors.CursorStats { return c.FloatArrayCursor.Stats() }

func (c floatArraySumCursor) Next() *cursors.FloatArray {
	a := c.FloatArrayCursor.Next()
	if len(a.Timestamps) == 0 {
		return a
	}

	ts := a.Timestamps[0]
	var acc float64

	for {
		for _, v := range a.Values {
			acc += v
		}
		a = c.FloatArrayCursor.Next()
		if len(a.Timestamps) == 0 {
			c.ts[0] = ts
			c.vs[0] = acc
			c.res.Timestamps = c.ts[:]
			c.res.Values = c.vs[:]
			return c.res
		}
	}
}

type integerFloatCountArrayCursor struct {
	cursors.FloatArrayCursor
}

func (c *integerFloatCountArrayCursor) Stats() cursors.CursorStats {
	return c.FloatArrayCursor.Stats()
}

func (c *integerFloatCountArrayCursor) Next() *cursors.IntegerArray {
	a := c.FloatArrayCursor.Next()
	if len(a.Timestamps) == 0 {
		return &cursors.IntegerArray{}
	}

	ts := a.Timestamps[0]
	var acc int64
	for {
		acc += int64(len(a.Timestamps))
		a = c.FloatArrayCursor.Next()
		if len(a.Timestamps) == 0 {
			res := cursors.NewIntegerArrayLen(1)
			res.Timestamps[0] = ts
			res.Values[0] = acc
			return res
		}
	}
}

type integerFloatWindowCountArrayCursor struct {
	cursors.FloatArrayCursor
	every int64
	tr    datatypes.TimestampRange
}

func (c *integerFloatWindowCountArrayCursor) Stats() cursors.CursorStats {
	return c.FloatArrayCursor.Stats()
}

func (c *integerFloatWindowCountArrayCursor) Next() *cursors.IntegerArray {
	a := c.FloatArrayCursor.Next()
	if a.Len() == 0 {
		return &cursors.IntegerArray{}
	}

	res := cursors.NewIntegerArrayLen(0)
	rowIdx := 0
	var acc int64 = 0

	// enumerate windows
WINDOWS:
	for {
		firstTimestamp := a.Timestamps[rowIdx]
		windowStart := firstTimestamp - firstTimestamp%c.every
		windowEnd := windowStart + c.every
		if windowEnd > c.tr.End {
			windowEnd = c.tr.End
		}
		for ; rowIdx < a.Len(); rowIdx++ {
			ts := a.Timestamps[rowIdx]
			if ts >= windowEnd {
				// new window detected, close the current window
				if acc > 0 {
					res.Timestamps = append(res.Timestamps, windowEnd)
					res.Values = append(res.Values, acc)
				}
				// start the new window
				acc = 0
				continue WINDOWS
			} else {
				acc++
			}
		}
		// get the next chunk
		a = c.FloatArrayCursor.Next()
		if a.Len() == 0 {
			if acc > 0 {
				res.Timestamps = append(res.Timestamps, windowEnd)
				res.Values = append(res.Values, acc)
			}
			break
		}
		rowIdx = 0
	}
	return res
}

type floatEmptyArrayCursor struct {
	res cursors.FloatArray
}

var FloatEmptyArrayCursor cursors.FloatArrayCursor = &floatEmptyArrayCursor{}

func (c *floatEmptyArrayCursor) Err() error                 { return nil }
func (c *floatEmptyArrayCursor) Close()                     {}
func (c *floatEmptyArrayCursor) Stats() cursors.CursorStats { return cursors.CursorStats{} }
func (c *floatEmptyArrayCursor) Next() *cursors.FloatArray  { return &c.res }

// ********************
// Integer Array Cursor

type integerArrayFilterCursor struct {
	cursors.IntegerArrayCursor
	cond expression
	m    *singleValue
	res  *cursors.IntegerArray
	tmp  *cursors.IntegerArray
}

func newIntegerFilterArrayCursor(cond expression) *integerArrayFilterCursor {
	return &integerArrayFilterCursor{
		cond: cond,
		m:    &singleValue{},
		res:  cursors.NewIntegerArrayLen(MaxPointsPerBlock),
		tmp:  &cursors.IntegerArray{},
	}
}

func (c *integerArrayFilterCursor) reset(cur cursors.IntegerArrayCursor) {
	c.IntegerArrayCursor = cur
	c.tmp.Timestamps, c.tmp.Values = nil, nil
}

func (c *integerArrayFilterCursor) Stats() cursors.CursorStats { return c.IntegerArrayCursor.Stats() }

func (c *integerArrayFilterCursor) Next() *cursors.IntegerArray {
	pos := 0
	c.res.Timestamps = c.res.Timestamps[:cap(c.res.Timestamps)]
	c.res.Values = c.res.Values[:cap(c.res.Values)]

	var a *cursors.IntegerArray

	if c.tmp.Len() > 0 {
		a = c.tmp
	} else {
		a = c.IntegerArrayCursor.Next()
	}

LOOP:
	for len(a.Timestamps) > 0 {
		for i, v := range a.Values {
			c.m.v = v
			if c.cond.EvalBool(c.m) {
				c.res.Timestamps[pos] = a.Timestamps[i]
				c.res.Values[pos] = v
				pos++
				if pos >= MaxPointsPerBlock {
					c.tmp.Timestamps = a.Timestamps[i+1:]
					c.tmp.Values = a.Values[i+1:]
					break LOOP
				}
			}
		}

		// Clear bufferred timestamps & values if we make it through a cursor.
		// The break above will skip this if a cursor is partially read.
		c.tmp.Timestamps = nil
		c.tmp.Values = nil

		a = c.IntegerArrayCursor.Next()
	}

	c.res.Timestamps = c.res.Timestamps[:pos]
	c.res.Values = c.res.Values[:pos]

	return c.res
}

type integerArrayCursor struct {
	cursors.IntegerArrayCursor
	cursorContext
	filter *integerArrayFilterCursor
}

func (c *integerArrayCursor) reset(cur cursors.IntegerArrayCursor, cursorIterator cursors.CursorIterator, cond expression) {
	if cond != nil {
		if c.filter == nil {
			c.filter = newIntegerFilterArrayCursor(cond)
		}
		c.filter.reset(cur)
		cur = c.filter
	}

	c.IntegerArrayCursor = cur
	c.cursorIterator = cursorIterator
	c.err = nil
}

func (c *integerArrayCursor) Err() error { return c.err }

func (c *integerArrayCursor) Stats() cursors.CursorStats {
	return c.IntegerArrayCursor.Stats()
}

func (c *integerArrayCursor) Next() *cursors.IntegerArray {
	for {
		a := c.IntegerArrayCursor.Next()
		if a.Len() == 0 {
			if c.nextArrayCursor() {
				continue
			}
		}
		return a
	}
}

func (c *integerArrayCursor) nextArrayCursor() bool {
	if c.cursorIterator == nil {
		return false
	}

	c.IntegerArrayCursor.Close()

	cur, _ := c.cursorIterator.Next(c.ctx, c.req)
	c.cursorIterator = nil

	var ok bool
	if cur != nil {
		var next cursors.IntegerArrayCursor
		next, ok = cur.(cursors.IntegerArrayCursor)
		if !ok {
			cur.Close()
			next = IntegerEmptyArrayCursor
			c.cursorIterator = nil
			c.err = errors.New("expected integer cursor")
		} else {
			if c.filter != nil {
				c.filter.reset(next)
				next = c.filter
			}
		}
		c.IntegerArrayCursor = next
	} else {
		c.IntegerArrayCursor = IntegerEmptyArrayCursor
	}

	return ok
}

type integerArraySumCursor struct {
	cursors.IntegerArrayCursor
	ts  [1]int64
	vs  [1]int64
	res *cursors.IntegerArray
}

func newIntegerArraySumCursor(cur cursors.IntegerArrayCursor) *integerArraySumCursor {
	return &integerArraySumCursor{
		IntegerArrayCursor: cur,
		res:                &cursors.IntegerArray{},
	}
}

func (c integerArraySumCursor) Stats() cursors.CursorStats { return c.IntegerArrayCursor.Stats() }

func (c integerArraySumCursor) Next() *cursors.IntegerArray {
	a := c.IntegerArrayCursor.Next()
	if len(a.Timestamps) == 0 {
		return a
	}

	ts := a.Timestamps[0]
	var acc int64

	for {
		for _, v := range a.Values {
			acc += v
		}
		a = c.IntegerArrayCursor.Next()
		if len(a.Timestamps) == 0 {
			c.ts[0] = ts
			c.vs[0] = acc
			c.res.Timestamps = c.ts[:]
			c.res.Values = c.vs[:]
			return c.res
		}
	}
}

type integerIntegerCountArrayCursor struct {
	cursors.IntegerArrayCursor
}

func (c *integerIntegerCountArrayCursor) Stats() cursors.CursorStats {
	return c.IntegerArrayCursor.Stats()
}

func (c *integerIntegerCountArrayCursor) Next() *cursors.IntegerArray {
	a := c.IntegerArrayCursor.Next()
	if len(a.Timestamps) == 0 {
		return &cursors.IntegerArray{}
	}

	ts := a.Timestamps[0]
	var acc int64
	for {
		acc += int64(len(a.Timestamps))
		a = c.IntegerArrayCursor.Next()
		if len(a.Timestamps) == 0 {
			res := cursors.NewIntegerArrayLen(1)
			res.Timestamps[0] = ts
			res.Values[0] = acc
			return res
		}
	}
}

type integerIntegerWindowCountArrayCursor struct {
	cursors.IntegerArrayCursor
	every int64
	tr    datatypes.TimestampRange
}

func (c *integerIntegerWindowCountArrayCursor) Stats() cursors.CursorStats {
	return c.IntegerArrayCursor.Stats()
}

func (c *integerIntegerWindowCountArrayCursor) Next() *cursors.IntegerArray {
	a := c.IntegerArrayCursor.Next()
	if a.Len() == 0 {
		return &cursors.IntegerArray{}
	}

	res := cursors.NewIntegerArrayLen(0)
	rowIdx := 0
	var acc int64 = 0

	// enumerate windows
WINDOWS:
	for {
		firstTimestamp := a.Timestamps[rowIdx]
		windowStart := firstTimestamp - firstTimestamp%c.every
		windowEnd := windowStart + c.every
		if windowEnd > c.tr.End {
			windowEnd = c.tr.End
		}
		for ; rowIdx < a.Len(); rowIdx++ {
			ts := a.Timestamps[rowIdx]
			if ts >= windowEnd {
				// new window detected, close the current window
				if acc > 0 {
					res.Timestamps = append(res.Timestamps, windowEnd)
					res.Values = append(res.Values, acc)
				}
				// start the new window
				acc = 0
				continue WINDOWS
			} else {
				acc++
			}
		}
		// get the next chunk
		a = c.IntegerArrayCursor.Next()
		if a.Len() == 0 {
			if acc > 0 {
				res.Timestamps = append(res.Timestamps, windowEnd)
				res.Values = append(res.Values, acc)
			}
			break
		}
		rowIdx = 0
	}
	return res
}

type integerEmptyArrayCursor struct {
	res cursors.IntegerArray
}

var IntegerEmptyArrayCursor cursors.IntegerArrayCursor = &integerEmptyArrayCursor{}

func (c *integerEmptyArrayCursor) Err() error                  { return nil }
func (c *integerEmptyArrayCursor) Close()                      {}
func (c *integerEmptyArrayCursor) Stats() cursors.CursorStats  { return cursors.CursorStats{} }
func (c *integerEmptyArrayCursor) Next() *cursors.IntegerArray { return &c.res }

// ********************
// Unsigned Array Cursor

type unsignedArrayFilterCursor struct {
	cursors.UnsignedArrayCursor
	cond expression
	m    *singleValue
	res  *cursors.UnsignedArray
	tmp  *cursors.UnsignedArray
}

func newUnsignedFilterArrayCursor(cond expression) *unsignedArrayFilterCursor {
	return &unsignedArrayFilterCursor{
		cond: cond,
		m:    &singleValue{},
		res:  cursors.NewUnsignedArrayLen(MaxPointsPerBlock),
		tmp:  &cursors.UnsignedArray{},
	}
}

func (c *unsignedArrayFilterCursor) reset(cur cursors.UnsignedArrayCursor) {
	c.UnsignedArrayCursor = cur
	c.tmp.Timestamps, c.tmp.Values = nil, nil
}

func (c *unsignedArrayFilterCursor) Stats() cursors.CursorStats { return c.UnsignedArrayCursor.Stats() }

func (c *unsignedArrayFilterCursor) Next() *cursors.UnsignedArray {
	pos := 0
	c.res.Timestamps = c.res.Timestamps[:cap(c.res.Timestamps)]
	c.res.Values = c.res.Values[:cap(c.res.Values)]

	var a *cursors.UnsignedArray

	if c.tmp.Len() > 0 {
		a = c.tmp
	} else {
		a = c.UnsignedArrayCursor.Next()
	}

LOOP:
	for len(a.Timestamps) > 0 {
		for i, v := range a.Values {
			c.m.v = v
			if c.cond.EvalBool(c.m) {
				c.res.Timestamps[pos] = a.Timestamps[i]
				c.res.Values[pos] = v
				pos++
				if pos >= MaxPointsPerBlock {
					c.tmp.Timestamps = a.Timestamps[i+1:]
					c.tmp.Values = a.Values[i+1:]
					break LOOP
				}
			}
		}

		// Clear bufferred timestamps & values if we make it through a cursor.
		// The break above will skip this if a cursor is partially read.
		c.tmp.Timestamps = nil
		c.tmp.Values = nil

		a = c.UnsignedArrayCursor.Next()
	}

	c.res.Timestamps = c.res.Timestamps[:pos]
	c.res.Values = c.res.Values[:pos]

	return c.res
}

type unsignedArrayCursor struct {
	cursors.UnsignedArrayCursor
	cursorContext
	filter *unsignedArrayFilterCursor
}

func (c *unsignedArrayCursor) reset(cur cursors.UnsignedArrayCursor, cursorIterator cursors.CursorIterator, cond expression) {
	if cond != nil {
		if c.filter == nil {
			c.filter = newUnsignedFilterArrayCursor(cond)
		}
		c.filter.reset(cur)
		cur = c.filter
	}

	c.UnsignedArrayCursor = cur
	c.cursorIterator = cursorIterator
	c.err = nil
}

func (c *unsignedArrayCursor) Err() error { return c.err }

func (c *unsignedArrayCursor) Stats() cursors.CursorStats {
	return c.UnsignedArrayCursor.Stats()
}

func (c *unsignedArrayCursor) Next() *cursors.UnsignedArray {
	for {
		a := c.UnsignedArrayCursor.Next()
		if a.Len() == 0 {
			if c.nextArrayCursor() {
				continue
			}
		}
		return a
	}
}

func (c *unsignedArrayCursor) nextArrayCursor() bool {
	if c.cursorIterator == nil {
		return false
	}

	c.UnsignedArrayCursor.Close()

	cur, _ := c.cursorIterator.Next(c.ctx, c.req)
	c.cursorIterator = nil

	var ok bool
	if cur != nil {
		var next cursors.UnsignedArrayCursor
		next, ok = cur.(cursors.UnsignedArrayCursor)
		if !ok {
			cur.Close()
			next = UnsignedEmptyArrayCursor
			c.cursorIterator = nil
			c.err = errors.New("expected unsigned cursor")
		} else {
			if c.filter != nil {
				c.filter.reset(next)
				next = c.filter
			}
		}
		c.UnsignedArrayCursor = next
	} else {
		c.UnsignedArrayCursor = UnsignedEmptyArrayCursor
	}

	return ok
}

type unsignedArraySumCursor struct {
	cursors.UnsignedArrayCursor
	ts  [1]int64
	vs  [1]uint64
	res *cursors.UnsignedArray
}

func newUnsignedArraySumCursor(cur cursors.UnsignedArrayCursor) *unsignedArraySumCursor {
	return &unsignedArraySumCursor{
		UnsignedArrayCursor: cur,
		res:                 &cursors.UnsignedArray{},
	}
}

func (c unsignedArraySumCursor) Stats() cursors.CursorStats { return c.UnsignedArrayCursor.Stats() }

func (c unsignedArraySumCursor) Next() *cursors.UnsignedArray {
	a := c.UnsignedArrayCursor.Next()
	if len(a.Timestamps) == 0 {
		return a
	}

	ts := a.Timestamps[0]
	var acc uint64

	for {
		for _, v := range a.Values {
			acc += v
		}
		a = c.UnsignedArrayCursor.Next()
		if len(a.Timestamps) == 0 {
			c.ts[0] = ts
			c.vs[0] = acc
			c.res.Timestamps = c.ts[:]
			c.res.Values = c.vs[:]
			return c.res
		}
	}
}

type integerUnsignedCountArrayCursor struct {
	cursors.UnsignedArrayCursor
}

func (c *integerUnsignedCountArrayCursor) Stats() cursors.CursorStats {
	return c.UnsignedArrayCursor.Stats()
}

func (c *integerUnsignedCountArrayCursor) Next() *cursors.IntegerArray {
	a := c.UnsignedArrayCursor.Next()
	if len(a.Timestamps) == 0 {
		return &cursors.IntegerArray{}
	}

	ts := a.Timestamps[0]
	var acc int64
	for {
		acc += int64(len(a.Timestamps))
		a = c.UnsignedArrayCursor.Next()
		if len(a.Timestamps) == 0 {
			res := cursors.NewIntegerArrayLen(1)
			res.Timestamps[0] = ts
			res.Values[0] = acc
			return res
		}
	}
}

type integerUnsignedWindowCountArrayCursor struct {
	cursors.UnsignedArrayCursor
	every int64
	tr    datatypes.TimestampRange
}

func (c *integerUnsignedWindowCountArrayCursor) Stats() cursors.CursorStats {
	return c.UnsignedArrayCursor.Stats()
}

func (c *integerUnsignedWindowCountArrayCursor) Next() *cursors.IntegerArray {
	a := c.UnsignedArrayCursor.Next()
	if a.Len() == 0 {
		return &cursors.IntegerArray{}
	}

	res := cursors.NewIntegerArrayLen(0)
	rowIdx := 0
	var acc int64 = 0

	// enumerate windows
WINDOWS:
	for {
		firstTimestamp := a.Timestamps[rowIdx]
		windowStart := firstTimestamp - firstTimestamp%c.every
		windowEnd := windowStart + c.every
		if windowEnd > c.tr.End {
			windowEnd = c.tr.End
		}
		for ; rowIdx < a.Len(); rowIdx++ {
			ts := a.Timestamps[rowIdx]
			if ts >= windowEnd {
				// new window detected, close the current window
				if acc > 0 {
					res.Timestamps = append(res.Timestamps, windowEnd)
					res.Values = append(res.Values, acc)
				}
				// start the new window
				acc = 0
				continue WINDOWS
			} else {
				acc++
			}
		}
		// get the next chunk
		a = c.UnsignedArrayCursor.Next()
		if a.Len() == 0 {
			if acc > 0 {
				res.Timestamps = append(res.Timestamps, windowEnd)
				res.Values = append(res.Values, acc)
			}
			break
		}
		rowIdx = 0
	}
	return res
}

type unsignedEmptyArrayCursor struct {
	res cursors.UnsignedArray
}

var UnsignedEmptyArrayCursor cursors.UnsignedArrayCursor = &unsignedEmptyArrayCursor{}

func (c *unsignedEmptyArrayCursor) Err() error                   { return nil }
func (c *unsignedEmptyArrayCursor) Close()                       {}
func (c *unsignedEmptyArrayCursor) Stats() cursors.CursorStats   { return cursors.CursorStats{} }
func (c *unsignedEmptyArrayCursor) Next() *cursors.UnsignedArray { return &c.res }

// ********************
// String Array Cursor

type stringArrayFilterCursor struct {
	cursors.StringArrayCursor
	cond expression
	m    *singleValue
	res  *cursors.StringArray
	tmp  *cursors.StringArray
}

func newStringFilterArrayCursor(cond expression) *stringArrayFilterCursor {
	return &stringArrayFilterCursor{
		cond: cond,
		m:    &singleValue{},
		res:  cursors.NewStringArrayLen(MaxPointsPerBlock),
		tmp:  &cursors.StringArray{},
	}
}

func (c *stringArrayFilterCursor) reset(cur cursors.StringArrayCursor) {
	c.StringArrayCursor = cur
	c.tmp.Timestamps, c.tmp.Values = nil, nil
}

func (c *stringArrayFilterCursor) Stats() cursors.CursorStats { return c.StringArrayCursor.Stats() }

func (c *stringArrayFilterCursor) Next() *cursors.StringArray {
	pos := 0
	c.res.Timestamps = c.res.Timestamps[:cap(c.res.Timestamps)]
	c.res.Values = c.res.Values[:cap(c.res.Values)]

	var a *cursors.StringArray

	if c.tmp.Len() > 0 {
		a = c.tmp
	} else {
		a = c.StringArrayCursor.Next()
	}

LOOP:
	for len(a.Timestamps) > 0 {
		for i, v := range a.Values {
			c.m.v = v
			if c.cond.EvalBool(c.m) {
				c.res.Timestamps[pos] = a.Timestamps[i]
				c.res.Values[pos] = v
				pos++
				if pos >= MaxPointsPerBlock {
					c.tmp.Timestamps = a.Timestamps[i+1:]
					c.tmp.Values = a.Values[i+1:]
					break LOOP
				}
			}
		}

		// Clear bufferred timestamps & values if we make it through a cursor.
		// The break above will skip this if a cursor is partially read.
		c.tmp.Timestamps = nil
		c.tmp.Values = nil

		a = c.StringArrayCursor.Next()
	}

	c.res.Timestamps = c.res.Timestamps[:pos]
	c.res.Values = c.res.Values[:pos]

	return c.res
}

type stringArrayCursor struct {
	cursors.StringArrayCursor
	cursorContext
	filter *stringArrayFilterCursor
}

func (c *stringArrayCursor) reset(cur cursors.StringArrayCursor, cursorIterator cursors.CursorIterator, cond expression) {
	if cond != nil {
		if c.filter == nil {
			c.filter = newStringFilterArrayCursor(cond)
		}
		c.filter.reset(cur)
		cur = c.filter
	}

	c.StringArrayCursor = cur
	c.cursorIterator = cursorIterator
	c.err = nil
}

func (c *stringArrayCursor) Err() error { return c.err }

func (c *stringArrayCursor) Stats() cursors.CursorStats {
	return c.StringArrayCursor.Stats()
}

func (c *stringArrayCursor) Next() *cursors.StringArray {
	for {
		a := c.StringArrayCursor.Next()
		if a.Len() == 0 {
			if c.nextArrayCursor() {
				continue
			}
		}
		return a
	}
}

func (c *stringArrayCursor) nextArrayCursor() bool {
	if c.cursorIterator == nil {
		return false
	}

	c.StringArrayCursor.Close()

	cur, _ := c.cursorIterator.Next(c.ctx, c.req)
	c.cursorIterator = nil

	var ok bool
	if cur != nil {
		var next cursors.StringArrayCursor
		next, ok = cur.(cursors.StringArrayCursor)
		if !ok {
			cur.Close()
			next = StringEmptyArrayCursor
			c.cursorIterator = nil
			c.err = errors.New("expected string cursor")
		} else {
			if c.filter != nil {
				c.filter.reset(next)
				next = c.filter
			}
		}
		c.StringArrayCursor = next
	} else {
		c.StringArrayCursor = StringEmptyArrayCursor
	}

	return ok
}

type integerStringCountArrayCursor struct {
	cursors.StringArrayCursor
}

func (c *integerStringCountArrayCursor) Stats() cursors.CursorStats {
	return c.StringArrayCursor.Stats()
}

func (c *integerStringCountArrayCursor) Next() *cursors.IntegerArray {
	a := c.StringArrayCursor.Next()
	if len(a.Timestamps) == 0 {
		return &cursors.IntegerArray{}
	}

	ts := a.Timestamps[0]
	var acc int64
	for {
		acc += int64(len(a.Timestamps))
		a = c.StringArrayCursor.Next()
		if len(a.Timestamps) == 0 {
			res := cursors.NewIntegerArrayLen(1)
			res.Timestamps[0] = ts
			res.Values[0] = acc
			return res
		}
	}
}

type integerStringWindowCountArrayCursor struct {
	cursors.StringArrayCursor
	every int64
	tr    datatypes.TimestampRange
}

func (c *integerStringWindowCountArrayCursor) Stats() cursors.CursorStats {
	return c.StringArrayCursor.Stats()
}

func (c *integerStringWindowCountArrayCursor) Next() *cursors.IntegerArray {
	a := c.StringArrayCursor.Next()
	if a.Len() == 0 {
		return &cursors.IntegerArray{}
	}

	res := cursors.NewIntegerArrayLen(0)
	rowIdx := 0
	var acc int64 = 0

	// enumerate windows
WINDOWS:
	for {
		firstTimestamp := a.Timestamps[rowIdx]
		windowStart := firstTimestamp - firstTimestamp%c.every
		windowEnd := windowStart + c.every
		if windowEnd > c.tr.End {
			windowEnd = c.tr.End
		}
		for ; rowIdx < a.Len(); rowIdx++ {
			ts := a.Timestamps[rowIdx]
			if ts >= windowEnd {
				// new window detected, close the current window
				if acc > 0 {
					res.Timestamps = append(res.Timestamps, windowEnd)
					res.Values = append(res.Values, acc)
				}
				// start the new window
				acc = 0
				continue WINDOWS
			} else {
				acc++
			}
		}
		// get the next chunk
		a = c.StringArrayCursor.Next()
		if a.Len() == 0 {
			if acc > 0 {
				res.Timestamps = append(res.Timestamps, windowEnd)
				res.Values = append(res.Values, acc)
			}
			break
		}
		rowIdx = 0
	}
	return res
}

type stringEmptyArrayCursor struct {
	res cursors.StringArray
}

var StringEmptyArrayCursor cursors.StringArrayCursor = &stringEmptyArrayCursor{}

func (c *stringEmptyArrayCursor) Err() error                 { return nil }
func (c *stringEmptyArrayCursor) Close()                     {}
func (c *stringEmptyArrayCursor) Stats() cursors.CursorStats { return cursors.CursorStats{} }
func (c *stringEmptyArrayCursor) Next() *cursors.StringArray { return &c.res }

// ********************
// Boolean Array Cursor

type booleanArrayFilterCursor struct {
	cursors.BooleanArrayCursor
	cond expression
	m    *singleValue
	res  *cursors.BooleanArray
	tmp  *cursors.BooleanArray
}

func newBooleanFilterArrayCursor(cond expression) *booleanArrayFilterCursor {
	return &booleanArrayFilterCursor{
		cond: cond,
		m:    &singleValue{},
		res:  cursors.NewBooleanArrayLen(MaxPointsPerBlock),
		tmp:  &cursors.BooleanArray{},
	}
}

func (c *booleanArrayFilterCursor) reset(cur cursors.BooleanArrayCursor) {
	c.BooleanArrayCursor = cur
	c.tmp.Timestamps, c.tmp.Values = nil, nil
}

func (c *booleanArrayFilterCursor) Stats() cursors.CursorStats { return c.BooleanArrayCursor.Stats() }

func (c *booleanArrayFilterCursor) Next() *cursors.BooleanArray {
	pos := 0
	c.res.Timestamps = c.res.Timestamps[:cap(c.res.Timestamps)]
	c.res.Values = c.res.Values[:cap(c.res.Values)]

	var a *cursors.BooleanArray

	if c.tmp.Len() > 0 {
		a = c.tmp
	} else {
		a = c.BooleanArrayCursor.Next()
	}

LOOP:
	for len(a.Timestamps) > 0 {
		for i, v := range a.Values {
			c.m.v = v
			if c.cond.EvalBool(c.m) {
				c.res.Timestamps[pos] = a.Timestamps[i]
				c.res.Values[pos] = v
				pos++
				if pos >= MaxPointsPerBlock {
					c.tmp.Timestamps = a.Timestamps[i+1:]
					c.tmp.Values = a.Values[i+1:]
					break LOOP
				}
			}
		}

		// Clear bufferred timestamps & values if we make it through a cursor.
		// The break above will skip this if a cursor is partially read.
		c.tmp.Timestamps = nil
		c.tmp.Values = nil

		a = c.BooleanArrayCursor.Next()
	}

	c.res.Timestamps = c.res.Timestamps[:pos]
	c.res.Values = c.res.Values[:pos]

	return c.res
}

type booleanArrayCursor struct {
	cursors.BooleanArrayCursor
	cursorContext
	filter *booleanArrayFilterCursor
}

func (c *booleanArrayCursor) reset(cur cursors.BooleanArrayCursor, cursorIterator cursors.CursorIterator, cond expression) {
	if cond != nil {
		if c.filter == nil {
			c.filter = newBooleanFilterArrayCursor(cond)
		}
		c.filter.reset(cur)
		cur = c.filter
	}

	c.BooleanArrayCursor = cur
	c.cursorIterator = cursorIterator
	c.err = nil
}

func (c *booleanArrayCursor) Err() error { return c.err }

func (c *booleanArrayCursor) Stats() cursors.CursorStats {
	return c.BooleanArrayCursor.Stats()
}

func (c *booleanArrayCursor) Next() *cursors.BooleanArray {
	for {
		a := c.BooleanArrayCursor.Next()
		if a.Len() == 0 {
			if c.nextArrayCursor() {
				continue
			}
		}
		return a
	}
}

func (c *booleanArrayCursor) nextArrayCursor() bool {
	if c.cursorIterator == nil {
		return false
	}

	c.BooleanArrayCursor.Close()

	cur, _ := c.cursorIterator.Next(c.ctx, c.req)
	c.cursorIterator = nil

	var ok bool
	if cur != nil {
		var next cursors.BooleanArrayCursor
		next, ok = cur.(cursors.BooleanArrayCursor)
		if !ok {
			cur.Close()
			next = BooleanEmptyArrayCursor
			c.cursorIterator = nil
			c.err = errors.New("expected boolean cursor")
		} else {
			if c.filter != nil {
				c.filter.reset(next)
				next = c.filter
			}
		}
		c.BooleanArrayCursor = next
	} else {
		c.BooleanArrayCursor = BooleanEmptyArrayCursor
	}

	return ok
}

type integerBooleanCountArrayCursor struct {
	cursors.BooleanArrayCursor
}

func (c *integerBooleanCountArrayCursor) Stats() cursors.CursorStats {
	return c.BooleanArrayCursor.Stats()
}

func (c *integerBooleanCountArrayCursor) Next() *cursors.IntegerArray {
	a := c.BooleanArrayCursor.Next()
	if len(a.Timestamps) == 0 {
		return &cursors.IntegerArray{}
	}

	ts := a.Timestamps[0]
	var acc int64
	for {
		acc += int64(len(a.Timestamps))
		a = c.BooleanArrayCursor.Next()
		if len(a.Timestamps) == 0 {
			res := cursors.NewIntegerArrayLen(1)
			res.Timestamps[0] = ts
			res.Values[0] = acc
			return res
		}
	}
}

type integerBooleanWindowCountArrayCursor struct {
	cursors.BooleanArrayCursor
	every int64
	tr    datatypes.TimestampRange
}

func (c *integerBooleanWindowCountArrayCursor) Stats() cursors.CursorStats {
	return c.BooleanArrayCursor.Stats()
}

func (c *integerBooleanWindowCountArrayCursor) Next() *cursors.IntegerArray {
	a := c.BooleanArrayCursor.Next()
	if a.Len() == 0 {
		return &cursors.IntegerArray{}
	}

	res := cursors.NewIntegerArrayLen(0)
	rowIdx := 0
	var acc int64 = 0

	// enumerate windows
WINDOWS:
	for {
		firstTimestamp := a.Timestamps[rowIdx]
		windowStart := firstTimestamp - firstTimestamp%c.every
		windowEnd := windowStart + c.every
		if windowEnd > c.tr.End {
			windowEnd = c.tr.End
		}
		for ; rowIdx < a.Len(); rowIdx++ {
			ts := a.Timestamps[rowIdx]
			if ts >= windowEnd {
				// new window detected, close the current window
				if acc > 0 {
					res.Timestamps = append(res.Timestamps, windowEnd)
					res.Values = append(res.Values, acc)
				}
				// start the new window
				acc = 0
				continue WINDOWS
			} else {
				acc++
			}
		}
		// get the next chunk
		a = c.BooleanArrayCursor.Next()
		if a.Len() == 0 {
			if acc > 0 {
				res.Timestamps = append(res.Timestamps, windowEnd)
				res.Values = append(res.Values, acc)
			}
			break
		}
		rowIdx = 0
	}
	return res
}

type booleanEmptyArrayCursor struct {
	res cursors.BooleanArray
}

var BooleanEmptyArrayCursor cursors.BooleanArrayCursor = &booleanEmptyArrayCursor{}

func (c *booleanEmptyArrayCursor) Err() error                  { return nil }
func (c *booleanEmptyArrayCursor) Close()                      {}
func (c *booleanEmptyArrayCursor) Stats() cursors.CursorStats  { return cursors.CursorStats{} }
func (c *booleanEmptyArrayCursor) Next() *cursors.BooleanArray { return &c.res }
