package types

import (
	. "github.com/onsi/gomega"
	"testing"
)

func TestNewZone(t *testing.T) {

}

func TestZone_FindLocation(t *testing.T) {
	RegisterTestingT(t)
	zone := NewZone(
		"zone.com.",
		[]string{
			"*",
		},
		"{}",
	)

	// root exact match
	label, matchType := zone.FindLocation("zone.com.")
	Expect(label).To(Equal("@"))
	Expect(matchType).To(Equal(ExactMatch))

	// no prefix, root wildcard match
	label, matchType = zone.FindLocation("a.zone.com.")
	Expect(label).To(Equal("*"))
	Expect(matchType).To(Equal(WildCardMatch))

	zone = NewZone(
		"zone.com.",
		[]string{
			"a.b.c.d",
			"k.l.m",
			"*.k.l.m",
			"n.o.p",
			"*.s",
			"t.u.v",
		},
		"",
	)

	// no prefix, no match
	label, matchType = zone.FindLocation("x.y.z.zone.com.")
	Expect(label).To(Equal(""))
	Expect(matchType).To(Equal(NoMatch))

	// label exact match
	label, matchType = zone.FindLocation("a.b.c.d.zone.com.")
	Expect(label).To(Equal("a.b.c.d"))
	Expect(matchType).To(Equal(ExactMatch))

	// non-empty ce, wildcard match
	label, matchType = zone.FindLocation("x.k.l.m.zone.com.")
	Expect(label).To(Equal("*.k.l.m"))
	Expect(matchType).To(Equal(WildCardMatch))

	// ce match
	label, matchType = zone.FindLocation("x.n.o.p.zone.com.")
	Expect(label).To(Equal("n.o.p"))
	Expect(matchType).To(Equal(CEMatch))

	// empty non-terminal match
	label, matchType = zone.FindLocation("c.d.zone.com.")
	Expect(label).To(Equal(""))
	Expect(matchType).To(Equal(EmptyNonterminalMatch))

	// empty ce, wildcard match
	label, matchType = zone.FindLocation("x.s.zone.com.")
	Expect(label).To(Equal("*.s"))
	Expect(matchType).To(Equal(WildCardMatch))

	// empty ce, no match
	label, matchType = zone.FindLocation("x.u.v.zone.com.")
	Expect(label).To(Equal(""))
	Expect(matchType).To(Equal(NoMatch))
}
