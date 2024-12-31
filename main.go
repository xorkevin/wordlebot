package main

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"errors"
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

	target, err := ParseWord("mambo")
	if err != nil {
		log.Fatalln(err)
	}
	SimulateGame(target, words)
}

const (
	allBits = 0x3ffffff
)

func SimulateGame(target WordleWord, words []WordleWord) {
	universe := WordleWord{allBits, allBits, allBits, allBits, allBits}
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			log.Fatalln("Failed reading input")
		}
		line = strings.TrimSpace(line)
		guess, err := ParseWord(line)
		if err != nil {
			log.Println(err)
			continue
		}
		pattern := target.ComputePattern(guess)
		universe = universe.Filter(pattern)
		fmt.Println(pattern)
		fmt.Println(universe.StringMask())
	}
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

func (w WordleWord) StringMask() string {
	return fmt.Sprintf("%026b,%026b,%026b,%026b,%026b", w[0], w[1], w[2], w[3], w[4])
}

func (w WordleWord) Or(other WordleWord) WordleWord {
	for i := range w {
		w[i] |= other[i]
	}
	return w
}

func (w WordleWord) And(other WordleWord) WordleWord {
	for i := range w {
		w[i] &= other[i]
	}
	return w
}

func (w WordleWord) Match(other WordleWord) bool {
	for i, v := range w {
		if v&other[i] != other[i] {
			return false
		}
	}
	return true
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
		if other[i] == v {
			pattern[i] = WordlePatternLetter{
				v:    v,
				kind: PatternKindG,
			}
		} else if (other[i] & fullset) != 0 {
			pattern[i] = WordlePatternLetter{
				v:    v,
				kind: PatternKindY,
			}
		} else {
			pattern[i] = WordlePatternLetter{
				v:    v,
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
