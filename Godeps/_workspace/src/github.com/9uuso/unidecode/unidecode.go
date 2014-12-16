// Package unidecode implements a unicode transliterator
// which replaces non-ASCII characters with their ASCII
// approximations.
package unidecode

import (
	"compress/zlib"
	"encoding/binary"
	"io"
	"strings"
	"sync"
	"unicode"

	"gopkgs.com/pool.v1"
)

const pooledCapacity = 64

var (
	mutex            sync.Mutex
	transliterations [65536][]rune

	slicePool  = pool.New(0)
	decoded    = false
	transCount = rune(len(transliterations))
	getUint16  = binary.LittleEndian.Uint16
)

func decodeTransliterations() {
	r, err := zlib.NewReader(strings.NewReader(tableData))
	if err != nil {
		panic(err)
	}
	defer r.Close()
	tmp1 := make([]byte, 2)
	tmp2 := tmp1[:1]
	for {
		if _, err := io.ReadAtLeast(r, tmp1, 2); err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		chr := getUint16(tmp1)
		if _, err := io.ReadAtLeast(r, tmp2, 1); err != nil {
			panic(err)
		}
		b := make([]byte, int(tmp2[0]))
		if _, err := io.ReadFull(r, b); err != nil {
			panic(err)
		}
		transliterations[int(chr)] = []rune(string(b))
	}
}

// Unidecode implements a unicode transliterator, which
// replaces non-ASCII characters with their ASCII
// counterparts.
// Given an unicode encoded string, returns
// another string with non-ASCII characters replaced
// with their closest ASCII counterparts.
// e.g. Unicode("áéíóú") => "aeiou"
func Unidecode(s string) string {
	if !decoded {
		mutex.Lock()
		if !decoded {
			decodeTransliterations()
			decoded = true
		}
		mutex.Unlock()
	}
	l := len(s)
	var r []rune
	if l > pooledCapacity {
		r = make([]rune, 0, len(s))
	} else {
		if x := slicePool.Get(); x != nil {
			r = x.([]rune)[:0]
		} else {
			r = make([]rune, 0, pooledCapacity)
		}
	}
	for _, c := range s {
		if c <= unicode.MaxASCII {
			r = append(r, c)
			continue
		}
		if c > unicode.MaxRune || c > transCount {
			/* Ignore reserved chars */
			continue
		}
		if d := transliterations[c]; d != nil {
			r = append(r, d...)
		}
	}
	res := string(r)
	if l <= pooledCapacity {
		slicePool.Put(r)
	}
	return res
}
