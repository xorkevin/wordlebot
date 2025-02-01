package main

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/bits"
	"os"
	"strings"
)

var (
	ErrWordLen  = errors.New("Error word length")
	ErrWordChar = errors.New("Error word char")
)

//go:embed wordlist.json
var wordlist []byte

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var strWords []string
	if err := json.Unmarshal(wordlist, &strWords); err != nil {
		log.Fatalln(err)
	}

	words := make([]WordleWord, 0, len(strWords))
	for _, i := range strWords {
		w, err := ParseWord(i)
		if err != nil {
			log.Fatalln(err)
		}
		words = append(words, w)
	}

	var targetWord string
	flag.StringVar(&targetWord, "target", "", "target word")

	flag.Parse()

	target, err := ParseWord(targetWord)
	if err != nil {
		log.Fatalln(err)
	}
	SimulateGame(target, words)
}

const (
	allBits = 0x3ffffff
)

func SimulateGame(target WordleWord, words []WordleWord) {
	universe := Universe{
		bitMask: WordleWord{allBits, allBits, allBits, allBits, allBits},
	}
	numPossibilities := len(words)
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Guess: ")
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			log.Fatalln("Failed reading input")
		}
		line = strings.TrimSpace(line)
		if line == "p" {
			for _, v := range words {
				if universe.Contains(v) {
					fmt.Println(v)
				}
			}
			continue
		}
		guess, err := ParseWord(line)
		if err != nil {
			log.Println(err)
			continue
		}
		universe, numPossibilities = CondenseUniverse(guess, target, universe, words)
		fmt.Printf("Pattern %s solution charset %026b eliminated charset %026b\n", target.ComputePattern(guess), universe.solutionChars, universe.eliminatedChars)
		fmt.Println("universe", universe.bitMask.StringMask())
		fmt.Println(numPossibilities, "possibilities")
		if numPossibilities < 2 {
			for _, v := range words {
				if universe.Contains(v) {
					fmt.Println(v)
					break
				}
			}
			break
		}
	}
}

type (
	Universe struct {
		bitMask                        WordleWord
		solutionChars, eliminatedChars uint32
	}
)

func CondenseUniverse(guess, target WordleWord, universe Universe, words []WordleWord) (Universe, int) {
	pattern := target.ComputePattern(guess)
	for _, v := range pattern {
		switch v.kind {
		case PatternKindB:
			universe.eliminatedChars |= v.v
		case PatternKindY, PatternKindG:
			universe.solutionChars |= v.v
		}
	}
	universe.bitMask = universe.bitMask.Filter(pattern)
	count := 0
	var condensed WordleWord
	for _, v := range words {
		if universe.Contains(v) {
			condensed = condensed.Or(v)
			count++
		}
	}
	universe.bitMask = condensed
	return universe, count
}

func (u Universe) Contains(v WordleWord) bool {
	vc := v.CharSet()
	return u.bitMask.Match(v) && vc&u.solutionChars == u.solutionChars && vc&u.eliminatedChars == 0
}

type (
	WordleWord [5]uint32

	PatternKind byte

	WordlePatternLetter struct {
		v    uint32
		kind PatternKind
	}

	WordlePattern [5]WordlePatternLetter
)

const (
	PatternKindB PatternKind = iota
	PatternKindY
	PatternKindG
)

func (w WordleWord) String() string {
	var b strings.Builder
	for _, v := range w {
		b.WriteByte(byte(bits.TrailingZeros32(v)) + 'A')
	}
	return b.String()
}

func (w WordleWord) StringMask() string {
	return fmt.Sprintf("%026b,%026b,%026b,%026b,%026b", w[0], w[1], w[2], w[3], w[4])
}

func (w WordleWord) Or(other WordleWord) WordleWord {
	return WordleWord{
		w[0] | other[0],
		w[1] | other[1],
		w[2] | other[2],
		w[3] | other[3],
		w[4] | other[4],
	}
}

