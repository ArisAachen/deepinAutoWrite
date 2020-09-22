package main

import (
	"testing"

	C "gopkg.in/check.v1"
)

func Test(t *testing.T) { C.TestingT(t) }

type testWrapper struct{}

func init() {
	C.Suite(testWrapper{})
}

func (*testWrapper) TestFormatImplementers(c *C.C) {
	c.Check(FormatImplementers([]string{"&Manager{}", "&ImageBlur{}", "&User{}"}), C.Equals,
		"var implementers = []dbusutil.Implementer{&Manager{},&ImageBlur{},&User{}}")
	c.Check(FormatImplementers([]string{}), C.NotNil)
	c.Check(FormatImplementers([]string{}), C.Equals, "var implementers = []dbusutil.Implementer{}")
}

func (*testWrapper) TestReplaceText(c *C.C) {
	c.Check(ReplaceText("This is a test old source", "old", "new"), C.Equals, "This is a test new source")
	c.Check(ReplaceText("This is a test not exist source", "old", "new"), C.NotNil)
	c.Check(ReplaceText("This is a test not exist source", "old", "new"), C.Not(C.Equals), "This is a test new source")
}

func (*testWrapper) TestUniqueSlice(c *C.C) {
	c.Check(UniqueSlice([]string{"the same", "the same", "the diff"}), C.DeepEquals, []string{"the same", "the diff"})
	c.Check(UniqueSlice([]string{"the same", "the same", "the diff"}), C.Not(C.DeepEquals), []string{"the same", "the same", "the diff"})
	c.Check(UniqueSlice([]string{"the unique", "the diff"}), C.DeepEquals, []string{"the unique", "the diff"})
}