func (w WordleWord) And(other WordleWord) WordleWord {
	return WordleWord{
		w[0] & other[0],
		w[1] & other[1],
		w[2] & other[2],
		w[3] & other[3],
		w[4] & other[4],
	}
}

func (w WordleWord) Match(other WordleWord) bool {
	return w.And(other) == other
}

func (w WordleWord) CharSet() uint32 {
	return w[0] | w[1] | w[2] | w[3] | w[4]
}

func (w WordleWord) Filter(pattern WordlePattern) WordleWord {
	for i, v := range pattern {
		switch v.kind {
		case PatternKindB:
			var mask uint32 = ^v.v
			w = w.And(WordleWord{mask, mask, mask, mask, mask})
		case PatternKindY:
			var mask uint32 = ^v.v
			w[i] &= mask
		case PatternKindG:
			var mask uint32 = v.v
			w[i] = mask
		}
	}
	return w
}

func (w WordleWord) ComputePattern(other WordleWord) WordlePattern {
	var fullset uint32
	for _, v := range w {
		fullset |= v
	}
	var pattern WordlePattern
	for i, v := range w {
		c := other[i]
		if c == v {
			pattern[i] = WordlePatternLetter{
				v:    c,
				kind: PatternKindG,
			}
		} else if (c & fullset) != 0 {
			pattern[i] = WordlePatternLetter{
				v:    c,
				kind: PatternKindY,
			}
		} else {
			pattern[i] = WordlePatternLetter{
				v:    c,
				kind: PatternKindB,
			}
		}
	}
	return pattern
}

func (p WordlePattern) String() string {
	var b strings.Builder
	for i, v := range p {
		if i != 0 {
			b.WriteByte(' ')
		}
		b.WriteByte(byte(bits.TrailingZeros32(v.v)) + 'A')
		b.WriteByte(':')
		switch v.kind {
		case PatternKindB:
			b.WriteByte('B')
		case PatternKindY:
			b.WriteByte('Y')
		case PatternKindG:
			b.WriteByte('G')
		}
	}
	return b.String()
}

func ParseWord(s string) (WordleWord, error) {
	if len(s) != 5 {
		return WordleWord{}, ErrWordLen
	}
	s = strings.ToUpper(s)
	var w WordleWord
	for i := range w {
		c := s[i] - 'A'
		if c > 'Z' {
			return w, ErrWordChar
		}
		w[i] = 1 << c
	}
	return w, nil
}

type (
	BitSet struct {
		bits []uint64
		size int
	}
)

func NewBitSet(size int) *BitSet {
	return &BitSet{
		bits: make([]uint64, (size+63)/64),
		size: 0,
	}
}

func (s *BitSet) Reset() {
	for i := range s.bits {
		s.bits[i] = 0
	}
	s.size = 0
}

func (s *BitSet) Size() int {
	return s.size
}

func (s *BitSet) Contains(i int) bool {
	a := i / 64
	mask := uint64(1) << (i % 64)
	return (s.bits[a] & mask) != 0
}

func (s *BitSet) Set(i int, b bool) bool {
	a := i / 64
	mask := uint64(1) << (i % 64)
	if b {
		diff := (s.bits[a] & mask) == 0
		s.bits[a] |= mask
		if diff {
			s.size++
		}
		return diff
	} else {
		diff := (s.bits[a] & mask) != 0
		s.bits[a] &^= mask
		if diff {
			s.size--
		}
		return diff
	}
}

func (s *BitSet) Insert(i int) bool {
	a := i / 64
	mask := uint64(1) << (i % 64)
	diff := (s.bits[a] & mask) == 0
	s.bits[a] |= mask
	if diff {
		s.size++
	}
	return diff
}

func (s *BitSet) Remove(i int) bool {
	a := i / 64
	mask := uint64(1) << (i % 64)
	diff := (s.bits[a] & mask) != 0
	s.bits[a] &^= mask
	if diff {
		s.size--
	}
	return diff
}
